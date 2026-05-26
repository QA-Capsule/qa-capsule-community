package core

import "testing"

func TestRBAC_Intelligence(t *testing.T) {
	if !CanViewHealing(RoleObserver) {
		t.Fatal("observer should view healing")
	}
	if CanManageQuarantine(RoleObserver) {
		t.Fatal("observer cannot manage quarantine")
	}
	if !CanManageQuarantine(RoleLead) {
		t.Fatal("lead should manage quarantine")
	}
	if !CanManageHealing(RoleLead) {
		t.Fatal("lead should manage healing")
	}
	if CanManageHealing(RoleObserver) {
		t.Fatal("observer should not manage healing")
	}
}
