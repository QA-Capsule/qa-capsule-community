package main

import (
	"log/slog"
	"os"

	"github.com/QA-Capsule/qa-capsule-community/pkg/api/server"
	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	core.EnsureProjectRoot()
	config := core.LoadConfig()
	if err := core.InitRemediationEngine(config.Plugins.Directory); err != nil {
		slog.Error("remediation engine init failed", "error", err)
		os.Exit(1)
	}
	core.InitDB("qa-capsule.db")
	server.Start(config)
}
