package storage

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ca-x/nowledge-mem-snap/internal/config"
)

type TestResult struct {
	OK      bool              `json:"ok"`
	Code    string            `json:"code,omitempty"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func Test(ctx context.Context, targetCfg config.TargetConfig, uploadFile bool) TestResult {
	cfg := config.ApplyEnv(config.Config{
		Targets: []config.TargetConfig{targetCfg},
	})
	if len(cfg.Targets) == 0 {
		return TestResult{OK: false, Code: "target_required", Message: "target is required"}
	}
	targetCfg = cfg.Targets[0]
	if err := config.ValidateTargetConfig(targetCfg); err != nil {
		return TestResult{
			OK:      false,
			Code:    "target_invalid",
			Message: "target config is invalid: " + err.Error(),
			Details: map[string]string{
				"error": err.Error(),
				"type":  targetCfg.Type,
			},
		}
	}

	if !uploadFile {
		return probeTarget(ctx, targetCfg)
	}

	targetCfg.Enabled = true
	target, err := NewFactory().Target(ctx, targetCfg)
	if err != nil {
		return TestResult{
			OK:      false,
			Code:    "target_open_failed",
			Message: "target cannot be opened: " + err.Error(),
			Details: map[string]string{
				"error": err.Error(),
				"type":  targetCfg.Type,
			},
		}
	}

	objectName := ".nowledge-mem-snap-test-" + time.Now().UTC().Format("20060102T150405.000000000Z") + ".txt"
	written, err := Write(ctx, target, objectName, []byte("nowledge-mem-snap target test\n"))
	if err != nil {
		return TestResult{
			OK:      false,
			Code:    "target_write_failed",
			Message: "target write test failed: " + err.Error(),
			Details: map[string]string{
				"error":  err.Error(),
				"type":   targetCfg.Type,
				"object": objectName,
			},
		}
	}
	if err := target.Fs.Remove(objectName); err != nil && !os.IsNotExist(err) {
		return TestResult{
			OK:      false,
			Code:    "target_cleanup_failed",
			Message: "target test file was written but cleanup failed: " + err.Error(),
			Details: map[string]string{
				"error":  err.Error(),
				"type":   targetCfg.Type,
				"object": objectName,
			},
		}
	}
	return TestResult{
		OK:      true,
		Code:    "target_upload_ok",
		Message: "Target is reachable and accepted a test write",
		Details: map[string]string{
			"bytes":  fmt.Sprintf("%d", written),
			"type":   targetCfg.Type,
			"object": objectName,
		},
	}
}

func probeTarget(ctx context.Context, targetCfg config.TargetConfig) TestResult {
	switch targetCfg.Type {
	case "s3":
		_, err := newS3Client(targetCfg.S3).HeadBucket(ctx, &awss3.HeadBucketInput{
			Bucket: aws.String(targetCfg.S3.BucketName),
		})
		if err != nil {
			return TestResult{
				OK:      false,
				Code:    "target_probe_failed",
				Message: "target connection test failed: " + err.Error(),
				Details: map[string]string{
					"error": err.Error(),
					"type":  targetCfg.Type,
				},
			}
		}
	case "webdav":
		client, err := newWebDAVClient(targetCfg.WebDAV)
		if err != nil {
			return TestResult{
				OK:      false,
				Code:    "target_probe_failed",
				Message: "target connection test failed: " + err.Error(),
				Details: map[string]string{
					"error": err.Error(),
					"type":  targetCfg.Type,
				},
			}
		}
		if _, err := (webDAVFileSystem{client: client}).Stat(ctx, "/"); err != nil {
			return TestResult{
				OK:      false,
				Code:    "target_probe_failed",
				Message: "target connection test failed: " + err.Error(),
				Details: map[string]string{
					"error": err.Error(),
					"type":  targetCfg.Type,
				},
			}
		}
	default:
		return TestResult{
			OK:      false,
			Code:    "unsupported_target_type",
			Message: fmt.Sprintf("unsupported target type %q", targetCfg.Type),
			Details: map[string]string{
				"type": targetCfg.Type,
			},
		}
	}
	return TestResult{
		OK:      true,
		Code:    "target_probe_ok",
		Message: "Target connection test succeeded",
		Details: map[string]string{
			"type": targetCfg.Type,
		},
	}
}
