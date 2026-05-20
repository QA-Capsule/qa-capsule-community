package core

import (
	"database/sql"
	"log"
)

// GetDescendantTeamIDs returns all child team IDs under rootTeamID (not including root).
func GetDescendantTeamIDs(rootTeamID int) ([]int, error) {
	rows, err := DB.Query("SELECT id, parent_id FROM teams")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	children := map[int][]int{}
	for rows.Next() {
		var id int
		var parent sql.NullInt64
		if err := rows.Scan(&id, &parent); err != nil {
			continue
		}
		if parent.Valid {
			pid := int(parent.Int64)
			children[pid] = append(children[pid], id)
		}
	}

	var out []int
	var walk func(int)
	walk = func(pid int) {
		for _, cid := range children[pid] {
			out = append(out, cid)
			walk(cid)
		}
	}
	walk(rootTeamID)
	return out, nil
}

// EnsureUserTeamMembership grants direct membership if missing (does not override inherited-only rows).
func EnsureUserTeamMembership(userID, teamID int, teamRole string) error {
	if teamRole == "" {
		teamRole = "team_viewer"
	}
	var existing int
	_ = DB.QueryRow("SELECT COUNT(*) FROM user_teams WHERE user_id = ? AND team_id = ?", userID, teamID).Scan(&existing)
	if existing > 0 {
		_, err := DB.Exec(`
			UPDATE user_teams SET team_role = ?, inherited_from = NULL
			WHERE user_id = ? AND team_id = ? AND inherited_from IS NULL`,
			teamRole, userID, teamID)
		return err
	}
	_, err := DB.Exec(`
		INSERT INTO user_teams (user_id, team_id, team_role, inherited_from)
		VALUES (?, ?, ?, NULL)`,
		userID, teamID, teamRole)
	return err
}

// AssignUserToTeamWithInheritance adds user to team and propagates to descendants when propagate is true.
func AssignUserToTeamWithInheritance(userID, teamID int, teamRole string, propagate bool) error {
	if err := EnsureUserTeamMembership(userID, teamID, teamRole); err != nil {
		return err
	}
	if !propagate {
		return nil
	}
	return PropagateMembershipToDescendants(userID, teamID, teamRole)
}

// PropagateMembershipToDescendants inherits membership on sub-groups unless opted out.
func PropagateMembershipToDescendants(userID, ancestorTeamID int, teamRole string) error {
	descendants, err := GetDescendantTeamIDs(ancestorTeamID)
	if err != nil {
		return err
	}
	for _, teamID := range descendants {
		var opted int
		_ = DB.QueryRow(`
			SELECT COUNT(*) FROM user_team_inheritance_optouts
			WHERE user_id = ? AND team_id = ? AND ancestor_team_id = ?`,
			userID, teamID, ancestorTeamID).Scan(&opted)
		if opted > 0 {
			continue
		}

		var direct int
		_ = DB.QueryRow(`
			SELECT COUNT(*) FROM user_teams
			WHERE user_id = ? AND team_id = ? AND inherited_from IS NULL`,
			userID, teamID).Scan(&direct)
		if direct > 0 {
			continue
		}

		var exists int
		_ = DB.QueryRow("SELECT COUNT(*) FROM user_teams WHERE user_id = ? AND team_id = ?", userID, teamID).Scan(&exists)
		if exists > 0 {
			_, err = DB.Exec(`
				UPDATE user_teams SET team_role = ?, inherited_from = ?
				WHERE user_id = ? AND team_id = ?`,
				teamRole, ancestorTeamID, userID, teamID)
		} else {
			_, err = DB.Exec(`
				INSERT INTO user_teams (user_id, team_id, team_role, inherited_from)
				VALUES (?, ?, ?, ?)`,
				userID, teamID, teamRole, ancestorTeamID)
		}
		if err != nil {
			log.Printf("[WARN] propagate membership user=%d team=%d: %v", userID, teamID, err)
		}
	}
	return nil
}

// RemoveUserFromTeam removes membership; opt-out inherited rows; cascade when removing direct parent assign.
func RemoveUserFromTeam(userID, teamID int) error {
	var inheritedFrom sql.NullInt64
	err := DB.QueryRow("SELECT inherited_from FROM user_teams WHERE user_id = ? AND team_id = ?", userID, teamID).Scan(&inheritedFrom)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	if inheritedFrom.Valid {
		_, _ = DB.Exec(`
			INSERT OR IGNORE INTO user_team_inheritance_optouts (user_id, team_id, ancestor_team_id)
			VALUES (?, ?, ?)`,
			userID, teamID, inheritedFrom.Int64)
		_, err = DB.Exec("DELETE FROM user_teams WHERE user_id = ? AND team_id = ?", userID, teamID)
		return err
	}

	_, err = DB.Exec("DELETE FROM user_teams WHERE user_id = ? AND team_id = ?", userID, teamID)
	if err != nil {
		return err
	}
	_, _ = DB.Exec("DELETE FROM user_teams WHERE user_id = ? AND inherited_from = ?", userID, teamID)
	descendants, _ := GetDescendantTeamIDs(teamID)
	for _, d := range descendants {
		_, _ = DB.Exec("DELETE FROM user_team_inheritance_optouts WHERE user_id = ? AND team_id = ? AND ancestor_team_id = ?", userID, d, teamID)
	}
	return nil
}
