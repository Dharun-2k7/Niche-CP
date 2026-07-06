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

    // Theme Switcher Logic
    const themeBtns = document.querySelectorAll('.theme-dot');
    if (themeBtns.length > 0) {
        themeBtns.forEach(btn => {
            btn.addEventListener('click', () => {
                const theme = btn.getAttribute('data-set');
                document.body.setAttribute('data-theme', theme);
                themeBtns.forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                localStorage.setItem('nichecp_theme', theme);
            });
        });

        // Load saved theme
        const savedTheme = localStorage.getItem('nichecp_theme');
        if (savedTheme) {
            document.body.setAttribute('data-theme', savedTheme);
            const activeBtn = document.querySelector(`.theme-dot[data-set="${savedTheme}"]`);
            if (activeBtn) {
                themeBtns.forEach(b => b.classList.remove('active'));
                activeBtn.classList.add('active');
            }
        }
    }

    // We only enforce login on the arena page. If we are on index.html, we don't redirect.
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
            monacoEditor = monaco.editor.create(editorContainer, {
                value: 'def solve(n):\n    # Write your logic here\n    pass',
                language: 'python',
                theme: 'vs-dark',
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
                    const response = await fetch('http://localhost:8080/api/run', {
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
                const response = await fetch('http://localhost:8080/api/submit', {
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
                            const statusRes = await fetch(`http://localhost:8080/api/submissions/${data.submission_id}`, {
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
// Global Navigation Renderer
// ==========================================
async function renderGlobalNav(token) {
    const container = document.getElementById('global-nav-container');
    if (!container) return;

    let role = 'student';
    let avatarUrl = '';
    
    if (token) {
        try {
            // Hardcoding backend URL for MVP since we are rendering from static HTML
            const res = await fetch('http://localhost:8080/api/profile', {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (res.ok) {
                const data = await res.json();
                role = data.role;
                avatarUrl = data.profile_picture_url || '';
            }
        } catch (e) {
            console.error("Failed to fetch profile for nav:", e);
        }
    }

    const navItems = [
        { label: 'Home', path: 'index.html' },
        { label: 'Roadmaps', path: '#' },
        { label: 'Contests', path: 'contests.html' },
        { label: 'Problems', path: 'problem.html' },
        { label: 'Arena', path: 'arena.html' }
    ];

    if (role === 'admin' || role === 'superadmin') {
        navItems.push({ label: 'Admin', path: 'admin.html' });
    }

    let currentPath = window.location.pathname.split('/').pop() || 'index.html';
    if (currentPath === '') currentPath = 'index.html';

    const navLinksHTML = navItems.map(item => {
        const isActive = (currentPath === item.path) ? 'active' : '';
        return `<a href="${item.path}" class="glass-nav-item ${isActive}">${item.label}</a>`;
    }).join('');

    const rightSideHTML = token 
        ? `
            <div class="glass-nav-icon">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path><path d="M13.73 21a2 2 0 0 1-3.46 0"></path></svg>
                <div class="notification-badge">0</div>
            </div>
            <img src="${avatarUrl || 'https://api.dicebear.com/7.x/avataaars/svg?seed=fallback'}" class="nav-avatar" onclick="window.location.href='profile.html'" alt="Profile" title="Go to Profile">
          `
        : `<a href="login.html" class="glass-nav-item" style="background:var(--primary-accent); color:#000; padding: 6px 16px;">Sign In</a>`;

    container.innerHTML = `
        <div class="glass-nav">
            <a href="index.html" class="nav-brand">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-right:8px; color:var(--primary-accent);"><polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2"></polygon><line x1="12" y1="22" x2="12" y2="15.5"></line><polyline points="22 8.5 12 15.5 2 8.5"></polyline><polyline points="2 15.5 12 8.5 22 15.5"></polyline><line x1="12" y1="2" x2="12" y2="8.5"></line></svg>
                NicheCP
            </a>
            <div class="glass-nav-center">
                ${navLinksHTML}
            </div>
            <div class="glass-nav-right">
                ${rightSideHTML}
            </div>
        </div>
    `;
}
