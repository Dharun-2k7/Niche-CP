// =========================================================================
// NicheCP - Global Interactions (Command Palette)
// =========================================================================

document.addEventListener('DOMContentLoaded', () => {
    initCommandPalette();
});

/* ================== 1. COMMAND PALETTE (Ctrl + K) ================== */
function initCommandPalette() {
    // Inject Command Palette HTML
    const paletteHTML = `
        <div id="command-palette-backdrop" style="display: none; position: fixed; top: 0; left: 0; width: 100vw; height: 100vh; background: rgba(0,0,0,0.5); backdrop-filter: blur(4px); z-index: 9999; align-items: flex-start; justify-content: center; padding-top: 15vh; opacity: 0; transition: opacity 0.2s;">
            <div id="command-palette-modal" style="background: var(--card-bg); border: 1px solid var(--card-border); width: 100%; max-width: 600px; border-radius: 12px; overflow: hidden; box-shadow: 0 20px 40px rgba(0,0,0,0.5); transform: scale(0.95); transition: transform 0.2s;">
                <div style="padding: 16px; border-bottom: 1px solid var(--card-border); display: flex; align-items: center;">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" stroke-width="2" style="margin-right: 12px;"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
                    <input type="text" id="command-palette-input" placeholder="Search problems, contests, or users..." style="background: transparent; border: none; color: var(--text-primary); font-family: 'Inter', sans-serif; font-size: 16px; width: 100%; outline: none;">
                    <span style="font-size: 12px; color: var(--text-muted); background: rgba(255,255,255,0.05); padding: 4px 8px; border-radius: 4px;">ESC</span>
                </div>
                <div id="command-palette-results" style="max-height: 300px; overflow-y: auto; padding: 8px 0;">
                    <div style="padding: 8px 16px; font-size: 12px; color: var(--text-muted); text-transform: uppercase; font-weight: 600;">Suggestions</div>
                    <a href="problems.html" class="cp-result-item" style="display: flex; align-items: center; padding: 12px 16px; text-decoration: none; color: var(--text-primary); transition: background 0.1s;">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" stroke-width="2" style="margin-right: 12px;"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline><line x1="16" y1="13" x2="8" y2="13"></line><line x1="16" y1="17" x2="8" y2="17"></line><polyline points="10 9 9 9 8 9"></polyline></svg>
                        Browse Problemset
                    </a>
                    <a href="contests.html" class="cp-result-item" style="display: flex; align-items: center; padding: 12px 16px; text-decoration: none; color: var(--text-primary); transition: background 0.1s;">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" stroke-width="2" style="margin-right: 12px;"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg>
                        View Upcoming Contests
                    </a>
                    <a href="profile.html" class="cp-result-item" style="display: flex; align-items: center; padding: 12px 16px; text-decoration: none; color: var(--text-primary); transition: background 0.1s;">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" stroke-width="2" style="margin-right: 12px;"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>
                        Go to Profile
                    </a>
                </div>
            </div>
        </div>
    `;
    
    document.body.insertAdjacentHTML('beforeend', paletteHTML);
    
    const backdrop = document.getElementById('command-palette-backdrop');
    const modal = document.getElementById('command-palette-modal');
    const input = document.getElementById('command-palette-input');
    let isOpen = false;

    // Hover styles for results (since injected dynamically)
    const style = document.createElement('style');
    style.innerHTML = `
        .cp-result-item:hover { background: rgba(255,255,255,0.05); }
    `;
    document.head.appendChild(style);

    function openPalette() {
        isOpen = true;
        backdrop.style.display = 'flex';
        // Trigger reflow
        void backdrop.offsetWidth;
        backdrop.style.opacity = '1';
        modal.style.transform = 'scale(1)';
        input.value = '';
        input.focus();
    }

    function closePalette() {
        isOpen = false;
        backdrop.style.opacity = '0';
        modal.style.transform = 'scale(0.95)';
        setTimeout(() => {
            if (!isOpen) backdrop.style.display = 'none';
        }, 200);
    }

    // Event Listeners
    document.addEventListener('keydown', (e) => {
        if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
            e.preventDefault();
            isOpen ? closePalette() : openPalette();
        }
        if (e.key === 'Escape' && isOpen) {
            closePalette();
        }
    });

    backdrop.addEventListener('click', (e) => {
        if (e.target === backdrop) closePalette();
    });
}

