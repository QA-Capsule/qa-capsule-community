//go:build enterprise

package core

import "strings"

type enterpriseEdition struct{}

func (enterpriseEdition) Active() bool {
	if DB == nil {
		return false
	}
	var key string
	if err := DB.QueryRow("SELECT license_key FROM enterprise_config WHERE id = 1").Scan(&key); err != nil {
		return false
	}
	return strings.TrimSpace(key) != ""
}

func init() {
	currentEdition = enterpriseEdition{}
}
