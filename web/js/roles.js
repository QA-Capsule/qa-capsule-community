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



export function applyRoleVisibility(role) {

    document.querySelectorAll('.role-admin').forEach(el => {

        el.style.display = role === 'admin' ? '' : 'none';

    });



    document.querySelectorAll('.admin-only').forEach(el => {

        el.style.display = role === 'admin' ? '' : 'none';

    });



    document.querySelectorAll('.role-workspace').forEach(el => {

        el.style.display = canManageTeams(role) ? '' : 'none';

    });



    document.querySelectorAll('.role-manager-only').forEach(el => {

        el.style.display = role === 'manager' ? '' : 'none';

    });



    document.querySelectorAll('.role-manager').forEach(el => {

        el.style.display = role === 'manager' ? '' : 'none';

    });



    document.querySelectorAll('.role-operator').forEach(el => {

        el.style.display = (role === 'operator' || role === 'manager') ? '' : 'none';

    });



    document.querySelectorAll('.role-non-admin').forEach(el => {

        el.style.display = role === 'admin' ? 'none' : '';

    });



    document.querySelectorAll('.role-chart-studio').forEach(el => {

        el.style.display = canAccessChartStudio(role) ? '' : 'none';

    });

}


