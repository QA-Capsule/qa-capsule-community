package core

// RoleDisplayName returns the human-facing role label shown in the UI.
func RoleDisplayName(role string) string {
	switch NormalizeRole(role) {
	case RoleAdmin:
		return "Platform Admin"
	case RoleManager:
		return "Manager"
	case RoleLead:
		return "Lead"
	case RoleObserver:
		return "Observer"
	default:
		return role
	}
}

// AllowedRolesMessage lists valid role codes with display names for API errors.
func AllowedRolesMessage() string {
	return "Allowed roles: admin (Platform Admin), manager (Manager), lead (Lead), observer (Observer)"
}
