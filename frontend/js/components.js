// =========================================================================
// NicheCP - Global Components Injector
// =========================================================================

// injectGlobalNav has been removed and replaced by app.js renderGlobalNav
// to prevent conflicting navbars.

window.handleLogout = function() {
    localStorage.removeItem('jwt_token');
    window.location.href = 'index.html';
};
