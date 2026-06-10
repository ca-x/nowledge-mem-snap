package storage

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/webdav"
)

type webDAVHTTPClient struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

func (c *webDAVHTTPClient) do(ctx context.Context, method, name string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.urlFor(name), body)
	if err != nil {
		return nil, err
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	return c.client.Do(req)
}

func (c *webDAVHTTPClient) doWithHeaders(ctx context.Context, method, name string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.urlFor(name), body)
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	return c.client.Do(req)
}

func (c *webDAVHTTPClient) urlFor(name string) string {
	parts := strings.Split(strings.Trim(cleanWebDAVName(name), "/"), "/")
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(part))
	}
	if len(escaped) == 0 {
		return c.baseURL + "/"
	}
	return c.baseURL + "/" + strings.Join(escaped, "/")
}

type webDAVFileSystem struct {
	client *webDAVHTTPClient
}

func (fs webDAVFileSystem) Mkdir(ctx context.Context, name string, _ os.FileMode) error {
	resp, err := fs.client.do(ctx, "MKCOL", name, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusMethodNotAllowed {
		return nil
	}
	return statusError("MKCOL", name, resp)
}

func (fs webDAVFileSystem) OpenFile(ctx context.Context, name string, flag int, _ os.FileMode) (webdav.File, error) {
	name = cleanWebDAVName(name)
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC) != 0 {
		return &webDAVWriteFile{ctx: ctx, fs: fs, name: name, buf: bytes.NewBuffer(nil)}, nil
	}
	data, info, err := fs.read(ctx, name)
	if err != nil {
		return nil, err
	}
	return newWebDAVReadFile(name, data, info), nil
}

func (fs webDAVFileSystem) RemoveAll(ctx context.Context, name string) error {
	resp, err := fs.client.do(ctx, http.MethodDelete, name, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return statusError("DELETE", name, resp)
}

func (fs webDAVFileSystem) Rename(ctx context.Context, oldName, newName string) error {
	req, err := http.NewRequestWithContext(ctx, "MOVE", fs.client.urlFor(oldName), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Destination", fs.client.urlFor(newName))
	req.Header.Set("Overwrite", "T")
	if fs.client.username != "" || fs.client.password != "" {
		req.SetBasicAuth(fs.client.username, fs.client.password)
	}
	resp, err := fs.client.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return nil
	}
	return statusError("MOVE", oldName, resp)
}

func (fs webDAVFileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	infos, err := fs.propfind(ctx, name, 0)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, os.ErrNotExist
	}
	return infos[0], nil
}

func (fs webDAVFileSystem) read(ctx context.Context, name string) ([]byte, os.FileInfo, error) {
	resp, err := fs.client.do(ctx, http.MethodGet, name, nil)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil, os.ErrNotExist
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, statusError("GET", name, resp)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	info := webDAVInfo{
		name:    path.Base(cleanWebDAVName(name)),
		size:    int64(len(data)),
		mode:    0644,
		modTime: time.Now(),
	}
	return data, info, nil
}

func (fs webDAVFileSystem) write(ctx context.Context, name string, data []byte) error {
	resp, err := fs.client.do(ctx, http.MethodPut, name, bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return nil
	}
	return statusError("PUT", name, resp)
}

