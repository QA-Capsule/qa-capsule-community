package core

import "testing"

func TestRBAC_Intelligence(t *testing.T) {
	if !CanViewRCA(RoleObserver) {
		t.Fatal("observer should view RCA")
	}
	if CanManageQuarantine(RoleObserver) {
		t.Fatal("observer cannot manage quarantine")
	}
	if !CanManageQuarantine(RoleLead) {
		t.Fatal("lead should manage quarantine")
	}
	if !CanConfigureAI(RoleManager) {
		t.Fatal("manager configures AI")
	}
	if CanConfigureAI(RoleLead) {
		t.Fatal("lead should not configure AI provider")
	}
}
