package core

import (
	"database/sql"
	"fmt"
	"time"
)

func IncidentExists(id int64) (bool, error) {
	var n int
	err := DB.QueryRow(`SELECT COUNT(*) FROM incidents WHERE id = ?`, id).Scan(&n)
	return n > 0, err
}

func InsertArtifact(a *IncidentArtifact) (int64, error) {
	res, err := DB.Exec(`INSERT INTO incident_artifacts (incident_id, file_name, content_type, size_bytes, storage_provider, storage_path)
		VALUES (?, ?, ?, ?, ?, ?)`,
		a.IncidentID, a.FileName, a.ContentType, a.SizeBytes, a.StorageProvider, a.StoragePath)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func ListArtifactsByIncident(incidentID int64) ([]IncidentArtifact, error) {
	rows, err := DB.Query(`SELECT id, incident_id, file_name, content_type, size_bytes, storage_provider, storage_path, created_at
		FROM incident_artifacts WHERE incident_id = ? ORDER BY created_at DESC`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IncidentArtifact
	for rows.Next() {
		var a IncidentArtifact
		var created string
		if err := rows.Scan(&a.ID, &a.IncidentID, &a.FileName, &a.ContentType, &a.SizeBytes, &a.StorageProvider, &a.StoragePath, &created); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
		out = append(out, a)
	}
	return out, nil
}

func GetArtifact(id int64) (IncidentArtifact, error) {
	var a IncidentArtifact
	var created string
	err := DB.QueryRow(`SELECT id, incident_id, file_name, content_type, size_bytes, storage_provider, storage_path, created_at
		FROM incident_artifacts WHERE id = ?`, id).Scan(
		&a.ID, &a.IncidentID, &a.FileName, &a.ContentType, &a.SizeBytes, &a.StorageProvider, &a.StoragePath, &created)
	if err == sql.ErrNoRows {
		return a, fmt.Errorf("artifact not found")
	}
	a.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
	return a, err
}

// IsFlakyFingerprint checks if this hash was marked flaky in production recently.
func IsFlakyFingerprint(fingerprint string) (bool, error) {
	var n int
	err := DB.QueryRow(`SELECT COUNT(*) FROM incidents
		WHERE fingerprint = ? AND name LIKE '[FLAKY]%' AND created_at > datetime('now', '-30 days')`,
		fingerprint).Scan(&n)
	return n > 0, err
}
