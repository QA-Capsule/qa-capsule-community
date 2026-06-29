package core

// UserCanAccessIncidentByID reports whether the user may mutate or read a
// sensitive incident resource. Admins and managers have global access; other
// roles are limited to incidents whose project belongs to one of their teams.
func UserCanAccessIncidentByID(username, role string, incidentID int64) bool {
	r := NormalizeRole(role)
	if r == RoleAdmin || r == RoleManager {
		return true
	}
	if DB == nil || incidentID <= 0 {
		return false
	}
	var projectName string
	if err := DB.QueryRow(`SELECT project_name FROM incidents WHERE id = ?`, incidentID).Scan(&projectName); err != nil {
		return false
	}
	var projectID string
	if err := DB.QueryRow(`SELECT id FROM projects WHERE name = ?`, projectName).Scan(&projectID); err != nil {
		return false
	}
	return UserCanAccessProject(username, role, projectID)
}

// UserCanAccessAllIncidents returns true only when the user may access every
// incident in ids (used for bulk resolve/delete).
func UserCanAccessAllIncidents(username, role string, ids []int) bool {
	for _, id := range ids {
		if id <= 0 {
			return false
		}
		if !UserCanAccessIncidentByID(username, role, int64(id)) {
			return false
		}
	}
	return true
}
