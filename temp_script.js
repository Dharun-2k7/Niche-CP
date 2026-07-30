    <script>
        document.addEventListener('DOMContentLoaded', async () => {
            const container = document.getElementById('contestsContainer');
            const token = localStorage.getItem('token');
            
            try {
                // Fetch contests
                const response = await fetch('/api/contests');
                if (!response.ok) throw new Error('Failed to fetch contests');
                const data = await response.json();
                const contestList = data.contests || (Array.isArray(data) ? data : []);
                const serverTime = data.server_time ? new Date(data.server_time).getTime() : new Date().getTime();
                const timeOffset = serverTime - new Date().getTime();
                
                if (!Array.isArray(contestList)) {
                    container.innerHTML = '<div style="text-align:center; color: var(--text-muted); margin-top: 40px;">No contests available.</div>';
                    return;
                }

                // Fetch registrations if logged in
                let myRegistrations = [];
                if (token) {
                    const regRes = await fetch('/api/contests/my-registrations', {
                        headers: { 'Authorization': `Bearer ${token}` }
                    });
                    if (regRes.ok) {
                        const regData = await regRes.json();
                        if (regData.registered_contests) {
                            myRegistrations = regData.registered_contests;
                        }
                    }
                }

                const upcoming = [];
                const running = [];
                const past = [];
                
                contestList.forEach(c => {
                    if (c.status === 'CREATED' || c.status === 'UPCOMING') upcoming.push(c);
                    else if (c.status === 'RUNNING') running.push(c);
                    else past.push(c);
                });

                let html = '';

                // HELPER: Format date
                const formatDate = (ds) => new Date(ds).toLocaleString('en-US', { weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit', timeZoneName: 'short' });
                const formatLen = (m) => `${Math.floor(m / 60)}:${(m % 60).toString().padStart(2, '0')}`;

                // ACTIVE / RUNNING SECTION
                if (running.length > 0) {
                    html += `
                        <div class="contest-list-header">
                            <div>LIVE NOW</div>
                            <div>LENGTH</div>
                            <div></div>
                        </div>
                    `;
                    running.forEach(c => {
                        const isReg = myRegistrations.includes(c.id);
                        let actionBtn = '';
                        if (isReg) {
                            actionBtn = `<a href="contest_arena.html?id=${c.id}" class="btn-enter" style="background:var(--success-color); color:#000;">Enter Contest</a>`;
                        } else {
                            actionBtn = `<button class="btn-enter" disabled style="background:#333; cursor:not-allowed; border:none; color:#888;">Registration Closed</button>`;
                        }
                        
                        html += `
                            <div class="contest-row" style="border-left: 3px solid var(--success-color);">
                                <div>
                                    <div class="contest-title" style="color:var(--success-color);">${c.title} <span style="font-size:12px; margin-left:8px; border:1px solid var(--success-color); padding:2px 6px; border-radius:12px;">LIVE</span></div>
                                    <div class="contest-meta">Started: ${formatDate(c.start_time)}</div>
                                </div>
                                <div class="contest-meta">${formatLen(c.duration_minutes)}</div>
                                <div class="contest-actions" style="display:flex; gap:12px; align-items:center;">
                                    <div style="display:flex; flex-direction:column; align-items:flex-end;">
                                        <div style="font-size:10px; color:var(--text-muted); text-transform:uppercase;">Remaining Time</div>
                                        <div class="contest-countdown" id="countdown-${c.id}" style="color:var(--success-color);">00:00:00:00</div>
                                    </div>
                                    <a href="rankings.html?contest_id=${c.id}" class="btn-ghost" style="padding: 10px 16px;">Leaderboard</a>
                                    ${actionBtn}
                                </div>
                            </div>
                        `;
                        
                        // Running Countdown Loop
                        setInterval(() => {
                            const now = new Date().getTime() + timeOffset;
                            const distance = new Date(c.end_time).getTime() - now;
                            const el = document.getElementById(`countdown-${c.id}`);
                            if(el && distance > 0) {
                                const d = Math.floor(distance / (1000 * 60 * 60 * 24));
                                const h = Math.floor((distance % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
                                const m = Math.floor((distance % (1000 * 60 * 60)) / (1000 * 60));
                                const s = Math.floor((distance % (1000 * 60)) / 1000);
                                el.innerText = `${d.toString().padStart(2,'0')}:${h.toString().padStart(2,'0')}:${m.toString().padStart(2,'0')}:${s.toString().padStart(2,'0')}`;
                            } else if (el) {
                                window.location.reload(); // Force reload to move to Previous
                            }
                        }, 1000);
                    });
                }

                // UPCOMING SECTION
                if (upcoming.length > 0) {
                    html += `
                        <div class="contest-list-header" style="margin-top: ${running.length > 0 ? '60px' : '0'};">
                            <div>UPCOMING</div>
                            <div>LENGTH</div>
                            <div></div>
                        </div>
                    `;
                    upcoming.forEach(c => {
                        const isReg = myRegistrations.includes(c.id);
                        let actionBtn = '';
                        if (isReg) {
                            actionBtn = `<button class="btn-enter" disabled style="background:#2a2a35; border:none; color:var(--text-muted); cursor:not-allowed;">Registered</button>`;
                        } else {
                            if (c.registration_open) {
                                if (token) {
                                    actionBtn = `<button class="btn-enter" onclick="registerForContest(${c.id}, this)">Register</button>`;
                                } else {
                                    actionBtn = `<a href="login.html" class="btn-enter">Log in to Register</a>`;
                                }
                            } else {
                                actionBtn = `<button class="btn-enter" disabled style="background:#2a2a35; border:none; color:var(--text-muted); cursor:not-allowed;">Registration Closed</button>`;
                            }
                        }
                        
                        html += `
                            <div class="contest-row">
                                <div>
                                    <div class="contest-title">${c.title}</div>
                                    <div class="contest-meta">${formatDate(c.start_time)}</div>
                                </div>
                                <div class="contest-meta">${formatLen(c.duration_minutes)}</div>
                                <div class="contest-actions" style="display:flex; gap:12px; align-items:center;">
                                    <div style="display:flex; flex-direction:column; align-items:flex-end;">
                                        <div style="font-size:10px; color:var(--text-muted); text-transform:uppercase;">Starts In</div>
                                        <div class="contest-countdown" id="countdown-${c.id}">00:00:00:00</div>
                                    </div>
                                    ${actionBtn}
                                </div>
                            </div>
                        `;
                        
                        // Start countdown loop
                        setInterval(() => {
                            const now = new Date().getTime() + timeOffset;
                            const distance = new Date(c.start_time).getTime() - now;
                            const el = document.getElementById(`countdown-${c.id}`);
                            if(el && distance > 0) {
                                const d = Math.floor(distance / (1000 * 60 * 60 * 24));
                                const h = Math.floor((distance % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
                                const m = Math.floor((distance % (1000 * 60 * 60)) / (1000 * 60));
                                const s = Math.floor((distance % (1000 * 60)) / 1000);
                                el.innerText = `${d.toString().padStart(2,'0')}:${h.toString().padStart(2,'0')}:${m.toString().padStart(2,'0')}:${s.toString().padStart(2,'0')}`;
                            } else if (el) {
                                window.location.reload(); // Force reload to move to Running
                            }
                        }, 1000);
                    });
                }

                // PREVIOUS SECTION
                if (past.length > 0) {
                    html += `
                        <div class="contest-list-header" style="margin-top: ${(running.length > 0 || upcoming.length > 0) ? '60px' : '0'};">
                            <div>PREVIOUS</div>
                            <div>LENGTH</div>
                            <div></div>
                        </div>
                    `;
                    past.forEach(c => {
                        html += `
                            <div class="contest-row" style="opacity: 0.7;">
                                <div>
                                    <div class="contest-title">${c.title}</div>
                                    <div class="contest-meta">${formatDate(c.start_time)}</div>
                                </div>
                                <div class="contest-meta">${formatLen(c.duration_minutes)}</div>
                                <div class="contest-actions">
                                    <a href="rankings.html?contest_id=${c.id}" class="btn-ghost" style="padding: 10px 16px;">View Results</a>
                                    <a href="contest_arena.html?id=${c.id}" class="btn-enter" style="background:transparent; border:1px solid #444; color:#fff;">Practice</a>
                                </div>
                            </div>
                        `;
                    });
                }

                if (html === '') {
                    html = `<div style="text-align:center; padding: 40px; color: var(--text-muted);">No contests found.</div>`;
                }

                container.innerHTML = html;

            } catch (err) {
                container.innerHTML = `<div style="color: #ef4444; padding: 20px; text-align:center;">Error loading contests. ${err.message}</div>`;
            }
        });

        // Global Registration Function
        async function registerForContest(contestId, btn) {
            const token = localStorage.getItem('token');
            if(!token) return window.location.href = 'login.html';
            
            btn.innerHTML = 'Registering...';
            btn.disabled = true;
            
            try {
                const res = await fetch(`/api/contests/${contestId}/register`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
                    body: JSON.stringify({})
                });
                const data = await res.json();
                
                if (res.ok) {
                    btn.innerHTML = 'Registered';
                    btn.style.background = '#2a2a35';
                    btn.style.color = 'var(--text-muted)';
                    btn.style.border = 'none';
                    // Don't re-enable button
                } else {
                    alert('Registration failed: ' + (data.error || 'Unknown error'));
                    btn.innerHTML = 'Register';
                    btn.disabled = false;
                }
            } catch (err) {
                alert('Network error');
                btn.innerHTML = 'Register';
                btn.disabled = false;
            }
        }
    </script>
