// =========================================================================
// Global Fetch Override for Network Errors
// =========================================================================
const originalFetch = window.fetch;
window.fetch = async function(...args) {
    try {
        const response = await originalFetch(...args);
        return response;
    } catch (error) {
        showConnectionErrorUI();
        throw new Error('NicheCP_Network_Error');
    }
};

function showConnectionErrorUI() {
    if (document.getElementById('nichecp-connection-error')) return;
    const errorHTML = `
        <div id="nichecp-connection-error" style="position: fixed; top: 0; left: 0; width: 100vw; height: 100vh; background: rgba(0,0,0,0.8); backdrop-filter: blur(10px); z-index: 10000; display: flex; align-items: center; justify-content: center; flex-direction: column;">
            <div class="module-card" style="text-align: center; max-width: 400px;">
                <svg viewBox="0 0 24 24" fill="none" stroke="#ef4444" stroke-width="2" style="width: 48px; height: 48px; margin-bottom: 16px;"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>
                <h3 style="font-size: 20px; font-weight: 600; margin-bottom: 8px; color: var(--text-primary);">Unable to connect to the server.</h3>
                <p style="color: var(--text-muted); font-size: 14px; margin-bottom: 24px;">The backend service appears to be offline or unreachable.</p>
                <div style="display: flex; gap: 12px; justify-content: center;">
                    <button class="btn-magnetic" style="padding: 10px 20px; border: none; border-radius: 8px; cursor: pointer; color: #fff;" onclick="window.location.reload()">Retry</button>
                    <button class="btn-ghost" style="padding: 10px 20px; border: 1px solid rgba(255,255,255,0.1); border-radius: 8px; background: transparent; color: var(--text-primary); cursor: pointer;" onclick="document.getElementById('nichecp-connection-error').remove()">Check Backend Status</button>
                </div>
            </div>
        </div>
    `;
    document.body.insertAdjacentHTML('beforeend', errorHTML);
}

