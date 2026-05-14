package core

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type ProjectConfig struct {
	ID       string `yaml:"id" json:"id"`
	Name     string `yaml:"name" json:"name"`
	CISystem string `yaml:"ci_system" json:"ci_system"`
	APIKey   string `yaml:"api_key" json:"api_key"`
	Routing  struct {
		SlackChannel   string `yaml:"slack_channel" json:"slack_channel"`
		JiraProjectKey string `yaml:"jira_project_key" json:"jira_project_key"`
		TeamsWebhook   string `yaml:"teams_webhook" json:"teams_webhook"`
	} `yaml:"routing" json:"routing"`
}

type TelemetryConfig struct {
	ReportPath          string `yaml:"report_path" json:"report_path"`
	WebsocketTimeoutSec int    `yaml:"websocket_timeout_sec" json:"websocket_timeout_sec"`
	WebhookToken        string `yaml:"webhook_token" json:"webhook_token"`
}

type Config struct {
	Server struct {
		Port     string `yaml:"port" json:"port"`
		LogLevel string `yaml:"log_level" json:"log_level"`
	} `yaml:"server" json:"server"`

	SMTP struct {
		Host     string `yaml:"host" json:"host"`
		Port     int    `yaml:"port" json:"port"`
		User     string `yaml:"user" json:"user"`
		Password string `yaml:"password" json:"password"`
		From     string `yaml:"from" json:"from"`
	} `yaml:"smtp" json:"smtp"`

	Security struct {
		Enabled       bool   `yaml:"enabled" json:"enabled"`
		AdminUser     string `yaml:"admin_user" json:"admin_user"`
		AdminPass     string `yaml:"admin_pass" json:"-"`
		AllowedDomain string `yaml:"allowed_domain" json:"allowed_domain"`
	} `yaml:"security" json:"security"`
	Telemetry TelemetryConfig `yaml:"telemetry" json:"telemetry"`
	Projects []ProjectConfig `yaml:"projects" json:"projects"`
	Plugins   struct {
		Directory  string `yaml:"directory" json:"directory"`
		AutoReload bool   `yaml:"auto_reload" json:"auto_reload"`
	} `yaml:"plugins" json:"plugins"`
	
}

func LoadConfig() Config {
	var config Config
	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal("[FATAL] Configuration file missing: ", err)
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatal("[FATAL] YAML parsing error: ", err)
	}
	return config
}
