package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"qacapsule/internal/core"

	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v3"
)

// registerSystemRoutes binds plugins, system settings, and websocket endpoints
func registerSystemRoutes(config *core.Config) {

	// Fetch all installed plugins by scanning JSON manifests
	http.HandleFunc("/api/plugins", jwtAuthMiddleware(config, false, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var pluginList []map[string]interface{}
		filepath.WalkDir(config.Plugins.Directory, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && filepath.Ext(path) == ".json" {
				fileData, _ := os.ReadFile(path)
				var pluginData map[string]interface{}
				if json.Unmarshal(fileData, &pluginData) == nil {
					relPath, _ := filepath.Rel(config.Plugins.Directory, path)
					pluginData["file_path"] = relPath
					pluginList = append(pluginList, pluginData)
				}
			}
			return nil
		})
		json.NewEncoder(w).Encode(pluginList)
	}))

	// Execute a specific plugin manually from the UI
	http.HandleFunc("/api/plugins/run", jwtAuthMiddleware(config, false, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FilePath string `json:"file_path"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		logs, err := core.RunSinglePlugin(*config, req.FilePath, "MANUAL", nil)

		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Execution failed", "logs": logs})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"logs": logs})
	}))

	// Update Plugin JSON configurations
	http.HandleFunc("/api/plugins/config", jwtAuthMiddleware(config, true, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FilePath string            `json:"file_path"`
			Env      map[string]string `json:"env"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		fullPath := filepath.Join(config.Plugins.Directory, req.FilePath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("[ERROR] Cannot read plugin file %s: %v", fullPath, err)
			http.Error(w, "Plugin file not found", http.StatusNotFound)
			return
		}

		var p map[string]interface{}
		if err := json.Unmarshal(data, &p); err != nil {
			log.Printf("[ERROR] Cannot parse plugin JSON: %v", err)
			http.Error(w, "Invalid plugin JSON", http.StatusInternalServerError)
			return
		}

		if p == nil {
			p = make(map[string]interface{})
		}

		p["env"] = req.Env

		newData, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
			return
		}

		if err := os.WriteFile(fullPath, newData, 0644); err != nil {
			log.Printf("[ERROR] Cannot write to plugin file %s: %v", fullPath, err)
			http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	// Fetch current global config.yaml
	http.HandleFunc("/api/config", jwtAuthMiddleware(config, false, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(config)
	}))

	// Update SMTP block in config.yaml
	http.HandleFunc("/api/config/smtp", jwtAuthMiddleware(config, true, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&config.SMTP)
		data, _ := yaml.Marshal(config)
		os.WriteFile("config.yaml", data, 0644)
		w.WriteHeader(http.StatusOK)
	}))

	// Update Security Policies in config.yaml
	http.HandleFunc("/api/config/policy", jwtAuthMiddleware(config, true, func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			AllowedDomain string `json:"allowed_domain"`
		}
		json.NewDecoder(r.Body).Decode(&payload)
		config.Security.AllowedDomain = payload.AllowedDomain
		data, _ := yaml.Marshal(config)
		os.WriteFile("config.yaml", data, 0644)
		w.WriteHeader(http.StatusOK)
	}))

	// Real-time raw telemetry stream for the dashboard
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			data, _ := os.ReadFile(config.Telemetry.ReportPath)
			if conn.WriteMessage(websocket.TextMessage, data) != nil {
				break
			}
			time.Sleep(1 * time.Second)
		}
	})
}