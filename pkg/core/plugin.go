package core

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PluginManifest définit la structure attendue dans le fichier JSON du plugin
type PluginManifest struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Status      string            `json:"status"`
	Command     string            `json:"command"`
	TriggerOn   []string          `json:"trigger_on"`
	Env         map[string]string `json:"env"`
}

// EvaluateAlertRules scanne les alertes et déclenche les plugins appropriés avec le contexte du projet
func EvaluateAlertRules(config Config, alert UnifiedAlert, projectContext map[string]string) {
	log.Printf("[ENGINE] Evaluating rules for new alert: %s", alert.Name)

	alertText := fmt.Sprintf("%v %v %v", alert.Name, alert.Error, alert.ConsoleLogs)

	filepath.WalkDir(config.Plugins.Directory, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(path) == ".json" {
			fileData, _ := os.ReadFile(path)
			var manifest PluginManifest
			
			if err := json.Unmarshal(fileData, &manifest); err == nil && manifest.Status == "Active" {
				triggered := false
				for _, triggerWord := range manifest.TriggerOn {
					if strings.Contains(alertText, triggerWord) {
						triggered = true
						break
					}
				}

				if triggered {
					log.Printf("[ENGINE] Rule matched! Auto-triggering: %s", manifest.Name)
					relPath, _ := filepath.Rel(config.Plugins.Directory, path)
					// On passe le projectContext au script
					go RunSinglePlugin(config, relPath, fmt.Sprintf("AUTO_EVENT: %s", alert.Name), projectContext)
				}
			}
		}
		return nil
	})
}

func RunSinglePlugin(config Config, pluginRelPath string, action string, projectContext map[string]string) (string, error) {
	fullPath := filepath.Join(config.Plugins.Directory, pluginRelPath)
	fileData, err := os.ReadFile(fullPath)
	if err != nil { 
		return "", fmt.Errorf("impossible de lire le manifest: %v", err) 
	}

	var manifest PluginManifest
	if err := json.Unmarshal(fileData, &manifest); err != nil { 
		return "", fmt.Errorf("json invalide: %v", err) 
	}

	if manifest.Command != "" {
		absDir, _ := filepath.Abs(filepath.Dir(fullPath))
		absCmdPath := filepath.Join(absDir, manifest.Command)

		cmd := exec.Command("sh", absCmdPath, action)
		cmd.Dir = absDir
		
		// 1. Variables système
		cmd.Env = os.Environ()
		
		// 2. Variables du fichier JSON (Configuration globale du plugin)
		for key, val := range manifest.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, val))
		}

		// 3. Variables dynamiques depuis l'UI (Écrase la conf globale si présent)
		if projectContext != nil {
			for key, val := range projectContext {
				if val != "" {
					cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, val))
				}
			}
		}

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			log.Printf("[RUNBOOK] Error executing %s: %v", manifest.Name, err)
			return outputStr + fmt.Sprintf("\n[EXIT STATUS] ERREUR: %v", err), err
		}
		
		log.Printf("[RUNBOOK] Success (%s):\n%s", manifest.Name, outputStr)
		return outputStr + "\n[EXIT STATUS] SUCCESS", nil
	}
	
	return "", fmt.Errorf("aucune commande définie dans le plugin")
}

func TriggerPlugins(config Config, action string) {
	filepath.WalkDir(config.Plugins.Directory, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(path) == ".json" {
			relPath, _ := filepath.Rel(config.Plugins.Directory, path)
			go RunSinglePlugin(config, relPath, action, nil)
		}
		return nil
	})
}