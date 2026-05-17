/**
 * web/js/iam.js
 * Identity & Access Management (Users, Organizations, Teams)
 */
import { fetchWithAuth, parseJwt } from './api.js';
import { notify, showConfirmModal, showPromptModal } from './ui.js';

export let allUsers = [];
export let allTeamsFlatList = [];
let currentSelectedOrgId = null;
let currentlyManagedUser = null;

export function loadOrganizations() {
    fetchWithAuth(`/api/teams?_ts=${Date.now()}`)
        .then(res => res.json())
        .then(data => {
            const treeContainer = document.getElementById('organization-tree');
            if (!data || data.length === 0) {
                treeContainer.innerHTML = "<p>No organizations found.</p>";
                return;
            }

            const treeMap = {};
            const roots = [];
            data.forEach(node => { treeMap[node.id] = { ...node, children: [] }; });
            data.forEach(node => {
                if (node.parent_id === null || node.parent_id === 0) {
                    roots.push(treeMap[node.id]);
                } else if (treeMap[node.parent_id]) {
                    treeMap[node.parent_id].children.push(treeMap[node.id]);
                }
            });

            treeContainer.innerHTML = roots.map(root => renderTreeNode(root)).join('');
            if (roots.length > 0) selectOrg(roots[0].id, roots[0].name);
        })
        .catch(err => notify("Failed to load directory", "error"));
}

function renderTreeNode(node) {
    const hasChildren = node.children && node.children.length > 0;
    const toggleIcon = hasChildren ? `<span class="tree-toggle" onclick="toggleTree(event, 'tree-children-${node.id}')">▼</span>` : `<span style="width:20px; display:inline-block;"></span>`;

    let html = `
    <div class="tree-node" id="node-container-${node.id}" style="${!node.parent_id ? 'margin-left: 0; border-left: none; padding-left: 0;' : ''}">
        <div style="display:flex; align-items:center;">
            ${toggleIcon}
            <div class="tree-item" id="org-node-${node.id}" onclick="window.selectOrg(${node.id}, '${node.name}')">
                <svg style="width:16px;height:16px;stroke:currentColor;fill:none;flex-shrink:0;" viewBox="0 0 24 24" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path></svg>
                ${node.name}
            </div>
        </div>`;

    if (hasChildren) {
        html += `<div id="tree-children-${node.id}" style="display:block;">` + node.children.map(child => renderTreeNode(child)).join('') + `</div>`;
    }
    html += `</div>`;
    return html;
}

export function toggleTree(e, id) {
    e.stopPropagation();
    const el = document.getElementById(id);
    const icon = e.target;
    if (el.style.display === 'none') {
        el.style.display = 'block';
        icon.innerText = '▼';
    } else {
        el.style.display = 'none';
        icon.innerText = '▶';
    }
}

export function selectOrg(id, name) {
    currentSelectedOrgId = id;
    document.querySelectorAll('.tree-item').forEach(el => el.classList.remove('active'));
    const selectedEl = document.getElementById(`org-node-${id}`);
    if (selectedEl) selectedEl.classList.add('active');

    document.getElementById('org-management-panel').style.display = 'block';
    document.getElementById('org-selected-name').innerText = name;
    document.getElementById('org-selected-id').innerText = `Internal DB ID: ${id}`;

    loadOrgMembers(id);
}

export function promptRenameGroup() {
    if (!currentSelectedOrgId) return;
    const currentName = document.getElementById('org-selected-name').innerText;

    showPromptModal("Rename Group", `Enter a new name for '${currentName}':`, currentName, function (newName) {
        if (newName === currentName || !newName) return;

        fetchWithAuth('/api/teams', {
            method: 'PUT',
            body: JSON.stringify({ id: currentSelectedOrgId, name: newName })
        }).then(res => {
            if (res.ok) { notify("Group renamed successfully!", "success"); loadOrganizations(); }
            else notify("Failed to rename group. Name might exist.", "error");
        });
    });
}

