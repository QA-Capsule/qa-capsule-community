package core

// Global role identifiers (stored in users.role). UI: Platform Admin, Manager, Lead, Observer.
const (
	RoleAdmin    = "admin"
	RoleManager  = "manager"
	RoleLead     = "lead"
	RoleObserver = "observer"
)

// Legacy codes migrated on startup (operator → lead, viewer → observer).
const (
	RoleOperatorLegacy = "operator"
	RoleViewerLegacy   = "viewer"
)

// NormalizeRole maps deprecated role codes to current ones.
func NormalizeRole(role string) string {
	switch role {
	case RoleOperatorLegacy:
		return RoleLead
	case RoleViewerLegacy:
		return RoleObserver
	default:
		return role
	}
}

// RoleLevel returns a numeric rank for hierarchy comparisons.
func RoleLevel(role string) int {
	switch NormalizeRole(role) {
	case RoleAdmin:
		return 4
	case RoleManager:
		return 3
	case RoleLead:
		return 2
	case RoleObserver:
		return 1
	default:
		return 0
	}
}

// HasMinRole is true when userRole meets or exceeds minRole in the hierarchy.
func HasMinRole(userRole, minRole string) bool {
	return RoleLevel(userRole) >= RoleLevel(minRole)
}

// IsValidRole reports whether role is one of the supported global roles (legacy codes accepted).
func IsValidRole(role string) bool {
	switch NormalizeRole(role) {
	case RoleAdmin, RoleManager, RoleLead, RoleObserver:
		return true
	default:
		return false
	}
}

// IsCanonicalRole is true only for current stored codes (no legacy operator/viewer).
func IsCanonicalRole(role string) bool {
	switch role {
	case RoleAdmin, RoleManager, RoleLead, RoleObserver:
		return true
	default:
		return false
	}
}

func IsManager(role string) bool {
	return NormalizeRole(role) == RoleManager
}

func IsAdmin(role string) bool {
	return NormalizeRole(role) == RoleAdmin
}

func IsLead(role string) bool {
	return NormalizeRole(role) == RoleLead
}

func IsObserver(role string) bool {
	return NormalizeRole(role) == RoleObserver
}

func CanAccessFinOps(role string) bool {
	return IsManager(role)
}

func CanManageTeams(role string) bool {
	return IsAdmin(role) || IsManager(role)
}

func CanManageIAM(role string) bool {
	return IsAdmin(role)
}

func CanManageProjects(role string) bool {
	return IsManager(role) || IsLead(role)
}

func CanAccessPlugins(role string) bool {
	return IsManager(role) || IsLead(role)
}

// CanManagePluginAutoRun allows toggling AUTO-RUN on integrations (Manager or Platform Admin).
func CanManagePluginAutoRun(role string) bool {
	return IsManager(role) || IsAdmin(role)
}

func CanResolveIncidents(role string) bool {
	return HasMinRole(role, RoleLead)
}

func CanDeleteIncidents(role string) bool {
	r := NormalizeRole(role)
	return r == RoleManager || r == RoleLead
}

func CanAccessChartStudio(role string) bool {
	r := NormalizeRole(role)
	return r == RoleManager || r == RoleLead || r == RoleObserver
}

// CanManageWorkflow allows creating/editing visual remediation DAGs (Lead+).
func CanManageWorkflow(role string) bool {
	return HasMinRole(role, RoleLead)
}
<<<<<<< HEAD

// CanViewRCA allows reading AI insights (Observer+ on operations roles).
func CanViewRCA(role string) bool {
	r := NormalizeRole(role)
	return r == RoleLead || r == RoleManager || r == RoleObserver
}

// CanViewQuarantine allows reading quarantine list.
func CanViewQuarantine(role string) bool {
	return CanViewRCA(role)
}

// CanManageQuarantine allows manual quarantine / lift (Lead+).
func CanManageQuarantine(role string) bool {
	return HasMinRole(role, RoleLead)
}

// CanConfigureAI allows changing LLM provider settings (Manager+).
func CanConfigureAI(role string) bool {
	return IsManager(role) || IsAdmin(role)
}
=======
>>>>>>> 70a3559fb4d4fbfe14293d19734d53e04a1553fb
