// =========================================================================
// NicheCP - Global Components Injector
// =========================================================================

document.addEventListener('DOMContentLoaded', () => {
    injectGlobalNav();
});

function injectGlobalNav() {
    const navContainer = document.getElementById('global-nav-container');
    if (!navContainer) return;

    const token = localStorage.getItem('jwt_token');
    const isLoggedIn = !!token;
    
    // Check current page to set active state
    const currentPath = window.location.pathname;
    const isActive = (path) => currentPath.includes(path) ? 'active' : '';

    const authHTML = isLoggedIn ? `
        <div class="glass-nav-right">
            <div class="glass-nav-icon" style="margin-right: 12px;">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path><path d="M13.73 21a2 2 0 0 1-3.46 0"></path></svg>
                <span class="notification-badge">3</span>
            </div>
            <a href="profile.html"><img src="https://ui-avatars.com/api/?name=User&background=random" class="nav-avatar" alt="Profile"></a>
            <button onclick="handleLogout()" class="btn-ghost" style="padding: 6px 16px; font-size: 13px; border-radius: 20px; border-color: rgba(255,255,255,0.1); color: var(--text-muted); cursor: pointer; margin-left: 8px;">Logout</button>
        </div>
    ` : `
        <div class="glass-nav-right">
            <a href="login.html" class="glass-nav-item">Login</a>
            <a href="register.html" class="btn-magnetic" style="padding: 6px 16px; font-size: 13px; border-radius: 20px; text-decoration: none;">Sign Up</a>
        </div>
    `;

    const themeSwitcherHTML = `
        <div class="theme-switcher" style="display:flex; gap:8px; margin-right: 16px; align-items: center; padding: 4px 8px;">
            <div class="theme-dot active" data-set="quantum" style="cursor:pointer; width:12px; height:12px; border-radius:50%;" title="Quantum"></div>
            <div class="theme-dot" data-set="aurora" style="cursor:pointer; width:12px; height:12px; border-radius:50%;" title="Aurora"></div>
            <div class="theme-dot" data-set="solar" style="cursor:pointer; width:12px; height:12px; border-radius:50%;" title="Solar"></div>
        </div>
    `;

    const navHTML = `
        <nav class="glass-nav">
            <a href="index.html" class="nav-brand">Niche<span style="color: var(--primary-accent);">CP</span></a>
            <div class="glass-nav-center">
                <a href="index.html" class="glass-nav-item ${currentPath.endsWith('/') || currentPath.endsWith('index.html') ? 'active' : ''}">Home</a>
                <a href="contests.html" class="glass-nav-item ${isActive('contests.html')}">Contests</a>
                <a href="arena.html" class="glass-nav-item ${isActive('arena.html')}">Arena</a>
                <a href="problem.html" class="glass-nav-item ${isActive('problem.html')}">Problems</a>
            </div>
            <div style="display:flex; align-items:center;">
                ${themeSwitcherHTML}
                ${authHTML}
            </div>
        </nav>
    `;

    navContainer.innerHTML = navHTML;

    // Initialize theme switcher after injecting it
    if (typeof initThemeSwitcher === 'function') {
        initThemeSwitcher();
    }
}

window.handleLogout = function() {
    localStorage.removeItem('jwt_token');
    window.location.href = 'index.html';
};
