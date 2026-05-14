package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

// AutoProvisionGitLabWebhook automates the creation of a webhook directly inside a GitLab repository.
func AutoProvisionGitLabWebhook(repoPath string, webhookURL string, secretToken string, gitlabToken string) error {
	log.Printf("[GITOPS] Attempting to auto-provision webhook in GitLab repository: %s", repoPath)

	// GitLab API requires the project path to be URL-encoded (e.g., "company/payment-api" -> "company%2Fpayment-api")
	encodedRepoPath := url.PathEscape(repoPath)
	apiURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/hooks", encodedRepoPath)

	// Payload defining exactly what SRE events we want GitLab to send us
	payload := map[string]interface{}{
		"url":                     webhookURL,
		"push_events":             false,
		"pipeline_events":         true,  // We want to know when pipelines crash
		"job_events":              true,  // We want to know when specific jobs fail
		"enable_ssl_verification": false, // Set to true in Production
		"token":                   secretToken,
	}

	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// Authenticate with GitLab using the SRE Admin's Personal Access Token
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", gitlabToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitLab API rejected the webhook (Status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	log.Printf("[GITOPS SUCCESS] SRE Webhook successfully injected into %s!", repoPath)
	return nil
}