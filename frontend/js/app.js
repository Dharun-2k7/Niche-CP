document.addEventListener('DOMContentLoaded', () => {
    const submitBtn = document.getElementById('submitBtn');
    const codeEditor = document.getElementById('codeEditor');
    const languageSelect = document.getElementById('language');
    const statusMessage = document.getElementById('statusMessage');

    submitBtn.addEventListener('click', async () => {
        const code = codeEditor.value;
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
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    user_id: 1, // Hardcoded for MVP
                    problem_id: 1, // Hardcoded for MVP
                    code: code,
                    language: language
                })
            });

            const data = await response.json();

            if (response.ok) {
                showStatus(`Submission Queued! ID: ${data.submission_id}. Waiting for execution...`, 'success');
                // In Phase 2, we will add polling here to check if the background worker finished running it
            } else {
                showStatus(`Error: ${data.error}`, 'error');
            }
        } catch (err) {
            showStatus('Failed to connect to backend server. Is it running on :8080?', 'error');
        } finally {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Submit Code';
        }
    });

    function showStatus(message, type) {
        statusMessage.textContent = message;
        statusMessage.className = `status ${type}`;
    }
});
