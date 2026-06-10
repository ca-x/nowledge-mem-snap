package app

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
	_ "time/tzdata"
)

func configureTimezone(logger *slog.Logger) error {
	tz := strings.TrimSpace(os.Getenv("TZ"))
	if tz == "" {
		logger.Info("timezone configured", "timezone", time.Local.String(), "source", "system")
		return nil
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return fmt.Errorf("load TZ %q: %w", tz, err)
	}
	time.Local = loc
	logger.Info("timezone configured", "timezone", loc.String(), "source", "TZ")
	return nil
}
