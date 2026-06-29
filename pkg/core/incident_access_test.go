package core

import (
	"testing"
)

func TestUserCanAccessIncidentByID_adminBypass(t *testing.T) {
	if !UserCanAccessIncidentByID("alice", RoleAdmin, 1) {
		t.Fatal("admin should access any incident")
	}
}

func TestUserCanAccessIncidentByID_invalidID(t *testing.T) {
	if UserCanAccessIncidentByID("alice", RoleLead, 0) {
		t.Fatal("invalid incident id should be denied")
	}
	if UserCanAccessIncidentByID("alice", RoleLead, -1) {
		t.Fatal("negative incident id should be denied")
	}
}

func TestUserCanAccessAllIncidents_empty(t *testing.T) {
	if !UserCanAccessAllIncidents("alice", RoleAdmin, nil) {
		t.Fatal("empty id list should be allowed")
	}
}
