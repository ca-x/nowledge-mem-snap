package source

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	mem "github.com/lib-x/nowledgemem-go"

	"github.com/lib-x/nowledge-mem-snap/internal/config"
)

type TestResult struct {
	OK      bool              `json:"ok"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func Test(ctx context.Context, sourceCfg config.SourceConfig) TestResult {
	cfg := config.ApplyEnv(config.Config{
		Sources: []config.SourceConfig{sourceCfg},
	})
	if len(cfg.Sources) == 0 {
		return TestResult{OK: false, Message: "source is required"}
	}
	sourceCfg = cfg.Sources[0]
	switch sourceCfg.Type {
	case "nowledgemem_api":
		return testNowledgeMem(ctx, sourceCfg.NowledgeMem)
	case "directory":
		return testDirectory(sourceCfg.Directory)
	default:
		return TestResult{OK: false, Message: fmt.Sprintf("unsupported source type %q", sourceCfg.Type)}
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
			Message: fmt.Sprintf("API key environment variable %s is not set", cfg.APIKeyEnv),
		}
	}
	parsed, err := url.Parse(cfg.APIURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return TestResult{OK: false, Message: "API URL must be an absolute http(s) URL"}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return TestResult{OK: false, Message: "API URL must use http or https"}
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
		return TestResult{OK: false, Message: "health check failed: " + err.Error()}
	}
	if err := client.Data.Checkpoint(ctx); err != nil {
		return TestResult{OK: false, Message: "data checkpoint failed: " + err.Error()}
	}
	details := map[string]string{
		"status": health.Status,
	}
	if health.Version != "" {
		details["version"] = health.Version
	}
	return TestResult{
		OK:      true,
		Message: "Nowledge Mem source is reachable and accepted the API key",
		Details: details,
	}
}

func testDirectory(source config.DirectorySource) TestResult {
	if err := config.ValidateDirectorySource(source); err != nil {
		return TestResult{OK: false, Message: err.Error()}
	}
	root, err := filepath.Abs(source.Path)
	if err != nil {
		return TestResult{OK: false, Message: err.Error()}
	}
	info, err := os.Stat(root)
	if err != nil {
		return TestResult{OK: false, Message: "directory is not readable: " + err.Error()}
	}
	if !info.IsDir() {
		return TestResult{OK: false, Message: fmt.Sprintf("%s is not a directory", source.Path)}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return TestResult{OK: false, Message: "directory cannot be listed: " + err.Error()}
	}
	return TestResult{
		OK:      true,
		Message: "Directory source is readable",
		Details: map[string]string{
			"entries": fmt.Sprintf("%d", len(entries)),
		},
	}
}