func (fs webDAVFileSystem) propfind(ctx context.Context, name string, depth int) ([]webDAVInfo, error) {
	resp, err := fs.client.doWithHeaders(ctx, "PROPFIND", name, strings.NewReader(`<?xml version="1.0"?><propfind xmlns="DAV:"><allprop/></propfind>`), map[string]string{
		"Depth":        strconv.Itoa(depth),
		"Content-Type": "application/xml; charset=utf-8",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, os.ErrNotExist
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, statusError("PROPFIND", name, resp)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var multistatus propMultiStatus
	if err := xml.Unmarshal(data, &multistatus); err != nil {
		return nil, err
	}
	infos := make([]webDAVInfo, 0, len(multistatus.Responses))
	for _, resp := range multistatus.Responses {
		info := webDAVInfo{name: path.Base(strings.TrimRight(resp.Href, "/")), mode: 0644, modTime: time.Now()}
		for _, propstat := range resp.PropStats {
			if propstat.Prop.ResourceType.Collection != nil {
				info.mode = os.ModeDir | 0755
			}
			if propstat.Prop.ContentLength != "" {
				if size, err := strconv.ParseInt(strings.TrimSpace(propstat.Prop.ContentLength), 10, 64); err == nil {
					info.size = size
				}
			}
			if propstat.Prop.GetLastModified != "" {
				if t, err := http.ParseTime(propstat.Prop.GetLastModified); err == nil {
					info.modTime = t
				}
			}
			if propstat.Prop.DisplayName != "" {
				info.name = propstat.Prop.DisplayName
			}
		}
		if info.name == "" || info.name == "." || info.name == "/" {
			info.name = path.Base(cleanWebDAVName(name))
		}
		infos = append(infos, info)
	}
	return infos, nil
}

type webDAVReadFile struct {
	*bytes.Reader
	name string
	info os.FileInfo
}

func newWebDAVReadFile(name string, data []byte, info os.FileInfo) *webDAVReadFile {
	return &webDAVReadFile{Reader: bytes.NewReader(data), name: name, info: info}
}

func (f *webDAVReadFile) Close() error                       { return nil }
func (f *webDAVReadFile) Readdir(int) ([]os.FileInfo, error) { return nil, os.ErrInvalid }
func (f *webDAVReadFile) Stat() (os.FileInfo, error)         { return f.info, nil }
func (f *webDAVReadFile) Write([]byte) (int, error)          { return 0, os.ErrPermission }

type webDAVWriteFile struct {
	ctx    context.Context
	fs     webDAVFileSystem
	name   string
	buf    *bytes.Buffer
	closed bool
}

func (f *webDAVWriteFile) Close() error {
	if f.closed {
		return nil
	}
	f.closed = true
	return f.fs.write(f.ctx, f.name, f.buf.Bytes())
}

func (f *webDAVWriteFile) Read([]byte) (int, error)           { return 0, os.ErrPermission }
func (f *webDAVWriteFile) Seek(int64, int) (int64, error)     { return 0, os.ErrInvalid }
func (f *webDAVWriteFile) Readdir(int) ([]os.FileInfo, error) { return nil, os.ErrInvalid }
func (f *webDAVWriteFile) Stat() (os.FileInfo, error) {
	return webDAVInfo{name: path.Base(f.name), size: int64(f.buf.Len()), mode: 0644, modTime: time.Now()}, nil
}
func (f *webDAVWriteFile) Write(p []byte) (int, error) { return f.buf.Write(p) }

type webDAVInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (i webDAVInfo) Name() string       { return i.name }
func (i webDAVInfo) Size() int64        { return i.size }
func (i webDAVInfo) Mode() os.FileMode  { return i.mode }
func (i webDAVInfo) ModTime() time.Time { return i.modTime }
func (i webDAVInfo) IsDir() bool        { return i.mode.IsDir() }
func (i webDAVInfo) Sys() any           { return nil }

type propMultiStatus struct {
	Responses []propResponse `xml:"response"`
}

type propResponse struct {
	Href      string     `xml:"href"`
	PropStats []propStat `xml:"propstat"`
}

type propStat struct {
	Prop prop `xml:"prop"`
}

type prop struct {
	DisplayName     string       `xml:"displayname"`
	ContentLength   string       `xml:"getcontentlength"`
	GetLastModified string       `xml:"getlastmodified"`
	ResourceType    resourceType `xml:"resourcetype"`
}

type resourceType struct {
	Collection *struct{} `xml:"collection"`
}

func cleanWebDAVName(raw string) string {
	raw = strings.ReplaceAll(raw, "\\", "/")
	cleaned := path.Clean("/" + strings.TrimLeft(raw, "/"))
	if cleaned == "." {
		return "/"
	}
	return cleaned
}

func statusError(op, name string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("%s %s failed: %s %s", op, name, resp.Status, strings.TrimSpace(string(body)))
}
