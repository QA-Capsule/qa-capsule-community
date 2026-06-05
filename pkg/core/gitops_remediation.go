package core

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// CreateRemediationPR opens a GitHub pull request with an updated file (any language/extension).
// repo must be "owner/name". Authentication uses GITHUB_TOKEN (or GITHUB_PAT).
func CreateRemediationPR(repo string, filePath string, newCode string) (prURL string, err error) {
	repo = strings.TrimSpace(repo)
	filePath = strings.TrimSpace(strings.TrimPrefix(filePath, "/"))
	if repo == "" || filePath == "" {
		return "", fmt.Errorf("repo and filePath are required")
	}
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("repo must be owner/name")
	}
	owner, name := parts[0], parts[1]

	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(os.Getenv("GITHUB_PAT"))
	}
	if token == "" {
		return "", fmt.Errorf("GITHUB_TOKEN or GITHUB_PAT required for remediation PR")
	}

	client := &http.Client{Timeout: 90 * time.Second}
	base := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, name)

	defaultBranch, err := githubDefaultBranch(client, base, token)
	if err != nil {
		return "", err
	}

	headSHA, err := githubBranchSHA(client, base, token, defaultBranch)
	if err != nil {
		return "", err
	}

	branchName := fmt.Sprintf("qa-capsule/heal-%d", time.Now().Unix())
	if err := githubCreateBranch(client, base, token, branchName, headSHA); err != nil {
		return "", err
	}

	fileSHA, err := githubFileSHA(client, base, token, filePath, defaultBranch)
	if err != nil {
		return "", err
	}

	msg := fmt.Sprintf("fix(test): self-healing proposal for %s", filePath)
	if err := githubPutFile(client, base, token, filePath, branchName, newCode, msg, fileSHA); err != nil {
		return "", err
	}

	prURL, err = githubCreatePR(client, base, token, branchName, defaultBranch,
		fmt.Sprintf("QA Capsule: remediate %s", filePath),
		"Self-healing fix proposal from QA Capsule (framework-agnostic). Review before merging.")
	return prURL, err
}

func githubDefaultBranch(client *http.Client, base, token string) (string, error) {
	var repo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := githubGET(client, base, token, &repo); err != nil {
		return "", err
	}
	if repo.DefaultBranch == "" {
		return "main", nil
	}
	return repo.DefaultBranch, nil
}

func githubBranchSHA(client *http.Client, base, token, branch string) (string, error) {
	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	url := fmt.Sprintf("%s/git/ref/heads/%s", base, branch)
	if err := githubGET(client, url, token, &ref); err != nil {
		return "", err
	}
	if ref.Object.SHA == "" {
		return "", fmt.Errorf("could not resolve branch %s", branch)
	}
	return ref.Object.SHA, nil
}

func githubCreateBranch(client *http.Client, base, token, branch, sha string) error {
	body := map[string]string{"ref": "refs/heads/" + branch, "sha": sha}
	url := base + "/git/refs"
	return githubPOST(client, url, token, body, nil)
}

func githubEncodePath(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}

func githubFileSHA(client *http.Client, base, token, path, ref string) (string, error) {
	url := fmt.Sprintf("%s/contents/%s?ref=%s", base, githubEncodePath(path), ref)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	githubHeaders(req, token)
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return "", nil
	}
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return "", fmt.Errorf("github contents GET %d: %s", res.StatusCode, truncateGitHubErr(raw))
	}
	var meta struct {
		SHA string `json:"sha"`
	}
	if json.Unmarshal(raw, &meta) != nil {
		return "", nil
	}
	return meta.SHA, nil
}

func githubPutFile(client *http.Client, base, token, path, branch, content, message, fileSHA string) error {
	body := map[string]interface{}{
		"message": message,
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
		"branch":  branch,
	}
	if fileSHA != "" {
		body["sha"] = fileSHA
	}
	apiURL := fmt.Sprintf("%s/contents/%s", base, githubEncodePath(path))
	return githubPOST(client, apiURL, token, body, nil)
}

func githubCreatePR(client *http.Client, base, token, head, baseBranch, title, body string) (string, error) {
	payload := map[string]string{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  baseBranch,
	}
	var out struct {
		HTMLURL string `json:"html_url"`
	}
	url := base + "/pulls"
	if err := githubPOST(client, url, token, payload, &out); err != nil {
		return "", err
	}
	if out.HTMLURL == "" {
		return "", fmt.Errorf("github PR created but html_url missing")
	}
	return out.HTMLURL, nil
}

func githubGET(client *http.Client, url, token string, dest interface{}) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	githubHeaders(req, token)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return fmt.Errorf("github GET %d: %s", res.StatusCode, truncateGitHubErr(raw))
	}
	return json.Unmarshal(raw, dest)
}

func githubPOST(client *http.Client, url, token string, body interface{}, dest interface{}) error {
	data, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	githubHeaders(req, token)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return fmt.Errorf("github POST %d: %s", res.StatusCode, truncateGitHubErr(raw))
	}
	if dest != nil {
		return json.Unmarshal(raw, dest)
	}
	return nil
}

func githubHeaders(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
}

func truncateGitHubErr(raw []byte) string {
	s := string(raw)
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}
