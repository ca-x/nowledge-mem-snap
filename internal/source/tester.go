package source

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	mem "github.com/lib-x/nowledgemem-go"
	"github.com/spf13/afero"

	"github.com/lib-x/nowledge-mem-snap/internal/config"
)

type TestResult struct {
	OK      bool              `json:"ok"`
	Code    string            `json:"code,omitempty"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func Test(ctx context.Context, sourceCfg config.SourceConfig) TestResult {
	cfg := config.ApplyEnv(config.Config{
		Sources: []config.SourceConfig{sourceCfg},
	})
	if len(cfg.Sources) == 0 {
		return TestResult{OK: false, Code: "source_required", Message: "source is required"}
	}
	sourceCfg = cfg.Sources[0]
	switch sourceCfg.Type {
	case "nowledgemem_api":
		return testNowledgeMem(ctx, sourceCfg.NowledgeMem)
	case "directory":
		return testDirectory(sourceCfg.Directory)
	default:
		return TestResult{
			OK:      false,
			Code:    "unsupported_source_type",
			Message: fmt.Sprintf("unsupported source type %q", sourceCfg.Type),
			Details: map[string]string{
				"type": string(sourceCfg.Type),
			},
		}
	}
}

func DownloadTest(ctx context.Context, sourceCfg config.SourceConfig, exportCfg config.ExportConfig) ([]byte, TestResult) {
	cfg := config.ApplyEnv(config.Config{
		Sources: []config.SourceConfig{sourceCfg},
	})
	if len(cfg.Sources) == 0 {
		return nil, TestResult{OK: false, Code: "source_required", Message: "source is required"}
	}
	sourceCfg = cfg.Sources[0]
	switch sourceCfg.Type {
	case "nowledgemem_api":
		return downloadNowledgeMem(ctx, sourceCfg.NowledgeMem, exportCfg)
	case "directory":
		if err := config.ValidateDirectorySource(sourceCfg.Directory); err != nil {
			return nil, TestResult{
				OK:      false,
				Code:    "directory_invalid",
				Message: err.Error(),
				Details: map[string]string{
					"error": err.Error(),
					"path":  sourceCfg.Directory.Path,
				},
			}
		}
		snap, err := NewExporter().exportDirectory(sourceCfg.Directory.Path)
		if err != nil {
			return nil, TestResult{
				OK:      false,
				Code:    "directory_download_failed",
				Message: "directory test download failed: " + err.Error(),
				Details: map[string]string{
					"error": err.Error(),
					"path":  sourceCfg.Directory.Path,
				},
			}
		}
		return snap.Data, TestResult{
			OK:      true,
			Code:    "directory_download_ok",
			Message: "Directory source test archive is ready",
			Details: map[string]string{
				"bytes": fmt.Sprintf("%d", snap.SizeBytes),
				"items": fmt.Sprintf("%d", snap.ItemCount),
			},
		}
	default:
		return nil, TestResult{
			OK:      false,
			Code:    "unsupported_source_type",
			Message: fmt.Sprintf("unsupported source type %q", sourceCfg.Type),
			Details: map[string]string{
				"type": string(sourceCfg.Type),
			},
		}
	}
}

func testNowledgeMem(ctx context.Context, cfg config.NowledgeConfig) TestResult {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	if cfg.APIKey == "" && cfg.APIKeyEnv != "" {
		return TestResult{
			OK:      false,
			Code:    "nowledgemem_api_key_required",
			Message: "API Key is required for this Nowledge Mem source",
		}
	}
	parsed, err := url.Parse(cfg.APIURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return TestResult{OK: false, Code: "nowledgemem_api_url_absolute", Message: "API URL must be an absolute http(s) URL"}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return TestResult{OK: false, Code: "nowledgemem_api_url_scheme", Message: "API URL must use http or https"}
	}
	client := mem.NewClient(
		mem.WithBaseURL(cfg.APIURL),
		mem.WithTimeout(timeout),
	)
	if cfg.APIKey != "" {
		client = mem.NewClient(
			mem.WithBaseURL(cfg.APIURL),
			mem.WithAPIKey(cfg.APIKey),
			mem.WithTimeout(timeout),
		)
	}
	defer client.Close()

	health, err := client.Health.Check(ctx)
	if err != nil {
		return TestResult{
			OK:      false,
			Code:    "nowledgemem_health_failed",
			Message: "health check failed: " + err.Error(),
			Details: map[string]string{
				"error": err.Error(),
			},
		}
	}
	if err := client.Data.Checkpoint(ctx); err != nil {
		return TestResult{
			OK:      false,
			Code:    "nowledgemem_checkpoint_failed",
			Message: "data checkpoint failed: " + err.Error(),
			Details: map[string]string{
				"error": err.Error(),
			},
		}
	}
	details := map[string]string{
		"status": health.Status,
	}
	if health.Version != "" {
		details["version"] = health.Version
	}
	return TestResult{
		OK:      true,
		Code:    "nowledgemem_ok",
		Message: "Nowledge Mem source is reachable and accepted the API key",
		Details: details,
	}
}

func downloadNowledgeMem(ctx context.Context, cfg config.NowledgeConfig, exportCfg config.ExportConfig) ([]byte, TestResult) {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	if cfg.APIKey == "" && cfg.APIKeyEnv != "" {
		return nil, TestResult{
			OK:      false,
			Code:    "nowledgemem_api_key_required",
			Message: "API Key is required for this Nowledge Mem source",
		}
	}
	parsed, err := url.Parse(cfg.APIURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, TestResult{OK: false, Code: "nowledgemem_api_url_absolute", Message: "API URL must be an absolute http(s) URL"}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, TestResult{OK: false, Code: "nowledgemem_api_url_scheme", Message: "API URL must use http or https"}
	}
	client := mem.NewClient(
		mem.WithBaseURL(cfg.APIURL),
		mem.WithTimeout(timeout),
	)
	if cfg.APIKey != "" {
		client = mem.NewClient(
			mem.WithBaseURL(cfg.APIURL),
			mem.WithAPIKey(cfg.APIKey),
			mem.WithTimeout(timeout),
		)
	}
	defer client.Close()

	if _, err := client.Health.Check(ctx); err != nil {
		return nil, TestResult{
			OK:      false,
			Code:    "nowledgemem_health_failed",
			Message: "health check failed: " + err.Error(),
			Details: map[string]string{
				"error": err.Error(),
			},
		}
	}
	if err := client.Data.Checkpoint(ctx); err != nil {
		return nil, TestResult{
			OK:      false,
			Code:    "nowledgemem_checkpoint_failed",
			Message: "data checkpoint failed: " + err.Error(),
			Details: map[string]string{
				"error": err.Error(),
			},
		}
	}
	data, err := client.Data.DownloadExport(ctx, exportDownloadRequest(exportCfg))
	if err != nil {
		return nil, TestResult{
			OK:      false,
			Code:    "nowledgemem_download_failed",
			Message: "download nowledge mem export failed: " + err.Error(),
			Details: map[string]string{
				"error": err.Error(),
			},
		}
	}
	return data, TestResult{
		OK:      true,
		Code:    "nowledgemem_download_ok",
		Message: "Nowledge Mem test export is ready",
		Details: map[string]string{
			"bytes": fmt.Sprintf("%d", len(data)),
		},
	}
}

func testDirectory(source config.DirectorySource) TestResult {
	if err := config.ValidateDirectorySource(source); err != nil {
		return TestResult{
			OK:      false,
			Code:    "directory_invalid",
			Message: err.Error(),
			Details: map[string]string{
				"error": err.Error(),
				"path":  source.Path,
			},
		}
	}
	root := filepath.Clean(source.Path)
	fs := afero.NewReadOnlyFs(afero.NewBasePathFs(afero.NewOsFs(), root))
	info, err := fs.Stat(".")
	if err != nil {
		return TestResult{
			OK:      false,
			Code:    "directory_unreadable",
			Message: "directory is not readable: " + err.Error(),
			Details: map[string]string{
				"error": err.Error(),
				"path":  source.Path,
			},
		}
	}
	if !info.IsDir() {
		return TestResult{
			OK:      false,
			Code:    "directory_not_directory",
			Message: fmt.Sprintf("%s is not a directory", source.Path),
			Details: map[string]string{
				"path": source.Path,
			},
		}
	}
	entries, err := afero.ReadDir(fs, ".")
	if err != nil {
		return TestResult{
			OK:      false,
			Code:    "directory_cannot_list",
			Message: "directory cannot be listed: " + err.Error(),
			Details: map[string]string{
				"error": err.Error(),
				"path":  source.Path,
			},
		}
	}
	return TestResult{
		OK:      true,
		Code:    "directory_ok",
		Message: "Directory source is readable",
		Details: map[string]string{
			"entries": fmt.Sprintf("%d", len(entries)),
		},
	}
}