export function loadOrgMembers(orgId) {
    fetchWithAuth(`/api/teams/members?team_id=${orgId}&_ts=${Date.now()}`)
        .then(res => res.json())
        .then(members => {
            const tbody = document.getElementById('org-members-list');
            if (!members || members.length === 0) {
                tbody.innerHTML = '<tr><td colspan="4" style="text-align:center; padding:20px; opacity:0.5;">No members assigned.</td></tr>';
                return;
            }
            tbody.innerHTML = members.map(m => {
                let roleColor = m.team_role === 'team_admin' ? '#d33833' : (m.team_role === 'team_operator' ? '#d29922' : '#58a6ff');
                return `
                <tr style="border-bottom:1px solid var(--border-main);">
                    <td style="padding:12px 10px;"><strong>${m.fullname}</strong><br><small style="opacity:0.6;">${m.username}</small></td>
                    <td><span style="border: 1px solid ${roleColor}; color: ${roleColor}; padding: 3px 8px; border-radius: 12px; font-size: 10px; font-weight: bold; text-transform:uppercase;">${m.team_role.replace('team_', '')}</span></td>
                    <td><span style="opacity:0.5; font-size: 11px; text-transform:uppercase;">${m.global_role}</span></td>
                    <td style="text-align:right;">
                        <button class="btn-secondary" style="font-size:10px; padding:3px 6px; color:#ff7b72; border-color:#ff7b72;" onclick="window.removeUserFromOrg('${m.username}', ${orgId})">REMOVE</button>
                    </td>
                </tr>`;
            }).join('');
        });
}

export function removeUserFromOrg(username, teamId) {
    showConfirmModal("Remove User?", `Are you sure you want to remove '${username}' from this group?`, "warning", function () {
        fetchWithAuth(`/api/teams/members?username=${username}&team_id=${teamId}`, { method: 'DELETE' })
            .then(res => {
                if (res.ok) { notify("User removed successfully", "success"); loadOrgMembers(teamId); }
                else notify("Failed to remove user", "error");
            });
    });
}

export function promptCreateSubGroup() {
    if (!currentSelectedOrgId) return notify("Please select a parent group first.", "error");
    showPromptModal("Create Sub-Group", "Enter the name of the new sub-group:", "e.g. Backend Squad", function (name) {
        fetchWithAuth('/api/teams', { method: 'POST', body: JSON.stringify({ name: name, parent_id: currentSelectedOrgId }) })
            .then(res => {
                if (res.ok) { notify("Sub-group created!", "success"); loadOrganizations(); }
                else notify("Failed to create group. Name might exist.", "error");
            });
    });
}

export function promptDeleteGroup() {
    if (!currentSelectedOrgId) return;
    showConfirmModal("Delete Branch?", "WARNING: Deleting this group will ALSO DELETE ALL of its Sub-Groups. Are you absolutely sure?", "danger", function () {
        fetchWithAuth(`/api/teams?id=${currentSelectedOrgId}`, { method: 'DELETE' })
            .then(res => {
                if (res.ok) { notify("Branch deleted successfully.", "success"); document.getElementById('org-management-panel').style.display = 'none'; loadOrganizations(); }
                else notify("Cannot delete Root Organization or error occurred.", "error");
            });
    });
}

export function handleUserSearch() {
    const val = document.getElementById('org-add-user-email').value.toLowerCase();
    const list = document.getElementById('user-autocomplete-list');
    list.innerHTML = '';
    if (!val) { list.style.display = 'none'; return; }

    const matches = allUsers.filter(u => u.username.toLowerCase().includes(val) || (u.fullname && u.fullname.toLowerCase().includes(val)));
    if (matches.length === 0) { list.style.display = 'none'; return; }

    matches.forEach(m => {
        const div = document.createElement('div');
        div.innerHTML = `<strong>${m.fullname}</strong> <span style="opacity:0.6">(${m.username})</span>`;
        div.onclick = () => {
            document.getElementById('org-add-user-email').value = m.username;
            list.style.display = 'none';
        };
        list.appendChild(div);
    });
    list.style.display = 'block';
}

export function assignUserToGroup() {
    if (!currentSelectedOrgId) return;
    const username = document.getElementById('org-add-user-email').value;
    const role = document.getElementById('org-add-user-role').value;
    if (!username) return notify("Please enter an email", "error");

    fetchWithAuth('/api/teams/members', { method: 'POST', body: JSON.stringify({ username: username.trim(), team_id: currentSelectedOrgId, team_role: role }) })
        .then(res => {
            if (res.ok) { notify("User assigned successfully!", "success"); document.getElementById('org-add-user-email').value = ''; loadOrgMembers(currentSelectedOrgId); }
            else if (res.status === 404) notify("User does not exist in the system.", "error");
            else notify("Assignment failed.", "error");
        });
}

