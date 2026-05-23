package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
	"github.com/spf13/cobra"
)

var (
	apiBaseURL string
	apiKey     string
)

func main() {
	root := &cobra.Command{
		Use:   "qacapsule",
		Short: "QA Capsule local test wrapper for developers",
	}
	root.PersistentFlags().StringVar(&apiBaseURL, "api", envOr("QACAPSULE_API_URL", "http://localhost:9000"), "QA Capsule API base URL")
	root.PersistentFlags().StringVar(&apiKey, "api-key", os.Getenv("QACAPSULE_API_KEY"), "Project API key (X-API-Key)")

	root.AddCommand(runCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runCmd() *cobra.Command {
	var testName, testError string
	cmd := &cobra.Command{
		Use:   "run [test command...]",
		Short: "Run a local test command and check flaky status on failure",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := exec.Command(args[0], args[1:]...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Stdin = os.Stdin

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigCh
				if c.Process != nil {
					_ = c.Process.Signal(os.Interrupt)
				}
			}()

			err := c.Run()
			if err == nil {
				return nil
			}

			name := testName
			if name == "" {
				name = os.Getenv("QACAPSULE_TEST_NAME")
			}
			if name == "" {
				name = args[0]
			}
			errText := testError
			if errText == "" {
				errText = os.Getenv("QACAPSULE_TEST_ERROR")
			}
			if errText == "" {
				errText = err.Error()
			}
			hash := core.IncidentFingerprint(name, errText)
			if flaky, msg := checkFlaky(hash); flaky {
				fmt.Fprintf(os.Stderr, "\n\x1b[33m⚠️  Ce test a échoué, mais il est instable en CI. Vous pouvez l'ignorer.\x1b[0m\n   %s\n", msg)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&testName, "test-name", "", "Test name for flaky fingerprint (same as CI)")
	cmd.Flags().StringVar(&testError, "test-error", "", "Error message for flaky fingerprint")
	return cmd
}

func checkFlaky(fingerprint string) (bool, string) {
	url := fmt.Sprintf("%s/api/incidents/check-flaky/%s", trimSlash(apiBaseURL), fingerprint)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, ""
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return false, ""
	}
	var out struct {
		Flaky   bool   `json:"flaky"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &out) == nil && out.Flaky {
		return true, out.Message
	}
	return false, ""
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
