package main

import (
	"github.com/QA-Capsule/qa-capsule-community/pkg/api/server"
	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

func main() {
	config := core.LoadConfig()
	core.InitDB("qa-capsule.db")
	server.Start(config)
}
