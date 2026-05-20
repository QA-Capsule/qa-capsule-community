/**

 * Global RBAC — mirrors pkg/core/rbac.go

 */

export const ROLE_LEVEL = { viewer: 1, operator: 2, manager: 3, admin: 4 };



export const ROLE_LABELS = {

    admin: 'System Admin',

    manager: 'QA Manager / DevOps / SRE Lead',

    operator: 'QA Lead',

    viewer: 'QA / Developer'

};



export const ROLE_DESCRIPTIONS = {

    admin: 'Workspaces hierarchy, global IAM (users & roles), and system settings.',

    manager: 'Full QA/DevOps lead: FinOps, workspaces, chart studio, CI gateways, plugins, incident triage.',

    operator: 'Resolve alerts, Chart Studio, CI gateways, plugins, and project webhooks.',

    viewer: 'Read-only incidents and logs, plus Chart Studio for custom metrics.'

};



export function roleLevel(role) {

    return ROLE_LEVEL[role] || 0;

}



export function hasMinRole(userRole, minRole) {

    return roleLevel(userRole) >= roleLevel(minRole);

}



export function roleLabel(role) {

    return ROLE_LABELS[role] || role;

}



export function isAdmin(role) {

    return role === 'admin';

}



/** FinOps — Manager only. */

export function canAccessFinOps(role) {

    return role === 'manager';

}



/** Workspaces — Admin or Manager. */

export function canManageTeams(role) {

    return role === 'admin' || role === 'manager';

}



/** IAM / global roles — Admin only. */

export function canManageIAM(role) {

    return role === 'admin';

}



export function canManageProjects(role) {

    return role === 'manager' || role === 'operator';

}



export function canAccessPlugins(role) {

    return role === 'manager' || role === 'operator';

}



export function canResolveIncidents(role) {

    return hasMinRole(role, 'operator') && role !== 'admin';

}



export function canAccessChartStudio(role) {

    return role === 'manager' || role === 'operator' || role === 'viewer';

}



/** Operational app areas (dashboard, finops, etc.) — not Admin. */

export function canAccessOperations(role) {

    return role !== 'admin';

}



function setRoleElVisible(el, visible) {
    if (!el) return;
    const isNav = el.classList.contains('nav-item');
    el.style.display = visible ? (isNav ? 'flex' : '') : 'none';
}

/** Elements tagged role-non-admin only (not also manager/operator/workspace/chart). */
function isPureNonAdminEl(el) {
    return el.classList.contains('role-non-admin')
        && !el.classList.contains('role-manager-only')
        && !el.classList.contains('role-operator')
        && !el.classList.contains('role-workspace')
        && !el.classList.contains('role-chart-studio');
}

export function applyRoleVisibility(role) {
    document.querySelectorAll('.role-admin, .admin-only').forEach(el => {
        setRoleElVisible(el, role === 'admin');
    });

    document.querySelectorAll('.role-workspace').forEach(el => {
        setRoleElVisible(el, canManageTeams(role));
    });

    document.querySelectorAll('.role-manager-only, .role-manager').forEach(el => {
        setRoleElVisible(el, role === 'manager');
    });

    document.querySelectorAll('.role-operator').forEach(el => {
        setRoleElVisible(el, role === 'operator' || role === 'manager');
    });

    document.querySelectorAll('.role-chart-studio').forEach(el => {
        setRoleElVisible(el, canAccessChartStudio(role));
    });

    document.querySelectorAll('.role-non-admin').forEach(el => {
        if (!isPureNonAdminEl(el)) return;
        setRoleElVisible(el, role !== 'admin');
    });

    document.querySelectorAll('.role-viewer-only').forEach(el => {
        setRoleElVisible(el, role === 'viewer');
    });

    document.querySelectorAll('.role-hide-viewer').forEach(el => {
        setRoleElVisible(el, role !== 'viewer');
    });
}

/** Views a role may open via switchView (used after login). */
export function defaultViewForRole(role) {
    if (role === 'admin') return 'organizations';
    return 'dashboard';
}

export function canAccessView(role, viewId) {
    switch (viewId) {
        case 'dashboard':
            return canAccessOperations(role);
        case 'about':
        case 'profile':
            return true;
        case 'charts':
            return canAccessChartStudio(role);
        case 'finops':
            return canAccessFinOps(role);
        case 'organizations':
            return canManageTeams(role);
        case 'management':
        case 'settings':
            return canManageIAM(role);
        case 'plugins':
            return canAccessPlugins(role);
        case 'ingestion':
            return canManageProjects(role);
        default:
            return false;
    }
}

export function accessDeniedMessage(viewId) {
    const labels = {
        dashboard: 'Dashboard',
        charts: 'Chart Studio',
        finops: 'FinOps Intelligence',
        organizations: 'Workspaces',
        management: 'IAM Access',
        settings: 'Settings',
        plugins: 'Plugin Engine',
        ingestion: 'CI/CD Gateways'
    };
    const name = labels[viewId] || viewId;
    return `You do not have access to ${name}.`;
}


