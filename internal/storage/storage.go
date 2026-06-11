package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	aferos3 "github.com/fclairamb/afero-s3"
	"github.com/lib-x/aferodav"
	"github.com/spf13/afero"

	"github.com/ca-x/nowledge-mem-snap/internal/archive"
	"github.com/ca-x/nowledge-mem-snap/internal/config"
)

type Target struct {
	Key  string
	Name string
	Fs   afero.Fs
}

type BackupObject struct {
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_bytes"`
	ModTime   time.Time `json:"modified_at"`
	Encrypted bool      `json:"encrypted"`
}

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) Target(ctx context.Context, target config.TargetConfig) (Target, error) {
	if !target.Enabled {
		return Target{}, fmt.Errorf("target %q is disabled", target.Key)
	}
	var fs afero.Fs
	var err error
	switch target.Type {
	case "s3":
		fs, err = newS3FS(target.S3)
	case "webdav":
		fs, err = newWebDAVFS(ctx, target.WebDAV)
	default:
		err = fmt.Errorf("unsupported target type %q", target.Type)
	}
	if err != nil {
		return Target{}, err
	}
	return Target{Key: target.Key, Name: target.Name, Fs: fs}, nil
}

func Write(ctx context.Context, target Target, objectName string, data []byte) (int64, error) {
	objectName, err := cleanObjectPath(objectName)
	if err != nil {
		return 0, err
	}
	dir := path.Dir(objectName)
	if dir != "." && dir != "/" {
		if err := target.Fs.MkdirAll(dir, 0755); err != nil {
			return 0, fmt.Errorf("create parent directory: %w", err)
		}
	}
	file, err := target.Fs.OpenFile(objectName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return 0, fmt.Errorf("open remote object: %w", err)
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = file.Close()
		case <-done:
		}
	}()
	n, copyErr := file.Write(data)
	close(done)
	closeErr := file.Close()
	if copyErr != nil {
		return int64(n), fmt.Errorf("write remote object: %w", copyErr)
	}
	if closeErr != nil {
		return int64(n), fmt.Errorf("close remote object: %w", closeErr)
	}
	if n != len(data) {
		return int64(n), io.ErrShortWrite
	}
	return int64(n), nil
}

func Read(ctx context.Context, target Target, objectName string) ([]byte, error) {
	objectName, err := cleanObjectPath(objectName)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	file, err := target.Fs.Open(objectName)
	if err != nil {
		return nil, fmt.Errorf("open remote object: %w", err)
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = file.Close()
		case <-done:
		}
	}()
	data, readErr := io.ReadAll(file)
	close(done)
	closeErr := file.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read remote object: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close remote object: %w", closeErr)
	}
	return data, nil
}

func ListBackupObjects(ctx context.Context, target Target, prefix string) ([]BackupObject, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, fmt.Errorf("object prefix is required")
	}
	scope, err := cleanObjectPath(prefix)
	if err != nil {
		return nil, err
	}
	if scope == "" || scope == "." || scope == "/" {
		return nil, fmt.Errorf("object prefix must not resolve to storage root")
	}
	objects, err := listBackupObjects(ctx, target.Fs, scope)
	if os.IsNotExist(err) {
		return []BackupObject{}, nil
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(objects, func(i, j int) bool {
		if objects[i].ModTime.Equal(objects[j].ModTime) {
			return objects[i].Name > objects[j].Name
		}
		return objects[i].ModTime.After(objects[j].ModTime)
	})
	return objects, nil
}

func ApplyRetention(ctx context.Context, target Target, task config.TaskConfig, now time.Time) (int, error) {
	retention := task.Retention
	if retention.Mode == "" || retention.Mode == "none" {
		return 0, nil
	}
	scope := archive.RetentionDirectory(task.ObjectPrefix, task.Key, task.Name)
	if scope == "" || scope == "." || scope == "/" {
		return 0, fmt.Errorf("retention requires object_prefix to contain a stable directory before date or timestamp tokens")
	}
	objects, err := listBackupObjects(ctx, target.Fs, scope)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	sort.Slice(objects, func(i, j int) bool {
		if objects[i].ModTime.Equal(objects[j].ModTime) {
			return objects[i].Name > objects[j].Name
		}
		return objects[i].ModTime.After(objects[j].ModTime)
	})

	cut := make(map[string]struct{})
	loc := time.Local
	switch retention.Mode {
	case "keep_last":
		for i, object := range objects {
			if i >= retention.KeepLast {
				cut[object.Name] = struct{}{}
			}
		}
	case "keep_days":
		cutoff := now.In(loc).AddDate(0, 0, -retention.KeepDays)
		for _, object := range objects {
			if object.ModTime.Before(cutoff) {
				cut[object.Name] = struct{}{}
			}
		}
	case "keep_after":
		cutoff, err := parseRetentionDate(retention.KeepAfter, loc)
		if err != nil {
			return 0, err
		}
		for _, object := range objects {
			if object.ModTime.Before(cutoff) {
				cut[object.Name] = struct{}{}
			}
		}
	case "keep_before":
		cutoff, err := parseRetentionDate(retention.KeepBefore, loc)
		if err != nil {
			return 0, err
		}
		for _, object := range objects {
			if !object.ModTime.Before(cutoff) {
				cut[object.Name] = struct{}{}
			}
		}
	default:
		return 0, fmt.Errorf("unsupported retention mode %q", retention.Mode)
	}

	deleted := 0
	for _, object := range objects {
		if _, ok := cut[object.Name]; !ok {
			continue
		}
		select {
		case <-ctx.Done():
			return deleted, ctx.Err()
		default:
		}
		if err := target.Fs.Remove(object.Name); err != nil && !os.IsNotExist(err) {
			return deleted, fmt.Errorf("delete %s: %w", object.Name, err)
		}
		deleted++
	}
	return deleted, nil
}

