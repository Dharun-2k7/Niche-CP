document.addEventListener('DOMContentLoaded', () => {
    // Auth Check Logic
    const urlParams = new URLSearchParams(window.location.search);
    const tokenFromUrl = urlParams.get('token');
    if (tokenFromUrl) {
        localStorage.setItem('jwt_token', tokenFromUrl);
        window.history.replaceState({}, document.title, window.location.pathname);
    }

    const token = localStorage.getItem('jwt_token');
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
            const input = customInput.value;

            if (!code.trim()) {
                showStatus('Please enter some code.', 'error');
                return;
            }

            runBtn.disabled = true;
            runBtn.textContent = 'Running...';
            terminalOutput.value = 'Executing...';

            try {
                const response = await fetch('http://localhost:8080/api/run', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ code, language, input })
                });

                const data = await response.json();

                if (response.ok) {
                    terminalOutput.value = data.output || 'No output';
                    if (data.stderr) {
                        terminalOutput.value += '\n\n[Errors]:\n' + data.stderr;
                    }
                } else {
                    terminalOutput.value = `Error: ${data.error}`;
                }
            } catch (err) {
                terminalOutput.value = 'Failed to connect to backend server.';
            } finally {
                runBtn.disabled = false;
                runBtn.textContent = 'Run Code';
            }
        });

        submitBtn.addEventListener('click', async () => {
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
                // This URL points to our Go Backend
                const response = await fetch('http://localhost:8080/api/submit', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${token}`
                    },
                    body: JSON.stringify({
                        problem_id: 1, // Hardcoded for MVP (will be dynamic soon)
                        code: code,
                        language: language
                    })
                });

                const data = await response.json();

                if (response.ok) {
                    showStatus(`Submission Queued! ID: ${data.submission_id}. Waiting for execution...`, 'pending');
                    
                    // Poll for status
                    const pollInterval = setInterval(async () => {
                        try {
                            const statusRes = await fetch(`http://localhost:8080/api/submissions/${data.submission_id}`);
                            if (statusRes.ok) {
                                const statusData = await statusRes.json();
                                if (statusData.status !== 'PENDING') {
                                    clearInterval(pollInterval);
                                    let statusType = 'error';
                                    if (statusData.status === 'ACCEPTED') {
                                        statusType = 'success';
                                    } else if (statusData.status === 'TIME_LIMIT_EXCEEDED' || statusData.status === 'MEMORY_LIMIT_EXCEEDED') {
                                        statusType = 'warning';
                                    }
                                    showStatus(`Submission ${statusData.status}! (Time: ${statusData.execution_time_ms || 0}ms)`, statusType);
                                    submitBtn.disabled = false;
                                    submitBtn.textContent = 'Submit Code';
                                }
                            }
                        } catch (e) {
                            clearInterval(pollInterval);
                            showStatus('Error checking status.', 'error');
                            submitBtn.disabled = false;
                            submitBtn.textContent = 'Submit Code';
                        }
                    }, 1000);
                    return; // Don't re-enable button yet
                } else {
                    showStatus(`Error: ${data.error}`, 'error');
                }
            } catch (err) {
                showStatus('Failed to connect to backend server. Is it running on :8080?', 'error');
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
