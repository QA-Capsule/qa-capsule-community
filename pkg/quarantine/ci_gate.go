package quarantine

import (
	"context"
	"errors"
	"strings"
)

// ErrMissingCIIdentifier is returned when neither hash nor test name is provided.
var ErrMissingCIIdentifier = errors.New("hash or test query parameter required")

// CIStatusResponse is returned to CI runners before executing a test.
type CIStatusResponse struct {
	ProjectName             string `json:"project_name"`
	Quarantined             bool   `json:"quarantined"`
	Skip                    bool   `json:"skip"`
	TestName                string `json:"test_name,omitempty"`
	TestIdentityFingerprint string `json:"fingerprint,omitempty"`
	Reason                  string `json:"reason,omitempty"`
	Source                  string `json:"source,omitempty"`
	Since                   string `json:"since,omitempty"`
	Message                 string `json:"message,omitempty"`
}

// ResolveCIIdentity maps CI query params to a stable test identity fingerprint.
// hash should be the test_identity fingerprint from GET /api/ci/quarantine (64-char hex).
// test is the human-readable test name when hash is unknown.
func ResolveCIIdentity(projectName, hash, testName string) (identityFP, displayName string, err error) {
	projectName = strings.TrimSpace(projectName)
	hash = strings.TrimSpace(strings.ToLower(hash))
	testName = NormalizeTestName(testName)

	if hash != "" {
		if !isHexFingerprint(hash) {
			return "", "", errors.New("hash must be a 64-character hex fingerprint")
		}
		if testName == "" {
			testName = hash[:8] + "…"
		}
		return hash, testName, nil
	}
	if testName != "" {
		return TestIdentityFingerprint(projectName, testName), testName, nil
	}
	return "", "", ErrMissingCIIdentifier
}

func isHexFingerprint(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// CheckCIStatus reports whether the pipeline should skip executing this test.
func (e *Engine) CheckCIStatus(ctx context.Context, projectName, hash, testName string) (CIStatusResponse, error) {
	out := CIStatusResponse{
		ProjectName: projectName,
		Quarantined: false,
		Skip:        false,
		Message:     "Test is not quarantined; execute normally.",
	}
	if e == nil || e.repo == nil {
		return out, nil
	}

	identityFP, displayName, err := ResolveCIIdentity(projectName, hash, testName)
	if err != nil {
		return out, err
	}
	out.TestIdentityFingerprint = identityFP
	out.TestName = displayName

	ent, err := e.repo.ActiveEntry(ctx, projectName, identityFP)
	if err != nil {
		return out, err
	}
	if ent == nil {
		return out, nil
	}

	out.Quarantined = true
	out.Skip = true
	out.Reason = string(ent.Reason)
	out.Source = string(ent.Source)
	out.Since = ent.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
	out.Message = "Test is quarantined; skip execution in CI."
	if ent.TestName != "" {
		out.TestName = ent.TestName
	}
	return out, nil
}
