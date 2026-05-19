package core

// Global role identifiers (stored in users.role).
const (
	RoleAdmin    = "admin"
	RoleManager  = "manager"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

// RoleLevel returns a numeric rank for hierarchy comparisons (excludes cross-role admin vs manager split).
func RoleLevel(role string) int {
	switch role {
	case RoleAdmin:
		return 4
	case RoleManager:
		return 3
	case RoleOperator:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}

// HasMinRole is true when userRole meets or exceeds minRole in the hierarchy.
func HasMinRole(userRole, minRole string) bool {
	return RoleLevel(userRole) >= RoleLevel(minRole)
}

// IsValidRole reports whether role is one of the supported global roles.
func IsValidRole(role string) bool {
	switch role {
	case RoleAdmin, RoleManager, RoleOperator, RoleViewer:
		return true
	default:
		return false
	}
}

// IsManager is true only for the QA Manager / SRE Lead role (not admin).
func IsManager(role string) bool {
	return role == RoleManager
}

// IsAdmin is true only for the System Admin role.
func IsAdmin(role string) bool {
	return role == RoleAdmin
}

// CanAccessFinOps — exclusive to Manager (admin handles infrastructure only).
func CanAccessFinOps(role string) bool {
	return IsManager(role)
}

// CanManageTeams — workspace hierarchy (Admin or Manager).
func CanManageTeams(role string) bool {
	return IsAdmin(role) || IsManager(role)
}

// CanManageIAM — global users and roles (Admin only).
func CanManageIAM(role string) bool {
	return IsAdmin(role)
}

// CanManageProjects — CI gateways & routing (Manager or Operator, not Admin).
func CanManageProjects(role string) bool {
	return IsManager(role) || role == RoleOperator
}

// CanAccessPlugins — plugin engine (Manager or Operator, not Admin).
func CanAccessPlugins(role string) bool {
	return IsManager(role) || role == RoleOperator
}

// CanResolveIncidents — Operator and above (not viewer).
func CanResolveIncidents(role string) bool {
	return HasMinRole(role, RoleOperator)
}

// CanAccessChartStudio — custom metrics charts (Manager, QA Lead, QA/Dev — not Admin).
func CanAccessChartStudio(role string) bool {
	return IsManager(role) || role == RoleOperator || role == RoleViewer
}
