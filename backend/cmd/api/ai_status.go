package main

import (
	"log/slog"

	"github.com/bilimbaga/bilimbaga/internal/config"
)

// logAIStatus logs the AI state once at startup (#510). With a provider it is one INFO line naming the provider and
// model. Without one it is one WARN line with the reason, which names variables and never a value. The key is never
// logged.
func logAIStatus(logger *slog.Logger, cfg config.AIConfig) {
	if cfg.Provider == "" {
		logger.Warn(cfg.OffReason)
		return
	}
	logger.Info("AI provider configured", "provider", cfg.Provider, "model", cfg.Model)
}
