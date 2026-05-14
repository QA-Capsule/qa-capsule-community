package main

import (
	"qacapsule/internal/api/server"
	"qacapsule/internal/core"
)

func main() {
	// 1. Charger la configuration
	config := core.LoadConfig()
	core.InitDB("qa-capsule.db")
	server.Start(config)
}