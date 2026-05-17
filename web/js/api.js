/**
 * web/js/api.js
 * Core networking and authentication handlers
 */

export function parseJwt(token) { 
    try { 
        return JSON.parse(atob(token.split('.')[1])); 
    } catch (e) { 
        return {}; 
    } 
}

export function performLogout() { 
    // SECURITY FIX: Only attempt reload if a token was actually present
    if (localStorage.getItem('sre-jwt')) {
        localStorage.removeItem('sre-jwt'); 
        location.reload(); 
    } else {
        const loginScreen = document.getElementById('login-screen');
        const appContainer = document.getElementById('app-container');
        const pwdScreen = document.getElementById('force-password-screen');
        
        if (loginScreen) loginScreen.style.display = 'flex';
        if (appContainer) appContainer.style.display = 'none';
        if (pwdScreen) pwdScreen.style.display = 'none';
    }
}

export function fetchWithAuth(url, opts = {}) {
    const token = localStorage.getItem('sre-jwt');
    
    // SECURITY FIX: Prevent fetch if no token is found
    if (!token) {
        performLogout();
        return Promise.reject('No authentication token found.');
    }

    opts.headers = { 
        ...opts.headers, 
        'Authorization': `Bearer ${token}`, 
        'Content-Type': 'application/json' 
    };
    
    return fetch(url, opts).then(res => { 
        if (res.status === 401) performLogout(); 
        return res; 
    });
}