export function openUserTeamsModal(username) {
    currentlyManagedUser = username;
    document.getElementById('manage-teams-username').innerText = username;
    document.getElementById('user-add-team-input').value = '';
    document.getElementById('user-teams-modal').style.display = 'flex';

    fetchWithAuth(`/api/teams?_ts=${Date.now()}`).then(res => res.json()).then(data => { allTeamsFlatList = data || []; });
    refreshUserTeamsModal();
}

export function closeUserTeamsModal() {
    document.getElementById('user-teams-modal').style.display = 'none';
}

export function refreshUserTeamsModal() {
    fetchWithAuth(`/api/users/teams?username=${currentlyManagedUser}&_ts=${Date.now()}`)
        .then(res => res.json())
        .then(teams => {
            const tbody = document.getElementById('user-teams-list');
            if (!teams || teams.length === 0) {
                tbody.innerHTML = '<tr><td style="text-align:center; padding:20px; opacity:0.5; color:#8b949e;">User is not assigned to any groups.</td></tr>';
                return;
            }
            tbody.innerHTML = teams.map(t => {
                let roleColor = t.role === 'team_admin' ? '#d33833' : (t.role === 'team_operator' ? '#d29922' : '#58a6ff');
                return `
            <tr style="border-bottom:1px solid #30363d;">
                <td style="padding:10px 15px; font-size:14px; font-weight:bold; color:#c9d1d9;">${t.name}</td>
                <td style="padding:10px 15px;"><span style="border: 1px solid ${roleColor}; color: ${roleColor}; padding: 2px 6px; border-radius: 12px; font-size: 9px; font-weight: bold; text-transform:uppercase;">${t.role.replace('team_', '')}</span></td>
                <td style="padding:10px 15px; text-align:right;">
                    <button class="btn-secondary" style="font-size:10px; padding:4px 8px; color:#ff7b72; border-color:#ff7b72;" onclick="window.removeUserFromTeamModal(${t.id})">REMOVE</button>
                </td>
            </tr>`;
            }).join('');
        });
}

export function removeUserFromTeamModal(teamId) {
    fetchWithAuth(`/api/teams/members?username=${currentlyManagedUser}&team_id=${teamId}`, { method: 'DELETE' })
        .then(res => {
            if (res.ok) { notify("Removed from group", "success"); refreshUserTeamsModal(); }
            else notify("Failed to remove", "error");
        });
}

export function handleTeamSearch() {
    const val = document.getElementById('user-add-team-input').value.toLowerCase();
    const list = document.getElementById('team-autocomplete-list');
    list.innerHTML = '';

    if (!val) { list.style.display = 'none'; return; }

    const matches = allTeamsFlatList.filter(t => t.name.toLowerCase().includes(val));
    if (matches.length === 0) { list.style.display = 'none'; return; }

    matches.forEach(m => {
        const div = document.createElement('div');
        div.innerHTML = `<span style="color:#58a6ff; margin-right:5px;">#${m.id}</span> <strong>${m.name}</strong>`;
        div.onclick = () => {
            document.getElementById('user-add-team-input').value = '';
            list.style.display = 'none';
            assignUserToTeamFromModal(m.id);
        };
        list.appendChild(div);
    });
    list.style.display = 'block';
}

export function assignUserToTeamFromModal(teamId) {
    fetchWithAuth('/api/teams/members', { method: 'POST', body: JSON.stringify({ username: currentlyManagedUser, team_id: teamId }) })
        .then(res => {
            if (res.ok) { notify("Added to group!", "success"); refreshUserTeamsModal(); }
            else notify("Failed to assign group", "error");
        });
}

export function loadUsers() {
    fetchWithAuth(`/api/users?_ts=${Date.now()}`)
        .then(res => { if (!res.ok) throw new Error(); return res.json(); })
        .then(users => {
            allUsers = users || [];
            if (document.getElementById('view-management').classList.contains('active')) {
                renderUserTable(allUsers);
            }
        })
        .catch(() => notify("Failed to load users", "error"));
}

