package main

import (
	"github.com/QA-Capsule/qa-capsule-community/pkg/api/server"
	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

func main() {
	// 1. Charger la configuration
	config := core.LoadConfig()
	core.InitDB("qa-capsule.db")
	server.Start(config)
}
