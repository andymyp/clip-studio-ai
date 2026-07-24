package main

import (
	"log"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/config"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/platform"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/server"
)

func main() {
	cfg := config.Load()
	deps, err := platform.Open(cfg)
	if err != nil {
		log.Fatalf("initialize dependencies: %v", err)
	}
	defer deps.Close()

	if err := server.New(cfg, deps).Run(":" + cfg.Port); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
