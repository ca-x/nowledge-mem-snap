package source

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	mem "github.com/lib-x/nowledgemem-go"
	"github.com/spf13/afero"

	"github.com/lib-x/nowledge-mem-snap/internal/config"
)

type Snapshot struct {
	Data      []byte
	SizeBytes int64
	ItemCount int
}

type Exporter struct{}

func NewExporter() *Exporter {
	return &Exporter{}
}

func (e *Exporter) Export(ctx context.Context, cfg config.Config, task config.TaskConfig) (Snapshot, error) {
	sourceCfg, ok := cfg.Source(task.SourceKey)
	if !ok {
		return Snapshot{}, fmt.Errorf("source %q was not found", task.SourceKey)
	}
	if !sourceCfg.Enabled {
		return Snapshot{}, fmt.Errorf("source %q is disabled", sourceCfg.Key)
	}
	switch sourceCfg.Type {
	case "nowledgemem_api":
		exportOption, ok := cfg.ExportOption(task.ExportOptionKey)
		if !ok {
			return Snapshot{}, fmt.Errorf("export option %q was not found", task.ExportOptionKey)
		}
		return e.exportNowledgeMem(ctx, sourceCfg, exportOption.Export)
	case "directory":
		if err := config.ValidateDirectorySource(sourceCfg.Directory); err != nil {
			return Snapshot{}, err
		}
		return e.exportDirectory(sourceCfg.Directory.Path)
	default:
		return Snapshot{}, fmt.Errorf("unsupported source type %q", sourceCfg.Type)
	}
}

func (e *Exporter) exportNowledgeMem(ctx context.Context, sourceCfg config.SourceConfig, exportCfg config.ExportConfig) (Snapshot, error) {
	timeout := sourceCfg.NowledgeMem.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	client := mem.NewClient(
		mem.WithBaseURL(sourceCfg.NowledgeMem.APIURL),
		mem.WithTimeout(timeout),
	)
	if sourceCfg.NowledgeMem.APIKey != "" {
		client = mem.NewClient(
			mem.WithBaseURL(sourceCfg.NowledgeMem.APIURL),
			mem.WithAPIKey(sourceCfg.NowledgeMem.APIKey),
			mem.WithTimeout(timeout),
		)
	}
	defer client.Close()

	if err := client.Data.Checkpoint(ctx); err != nil {
		return Snapshot{}, fmt.Errorf("nowledge mem checkpoint: %w", err)
	}
	data, err := client.Data.DownloadExport(ctx, exportDownloadRequest(exportCfg))
	if err != nil {
		return Snapshot{}, fmt.Errorf("download nowledge mem export: %w", err)
	}
	return Snapshot{
		Data:      data,
		SizeBytes: int64(len(data)),
	}, nil
}

func exportDownloadRequest(exportCfg config.ExportConfig) *mem.DataExportDownloadRequest {
	return &mem.DataExportDownloadRequest{
		IncludeMemories:             exportCfg.IncludeMemories,
		IncludeThreads:              exportCfg.IncludeThreads,
		IncludeMessages:             exportCfg.IncludeMessages,
		IncludeEntities:             exportCfg.IncludeEntities,
		IncludeLabels:               exportCfg.IncludeLabels,
		IncludeSources:              exportCfg.IncludeSources,
		IncludeCommunities:          exportCfg.IncludeCommunities,
		IncludeSkills:               exportCfg.IncludeSkills,
		IncludeEdges:                exportCfg.IncludeEdges,
		IncludeWorkingMemory:        exportCfg.IncludeWorkingMemory,
		IncludeWorkingMemoryArchive: exportCfg.IncludeWorkingMemoryArchive,
		IncludeSourceFiles:          exportCfg.IncludeSourceFiles,
	}
}

func (e *Exporter) exportDirectory(root string) (Snapshot, error) {
	root = filepath.Clean(root)
	fs := afero.NewReadOnlyFs(afero.NewBasePathFs(afero.NewOsFs(), root))
	info, err := fs.Stat(".")
	if err != nil {
		return Snapshot{}, fmt.Errorf("stat directory source: %w", err)
	}
	if !info.IsDir() {
		return Snapshot{}, fmt.Errorf("directory source %q is not a directory", root)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	var files int
	err = afero.Walk(fs, ".", func(name string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		name = strings.TrimPrefix(filepath.ToSlash(name), "./")
		if strings.Contains(name, "\x00") {
			return fmt.Errorf("path contains invalid character: %q", name)
		}
		if info.IsDir() {
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			header.Name = strings.TrimRight(name, "/") + "/"
			_, err = zw.CreateHeader(header)
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = name
		header.Method = zip.Deflate
		w, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		f, err := fs.Open(name)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		files++
		return nil
	})
	if closeErr := zw.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("zip directory source: %w", err)
	}
	return Snapshot{
		Data:      buf.Bytes(),
		SizeBytes: int64(buf.Len()),
		ItemCount: files,
	}, nil
}