export function renderUserTable(users) {
    const tbody = document.getElementById('user-list-body');
    if (!tbody) return;

    const currentUser = parseJwt(localStorage.getItem('sre-jwt')).username;
    if (users.length === 0) { tbody.innerHTML = '<tr><td colspan="4" style="text-align:center;">No identities found.</td></tr>'; return; }

    tbody.innerHTML = users.map(u => {
        const isMe = u.username === currentUser;
        const disabledState = isMe ? 'disabled style="opacity:0.4; cursor:not-allowed;" title="You cannot modify yourself"' : '';
        return `
            <tr style="border-bottom:1px solid var(--border-main);">
                <td style="padding:15px 10px; color:${u.is_active ? 'var(--log-pass)' : 'var(--log-fatal)'}; font-weight:bold;">${u.is_active ? '● ACTIVE' : '○ DISABLED'}</td>
                <td><strong>${u.fullname || 'N/A'}</strong> ${isMe ? '<small>(YOU)</small>' : ''}<br><small style="opacity:0.6;">${u.username}</small></td>
                <td><small style="text-transform:uppercase;">${u.role}</small></td>
                <td style="text-align:right;">
                    <button class="btn-secondary" style="font-size:10px; border-color:#58a6ff; color:#58a6ff;" onclick="window.openUserTeamsModal('${u.username}')">TEAMS</button>
                    <button class="btn-secondary" style="font-size:10px;" onclick="window.adminResetPassword('${u.username}')" ${disabledState}>RESET PWD</button>
                    <button class="btn-secondary" style="font-size:10px; color:${u.is_active ? 'var(--log-fatal)' : 'var(--log-pass)'}" onclick="window.toggleUserStatus('${u.username}', ${u.is_active})" ${disabledState}>${u.is_active ? 'Disable' : 'Enable'}</button>
                    <button class="btn-primary" style="color:var(--log-fatal); border-color:var(--log-fatal); font-size:10px;" onclick="window.deleteUser('${u.username}')" ${disabledState}>DEL</button>
                </td></tr>`;
    }).join('');
}

export function filterUsers() {
    const q = document.getElementById('user-search').value.toLowerCase();
    const r = document.getElementById('role-filter').value;
    renderUserTable(allUsers.filter(u => (u.username.toLowerCase().includes(q) || (u.fullname && u.fullname.toLowerCase().includes(q))) && (r === 'all' || u.role === r)));
}

export function createUser() {
    const name = document.getElementById('new-user-fullname').value;
    const email = document.getElementById('new-user-name').value;
    if (!name || !email) return notify("Name and Email are required", "error");

    fetchWithAuth('/api/users', { method: 'POST', body: JSON.stringify({ username: email, fullname: name, role: document.getElementById('new-user-role').value }) })
        .then(res => {
            if (res.ok) {
                notify("Identity deployed!", "success");
                document.getElementById('new-user-fullname').value = '';
                document.getElementById('new-user-name').value = '';
                loadUsers();
            } else { notify("Failed to provision identity.", "error"); }
        });
}

export function toggleUserStatus(username, current) {
    fetchWithAuth('/api/users/status', { method: 'POST', body: JSON.stringify({ username, is_active: !current }) })
        .then(res => {
            if (res.ok) { notify("Status updated", "success"); loadUsers(); }
            else notify("Update failed", "error");
        });
}

export function adminResetPassword(username) {
    showConfirmModal("Reset Password?", `Are you sure you want to force a password reset for '${username}'?`, "warning", function () {
        fetchWithAuth('/api/users/reset-password', { method: 'POST', body: JSON.stringify({ username }) })
            .then(res => {
                if (res.ok) notify("New password emailed", "success");
                else notify("Reset failed", "error");
            });
    });
}

export function deleteUser(username) {
    showConfirmModal("Delete Identity?", `Are you absolutely sure you want to completely remove the user '${username}'? This action cannot be undone.`, "danger", function () {
        fetchWithAuth(`/api/users/delete?username=${username}`, { method: 'DELETE' }).then(res => {
            if (res.ok) { notify("Identity deleted", "success"); loadUsers(); }
            else notify("Deletion failed", "error");
        });
    });
}