func listBackupObjects(ctx context.Context, fs afero.Fs, scope string) ([]BackupObject, error) {
	objects := []BackupObject{}
	err := afero.Walk(fs, scope, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if info == nil || info.IsDir() {
			return nil
		}
		if !isBackupObject(name) {
			return nil
		}
		objects = append(objects, BackupObject{
			Name:      name,
			SizeBytes: info.Size(),
			ModTime:   info.ModTime().UTC(),
			Encrypted: strings.HasSuffix(name, archive.EncryptedExtension),
		})
		return nil
	})
	return objects, err
}

func isBackupObject(name string) bool {
	return strings.HasSuffix(name, archive.PlainExtension) || strings.HasSuffix(name, archive.EncryptedExtension)
}

func parseRetentionDate(raw string, loc *time.Location) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("retention date is required")
	}
	if loc == nil {
		loc = time.Local
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", raw, loc); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("retention date must use YYYY-MM-DD or RFC3339")
}

func newS3FS(cfg config.S3Config) (afero.Fs, error) {
	client := newS3Client(cfg)
	fs := aferos3.NewFsFromClient(cfg.BucketName, client)
	return prefixFS{Fs: fs, prefix: cfg.RootPrefix}, nil
}

func newS3Client(cfg config.S3Config) *awss3.Client {
	awsCfg := aws.Config{
		Region: cfg.Region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:       "aws",
				URL:               cfg.EndpointURL,
				SigningRegion:     cfg.Region,
				HostnameImmutable: true,
			}, nil
		}),
	}
	client := awss3.NewFromConfig(awsCfg, func(options *awss3.Options) {
		options.UsePathStyle = cfg.PathStyle
	})
	return client
}

func newWebDAVFS(ctx context.Context, cfg config.WebDAVConfig) (afero.Fs, error) {
	client, err := newWebDAVClient(cfg)
	if err != nil {
		return nil, err
	}
	return prefixFS{Fs: aferodav.New(webDAVFileSystem{client: client}, ctx), prefix: cfg.RootPrefix}, nil
}

func newWebDAVClient(cfg config.WebDAVConfig) (*webDAVHTTPClient, error) {
	baseURL, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse WebDAV URL: %w", err)
	}
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, fmt.Errorf("WebDAV URL must use http or https")
	}
	client := &webDAVHTTPClient{
		baseURL:  strings.TrimRight(baseURL.String(), "/"),
		username: cfg.Username,
		password: cfg.Password,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
	return client, nil
}

type prefixFS struct {
	afero.Fs
	prefix string
}

func (fs prefixFS) withPrefix(name string) string {
	cleaned, _ := cleanObjectPath(name)
	prefix := strings.Trim(fs.prefix, "/")
	if prefix == "" {
		return cleaned
	}
	return path.Join(prefix, cleaned)
}

func (fs prefixFS) Create(name string) (afero.File, error) {
	return fs.Fs.Create(fs.withPrefix(name))
}

func (fs prefixFS) Mkdir(name string, perm os.FileMode) error {
	return fs.Fs.Mkdir(fs.withPrefix(name), perm)
}

func (fs prefixFS) MkdirAll(name string, perm os.FileMode) error {
	return fs.Fs.MkdirAll(fs.withPrefix(name), perm)
}

func (fs prefixFS) Open(name string) (afero.File, error) {
	return fs.Fs.Open(fs.withPrefix(name))
}

func (fs prefixFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return fs.Fs.OpenFile(fs.withPrefix(name), flag, perm)
}

func (fs prefixFS) Remove(name string) error {
	return fs.Fs.Remove(fs.withPrefix(name))
}

func (fs prefixFS) RemoveAll(name string) error {
	return fs.Fs.RemoveAll(fs.withPrefix(name))
}

func (fs prefixFS) Rename(oldname, newname string) error {
	return fs.Fs.Rename(fs.withPrefix(oldname), fs.withPrefix(newname))
}

func (fs prefixFS) Stat(name string) (os.FileInfo, error) {
	return fs.Fs.Stat(fs.withPrefix(name))
}

func (fs prefixFS) Chmod(name string, mode os.FileMode) error {
	return fs.Fs.Chmod(fs.withPrefix(name), mode)
}

func (fs prefixFS) Chown(name string, uid, gid int) error {
	return fs.Fs.Chown(fs.withPrefix(name), uid, gid)
}

func (fs prefixFS) Chtimes(name string, atime, mtime time.Time) error {
	return fs.Fs.Chtimes(fs.withPrefix(name), atime, mtime)
}

func cleanObjectPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("object path is required")
	}
	if strings.Contains(raw, "\x00") {
		return "", fmt.Errorf("object path contains invalid character")
	}
	if strings.Contains(raw, "\\") {
		return "", fmt.Errorf("object path must use / separators")
	}
	for _, part := range strings.Split(raw, "/") {
		if part == ".." {
			return "", fmt.Errorf("object path cannot contain ..")
		}
	}
	cleaned := path.Clean("/" + strings.TrimPrefix(raw, "/"))
	return strings.TrimPrefix(cleaned, "/"), nil
}