document.addEventListener('DOMContentLoaded', () => {
    // Auth Check Logic
    const urlParams = new URLSearchParams(window.location.search);
    const hashParams = new URLSearchParams(window.location.hash.substring(1));
    const tokenFromUrl = hashParams.get('token') || urlParams.get('token');
    if (tokenFromUrl) {
        localStorage.setItem('jwt_token', tokenFromUrl);
        window.history.replaceState({}, document.title, window.location.pathname);
    }

    const token = localStorage.getItem('jwt_token');

    // Render Global Glassmorphic Navigation
    renderGlobalNav(token);

    // Fetch and render homepage dynamic data if on the homepage
    if (document.getElementById('dynamic-upcoming-contests')) {
        initHomepageData(token);
    }

    // Theme switcher logic removed (NicheCP uses a strict premium dark theme).    // We only enforce login on the arena page. If we are on index.html, we don't redirect.
    // (Disabled for now so you can view the arena UI locally without logging in)
    /*
    if (!token && window.location.pathname.includes('arena.html')) {
        window.location.href = 'login.html';
        return;
    }
    */

    // Spotlight Effect for Cyber Cards (Landing Page)
    const cards = document.querySelectorAll('.spotlight-card');
    cards.forEach(card => {
        card.addEventListener('mousemove', e => {
            const rect = card.getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;
            card.style.setProperty('--mouse-x', `${x}px`);
            card.style.setProperty('--mouse-y', `${y}px`);
        });
    });

    // Arena Logic (Only runs if elements exist)
    const submitBtn = document.getElementById('submitBtn');
    const runBtn = document.getElementById('runBtn');
    const languageSelect = document.getElementById('language');
    const statusMessage = document.getElementById('statusMessage');
    const customInput = document.getElementById('customInput');
    const terminalOutput = document.getElementById('terminalOutput');

    // Initialize Monaco Editor if we are on arena.html
    let monacoEditor = null;
    const editorContainer = document.getElementById('codeEditorContainer');
    if (editorContainer && window.require) {
        require.config({ paths: { 'vs': 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.44.0/min/vs' }});
        require(['vs/editor/editor.main'], function() {
            monaco.editor.defineTheme('nichecp-dark', {
                base: 'vs-dark',
                inherit: true,
                rules: [
                    { background: '060606' },
                    { token: 'keyword', foreground: '3B82F6' },
                    { token: 'string', foreground: '10B981' },
                    { token: 'comment', foreground: '6B7280', fontStyle: 'italic' },
                    { token: 'number', foreground: 'F59E0B' }
                ],
                colors: {
                    'editor.background': '#060606',
                    'editor.foreground': '#F3F4F6',
                    'editor.lineHighlightBackground': '#111115',
                    'editorLineNumber.foreground': '#4B5563',
                    'editorIndentGuide.background': '#1F2937',
                    'editorSuggestWidget.background': '#0D0D12',
                    'editorSuggestWidget.border': '#1F2937'
                }
            });

            monacoEditor = monaco.editor.create(editorContainer, {
                value: 'def solve(n):\n    # Write your logic here\n    pass',
                language: 'python',
                theme: 'nichecp-dark',
                automaticLayout: true,
                minimap: { enabled: false },
                fontSize: 14,
                fontFamily: "'JetBrains Mono', 'Courier New', monospace"
            });
            
            // Handle language change
            languageSelect.addEventListener('change', (e) => {
                let lang = e.target.value;
                if (lang === 'cpp') lang = 'cpp';
                else if (lang === 'go') lang = 'go';
                else lang = 'python';
                monaco.editor.setModelLanguage(monacoEditor.getModel(), lang);
            });
        });
    }

    if (runBtn && submitBtn) {
        runBtn.addEventListener('click', async () => {
            const code = monacoEditor ? monacoEditor.getValue() : '';
            const language = languageSelect.value;
            
            if (!code.trim()) {
                showStatus('Please enter some code.', 'error');
                return;
            }

            let runs = [];
            if (customInput && customInput.value.trim()) {
                runs.push({ input: customInput.value, expected: null });
            } else if (window.currentProblemSamples && window.currentProblemSamples.length > 0) {
                runs = window.currentProblemSamples.map(s => ({ input: s.input, expected: s.expected_output }));
            } else {
                runs.push({ input: '', expected: null });
            }

            runBtn.disabled = true;
            runBtn.textContent = 'Running...';
            terminalOutput.value = '';

            try {
                // Auto switch to Test Result tab
                const resTab = document.querySelector('.tc-tab[data-target="pane-results"]');
                if (resTab) resTab.click();

                for (let i = 0; i < runs.length; i++) {
                    terminalOutput.value += `=== Test ${i+1} ===\n`;
                    const response = await fetch('/api/run', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ code, language, input: runs[i].input })
                    });

                    const data = await response.json();

                    if (response.ok) {
                        const out = data.output || 'No output';
                        terminalOutput.value += out + '\n';
                        
                        if (runs[i].expected !== null) {
                            if (out.trim() === runs[i].expected.trim()) {
                                terminalOutput.value += `\n[Result]: ✅ Passed\n\n`;
                            } else {
                                terminalOutput.value += `\n[Result]: ❌ Failed\n[Expected]: ${runs[i].expected}\n\n`;
                            }
                        }
                        
                        if (data.stderr) {
                            terminalOutput.value += '[Errors]:\n' + data.stderr + '\n\n';
                        }
                    } else {
                        terminalOutput.value += `Error: ${data.error}\n\n`;
                    }
                }
            } catch (err) {
                terminalOutput.value += 'Failed to connect to backend server.\n';
            } finally {
                runBtn.disabled = false;
                runBtn.textContent = 'Run Code';
            }
        });

        submitBtn.addEventListener('click', async () => {
            if (!token) {
                showStatus('You must be signed in to submit code to the Judge.', 'error');
                return;
            }

            const code = monacoEditor ? monacoEditor.getValue() : '';
            const language = languageSelect.value;

            if (!code.trim()) {
                showStatus('Please enter some code.', 'error');
                return;
            }

            // Show pending state
            showStatus('Submitting to Judge...', 'pending');
            submitBtn.disabled = true;
            submitBtn.textContent = 'Processing...';

            try {
                const contestId = new URLSearchParams(window.location.search).get('contest_id');
                const submitBody = {
                    problem_id: parseInt(new URLSearchParams(window.location.search).get('problem_id')) || 1,
                    code: code,
                    language: language
                };
                if (contestId) {
                    submitBody.contest_id = parseInt(contestId);
                }

                // This URL points to our Go Backend
                const response = await fetch('/api/submit', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${token}`
                    },
                    body: JSON.stringify(submitBody)
                });

                const data = await response.json();

                if (response.ok) {
                    showStatus(`Submission Queued! ID: ${data.submission_id}. Waiting for execution...`, 'pending');
                    
                    // Poll for status
                    let pollCount = 0;
                    const maxPolls = 100; // 100 seconds max wait
                    const pollInterval = setInterval(async () => {
                        pollCount++;
                        if (pollCount > maxPolls) {
                            clearInterval(pollInterval);
                            showStatus('Execution timed out. Please try again later.', 'error');
                            submitBtn.disabled = false;
                            submitBtn.textContent = 'Submit Code';
                            return;
                        }

                        try {
                            const statusRes = await fetch(`/api/submissions/${data.submission_id}`, {
                                headers: {
                                    'Authorization': `Bearer ${token}`
                                }
                            });
                            if (statusRes.ok) {
                                const statusData = await statusRes.json();
                                if (statusData.status !== 'PENDING') {
                                    clearInterval(pollInterval);
                                    let statusType = 'error';
                                    let color = '#ef4444';
                                    let icon = `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>`;
                                    let text = statusData.status.replace(/_/g, ' ');

                                    if (statusData.status === 'ACCEPTED') {
                                        statusType = 'success';
                                        color = '#10b981';
                                        icon = `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2"><path d="M22 11.08V12a10.08 10.08 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>`;
                                    } else if (statusData.status === 'TIME_LIMIT_EXCEEDED' || statusData.status === 'MEMORY_LIMIT_EXCEEDED') {
                                        statusType = 'warning';
                                        color = '#f59e0b';
                                        icon = `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg>`;
                                    }
                                    
                                    // Update styling of the panel
                                    const verdictPanel = document.getElementById('verdictPanel');
                                    if (verdictPanel) {
                                        verdictPanel.style.display = 'flex';
                                        verdictPanel.style.borderColor = color;
                                        document.getElementById('verdictIcon').innerHTML = icon;
                                        const vText = document.getElementById('verdictText');
                                        vText.textContent = text;
                                        vText.style.color = color;
                                        document.getElementById('verdictSubtext').textContent = `Runtime: ${statusData.execution_time_ms || 0}ms | Memory: ~0MB`;
                                    }
                                    
                                    showStatus(`Submission Finished!`, 'success');
                                    submitBtn.disabled = false;
                                    submitBtn.textContent = 'Submit Code';
                                }
                            }
                        } catch (e) {
                            clearInterval(pollInterval);
                            showStatus(`Status Check Error: ${e.message}`, 'error');
                            submitBtn.disabled = false;
                            submitBtn.textContent = 'Submit Code';
                        }
                    }, 1000);
                    return; // Don't re-enable button yet
                } else {
                    showStatus(`Judge Error: ${data.error}`, 'error');
                }
            } catch (err) {
                console.error("Submit Fetch Error:", err);
                showStatus(`Network Error: ${err.message}. Is backend running?`, 'error');
            } finally {
                // If it didn't return early due to polling, re-enable
                if (submitBtn.textContent !== 'Processing...') {
                    submitBtn.disabled = false;
                    submitBtn.textContent = 'Submit Code';
                }
            }
        });

        function showStatus(message, type) {
            statusMessage.textContent = message;
            statusMessage.className = `status-banner show ${type}`;
            
            // Auto-hide the banner after 4 seconds if it's a final result
            if (type !== 'pending') {
                setTimeout(() => {
                    statusMessage.classList.remove('show');
                }, 4000);
            }
        }
    }
});

// ==========================================
// ==========================================
// Homepage Data Fetcher
// ==========================================
async function initHomepageData(token) {
    try {
        // Fetch Upcoming Contests
        const contestsRes = await fetch('/api/public/upcoming-contests');
        const contestsContainer = document.getElementById('dynamic-upcoming-contests');
        if (contestsRes.ok) {
            const contests = await contestsRes.json();
            if (!contests || contests.length === 0) {
                contestsContainer.innerHTML = `<div class="mobile-table-card" style="padding: 40px 24px; text-align: center; color: var(--text-muted); font-size: 14px;">No upcoming contests scheduled at the moment.</div>`;
            } else {
                let html = '';
                contests.forEach(c => {
                    let statusColor = c.status === 'RUNNING' ? 'var(--primary-accent)' : 'var(--text-muted)';
                    let actionBtn = `<a href="contest_arena.html?id=${c.id}" class="btn-ghost" style="padding: 6px 12px; font-size: 12px;">View</a>`;
                    
                    html += `
                        <div class="mobile-table-row" style="display: grid; grid-template-columns: 2fr 1fr 1fr 1fr 1fr 1fr; gap: 16px; padding: 16px 24px; border-bottom: 1px solid rgba(255,255,255,0.05); align-items: center;">
                            <div style="font-weight: 500;">${c.title}</div>
                            <div><span class="tag tag-primary">${c.type}</span></div>
                            <div style="color: var(--text-muted); font-size: 13px;">${new Date(c.start_time).toLocaleString()}</div>
                            <div style="color: var(--text-muted); font-size: 13px;">${c.duration_minutes}m</div>
                            <div style="color: ${statusColor}; font-weight: 500; font-size: 12px;">${c.status}</div>
                            <div>${actionBtn}</div>
                        </div>
                    `;
                });
                contestsContainer.innerHTML = html;
            }
        } else {
            contestsContainer.innerHTML = `<div class="mobile-table-card" style="padding: 40px 24px; text-align: center; color: #ef4444; font-size: 14px;">Failed to load contests.</div>`;
        }

        // Fetch Recent Problems
        const problemsRes = await fetch('/api/public/recent-problems');
        const problemsContainer = document.getElementById('dynamic-recent-problems');
        if (problemsRes.ok) {
            const problems = await problemsRes.json();
            if (!problems || problems.length === 0) {
                problemsContainer.innerHTML = `<div class="mobile-table-card" style="padding: 40px 24px; text-align: center; color: var(--text-muted); font-size: 14px;">No problems available.</div>`;
            } else {
                let html = '';
                problems.forEach(p => {
                    let difficultyColor = p.difficulty === 'Easy' ? '#10b981' : p.difficulty === 'Medium' ? '#f59e0b' : '#ef4444';
                    html += `
                        <div class="mobile-table-row" style="display: grid; grid-template-columns: 2fr 1fr 2fr 1fr; gap: 16px; padding: 16px 24px; border-bottom: 1px solid rgba(255,255,255,0.05); align-items: center; cursor: pointer; transition: background 0.2s;" onclick="window.location.href='arena.html?problem_id=${p.id}'" onmouseover="this.style.background='rgba(255,255,255,0.02)'" onmouseout="this.style.background='transparent'">
                            <div style="font-weight: 500;">${p.title}</div>
                            <div style="color: ${difficultyColor}; font-weight: 500;">${p.difficulty}</div>
                            <div><span class="tag" style="background: rgba(255,255,255,0.05);">${p.tags || 'General'}</span></div>
                            <div style="color: var(--text-muted); font-size: 12px;">${new Date(p.created_at).toLocaleDateString()}</div>
                        </div>
                    `;
                });
                problemsContainer.innerHTML = html;
            }
        } else {
            problemsContainer.innerHTML = `<div class="mobile-table-card" style="padding: 40px 24px; text-align: center; color: #ef4444; font-size: 14px;">Failed to load problems.</div>`;
        }

        // Fetch User Statistics if logged in
        if (token) {
            const statsSection = document.getElementById('user-statistics-section');
            if (statsSection) statsSection.style.display = 'block';

            const statsRes = await fetch('/api/profile/stats', {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (statsRes.ok) {
                const stats = await statsRes.json();
                document.getElementById('stat-solved').innerText = stats.problems_solved;
                document.getElementById('stat-contests').innerText = stats.contest_participation;
                document.getElementById('stat-rating').innerText = stats.rating;
                document.getElementById('stat-submissions').innerText = stats.submission_count;
            }
        }
    } catch (err) {
        console.error("Failed to load homepage data:", err);
    }
}

// ==========================================
// ==========================================
// Global Navigation Renderer
// ==========================================
async function renderGlobalNav(token) {
    const container = document.getElementById('global-nav-container');
    if (!container) return;

    let role = '';
    let avatarUrl = '';
    let userEmail = '';
    let userName = '';
    
    if (token) {
        try {
            const res = await fetch('/api/profile', {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (res.ok) {
                const data = await res.json();
                role = data.role;
                avatarUrl = data.profile_picture_url || '';
                userEmail = data.email || '';
                userName = data.name || '';
            }
        } catch (e) {
            console.error("Failed to fetch profile for nav:", e);
        }
    }

    let currentPath = window.location.pathname.split('/').pop() || 'index.html';
    if (currentPath === '') currentPath = 'index.html';

    let navLinksHTML = '';
    let closedRightSideHTML = '';
    let openedDrawerBottomHTML = '';

    if (token) {
        // Authenticated Navbar
        const authNavItems = [
            { label: 'Dashboard', path: 'index.html' },
            { label: 'Problems', path: 'arena.html' },
            { label: 'Contests', path: 'contests.html' },
            { label: 'Practice', path: 'practice.html' },
            { label: 'Learn', path: 'learn.html' },
            { label: 'Blogs', path: 'blogs.html' }
        ];

        const normRole = (role || '').trim().toLowerCase();
        const isSuperAdminEmail = (userEmail || '').trim().toLowerCase() === 'dharunkaarthick07@gmail.com';
        if (normRole === 'admin' || normRole === 'superadmin' || isSuperAdminEmail) {
            authNavItems.push({ label: 'Admin', path: 'admin.html' });
        }

        navLinksHTML = authNavItems.map(item => {
            const isActive = (currentPath === item.path) ? 'active' : '';
            return `<a href="${item.path}" class="glass-nav-item ${isActive}">${item.label}</a>`;
        }).join('');

        // Closed Bar: Interactive Avatar Trigger & Smooth Dropdown Menu
        const normRole = (role || '').trim().toLowerCase();
        const isSuperAdminEmail = (userEmail || '').trim().toLowerCase() === 'dharunkaarthick07@gmail.com';
        const isAdmin = normRole === 'admin' || normRole === 'superadmin' || isSuperAdminEmail;

        closedRightSideHTML = `
            <div class="nav-profile-dropdown-container" id="navProfileDropdownContainer">
                <button type="button" class="nav-profile-trigger" id="navProfileTrigger" aria-label="User menu" aria-expanded="false" aria-haspopup="true">
                    <img src="${avatarUrl || 'https://api.dicebear.com/10.x/critters/svg?seed=Felix'}" class="nav-avatar" alt="${userName || 'User'} Profile">
                    <svg class="nav-avatar-chevron" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                        <polyline points="6 9 12 15 18 9"></polyline>
                    </svg>
                </button>
                <div class="nav-profile-dropdown" id="navProfileDropdown">
                    <div class="nav-profile-header">
                        <img src="${avatarUrl || 'https://api.dicebear.com/10.x/critters/svg?seed=Felix'}" class="nav-profile-header-avatar" alt="Avatar">
                        <div class="nav-profile-header-info">
                            <div class="nav-profile-name">${userName || 'User'}</div>
                            <div class="nav-profile-email">${userEmail || 'Member'}</div>
                        </div>
                        ${isAdmin ? `<span class="nav-profile-role-badge">ADMIN</span>` : `<span class="nav-profile-role-badge user">MEMBER</span>`}
                    </div>
                    <div class="nav-profile-divider"></div>
                    <a href="profile.html" class="nav-profile-item">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>
                        <span>Profile</span>
                    </a>
                    ${isAdmin ? `
                    <a href="admin.html" class="nav-profile-item">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg>
                        <span>Admin Portal</span>
                    </a>
                    ` : ''}
                    <div class="nav-profile-divider"></div>
                    <button type="button" class="nav-profile-item nav-profile-logout" onclick="window.handleLogout()">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path><polyline points="16 17 21 12 16 7"></polyline><line x1="21" y1="12" x2="9" y2="12"></line></svg>
                        <span>Logout</span>
                    </button>
                </div>
            </div>
        `;

        // Opened Drawer: User info & Logout button
        openedDrawerBottomHTML = `
            <div style="display: flex; align-items: center; gap: 12px; margin-bottom: 8px;">
                <img src="${avatarUrl || 'https://api.dicebear.com/10.x/critters/svg?seed=Felix'}" style="width: 40px; height: 40px; border-radius: 50%; object-fit: cover;" alt="Profile">
                <div style="text-align: left;">
                    <div style="font-weight: 600; font-size: 14px; color: #fff;">${userName || 'User'}</div>
                    <div style="font-size: 12px; color: var(--text-muted);">${userEmail}</div>
                </div>
            </div>
            <button class="btn-ghost" style="width: 100%; padding: 12px; display: flex; align-items: center; justify-content: center; gap: 8px; border: 1px solid rgba(239, 68, 68, 0.3); border-radius: 12px; color: #ef4444; background: rgba(239, 68, 68, 0.08); cursor: pointer;" onclick="window.handleLogout()">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path><polyline points="16 17 21 12 16 7"></polyline><line x1="21" y1="12" x2="9" y2="12"></line></svg>
                Logout
            </button>
        `;
    } else {
        // Unauthenticated Navbar
        const unauthNavItems = [
            { label: 'Home', path: 'index.html' },
            { label: 'Problems', path: 'arena.html' },
            { label: 'Contests', path: 'contests.html' },
            { label: 'Learn', path: 'learn.html' },
            { label: 'Blogs', path: 'blogs.html' },
            { label: 'About', path: 'about.html' }
        ];

        navLinksHTML = unauthNavItems.map(item => {
            const isActive = (currentPath === item.path) ? 'active' : '';
            return `<a href="${item.path}" class="glass-nav-item ${isActive}">${item.label}</a>`;
        }).join('');

        // Closed Bar: ONLY Login button
        closedRightSideHTML = `
            <a href="login.html" class="glass-nav-item" style="padding: 6px 16px; border-radius: 20px; background: rgba(255,255,255,0.08);">Login</a>
        `;

        // Opened Drawer: Login & Sign Up buttons
        openedDrawerBottomHTML = `
            <a href="login.html" class="glass-nav-item" style="width: 100%; text-align: center; padding: 12px; border-radius: 12px; background: rgba(255,255,255,0.08); font-size: 15px;">Login</a>
            <a href="register.html" class="btn-primary" style="width: 100%; text-align: center; padding: 12px; border-radius: 12px; text-decoration: none; font-size: 15px;">Sign Up</a>
        `;
    }

    container.innerHTML = `
        <div class="glass-nav fade-in-down">
            <a href="index.html" class="nav-brand">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="var(--primary-accent)" stroke-width="2" style="margin-right:8px;"><polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2"></polygon><line x1="12" y1="22" x2="12" y2="15.5"></line><polyline points="22 8.5 12 15.5 2 8.5"></polyline><polyline points="2 15.5 12 8.5 22 15.5"></polyline><line x1="12" y1="2" x2="12" y2="8.5"></line></svg>
                NicheCP
            </a>
            <div class="glass-nav-center">
                ${navLinksHTML}
            </div>
            <div class="glass-nav-right">
                <div style="display: flex; align-items: center; gap: 8px;">
                    ${closedRightSideHTML}
                </div>
                <button class="hamburger-btn" onclick="document.getElementById('mobileNavDrawer').classList.add('active')" aria-label="Open Menu">
                    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="3" y1="12" x2="21" y2="12"></line><line x1="3" y1="6" x2="21" y2="6"></line><line x1="3" y1="18" x2="21" y2="18"></line></svg>
                </button>
            </div>
        </div>
        <div class="mobile-nav-drawer" id="mobileNavDrawer">
            <button class="close-drawer-btn" onclick="document.getElementById('mobileNavDrawer').classList.remove('active')" aria-label="Close Menu">
                <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
            </button>
            <div style="display: flex; flex-direction: column; align-items: center; gap: 14px; width: 100%;">
                ${navLinksHTML}
            </div>
            <div style="display: flex; flex-direction: column; align-items: center; gap: 12px; margin-top: 24px; border-top: 1px solid rgba(255,255,255,0.1); padding-top: 20px; width: 85%;">
                ${openedDrawerBottomHTML}
            </div>
        </div>
    `;

    // Interactive click/tap handler & outside click listener for profile dropdown
    const trigger = document.getElementById('navProfileTrigger');
    const dropdownContainer = document.getElementById('navProfileDropdownContainer');
    if (trigger && dropdownContainer) {
        trigger.addEventListener('click', (e) => {
            e.stopPropagation();
            const isExpanded = dropdownContainer.classList.toggle('active');
            trigger.setAttribute('aria-expanded', isExpanded ? 'true' : 'false');
        });
        
        document.addEventListener('click', (e) => {
            if (!dropdownContainer.contains(e.target)) {
                dropdownContainer.classList.remove('active');
                trigger.setAttribute('aria-expanded', 'false');
            }
        });
    }
}
