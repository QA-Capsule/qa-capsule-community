package core

import "testing"

func TestCanManageWorkflow(t *testing.T) {
	if !CanManageWorkflow(RoleAdmin) {
		t.Fatal("admin should manage workflow")
	}
	if !CanManageWorkflow(RoleManager) {
		t.Fatal("manager should manage workflow")
	}
	if !CanManageWorkflow(RoleLead) {
		t.Fatal("lead should manage workflow")
	}
	if CanManageWorkflow(RoleObserver) {
		t.Fatal("observer must not manage workflow")
	}
	if CanManageWorkflow(RoleViewerLegacy) {
		t.Fatal("legacy viewer must not manage workflow")
	}
}
