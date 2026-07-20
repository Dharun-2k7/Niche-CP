# 04 - Frontend Architecture

## Purpose
The NicheCP frontend is designed to be lightweight, incredibly fast, and visually premium. Instead of relying on a bulky Single Page Application (SPA) framework like React or Next.js, the decision was made to build a "God-Mode Aesthetic" UI using Vanilla HTML5, CSS3 (Glassmorphism), and modular ES6 JavaScript.

## Frontend Structure

The structure is contained within the `frontend/` directory.

### `css/`
- **`style.css`**: The monolithic stylesheet containing all CSS variables (Design Tokens), Dark Theme / Light Theme variations, Grid/Flexbox layouts, and Glassmorphism effects (backdrop-filters). It utilizes a customized color palette specifically requested for a premium feel (e.g., `#ff4081` accents).

### `js/`
- **`app.js`**: The core state management and API communication hub. Handles JWT storage (`localStorage`), global fetch wrappers with error handling, and DOM initialization.
- **`animations.js`**: A dedicated file leveraging **GSAP (GreenSock Animation Platform)** to handle complex micro-interactions, page transitions, hero section reveals, and hover states, giving the platform its "wow" factor without CSS bloat.
- **`components.js`**: Contains reusable DOM manipulation functions (e.g., rendering standard navigation bars, footer injections, or alert toasts).

### Pages
- **`index.html`**: The landing page and hero section.
- **`login.html` / `register.html`**: The authentication portals.
- **`dashboard.html`**: User profile, solved problems overview, and rating graph (powered by Chart.js).
- **`arena.html`**: The primary problem-solving interface. Features a split-pane layout with the problem description on the left and the **Monaco Editor** (VS Code's core) on the right.
- **`contests.html`**: Lists upcoming, ongoing, and past contests.
- **`admin.html`**: A protected view accessible only by users with the `admin` or `superadmin` role, providing wizards for creating contests and appending testcases.

## API Communication & State Management
- **Stateless HTTP:** The frontend relies on native `fetch()`. A global interceptor in `app.js` automatically attaches the `Authorization: Bearer <token>` header to all outgoing requests.
- **Error Handling:** If an API responds with `401 Unauthorized` (e.g., expired JWT), the frontend automatically purges `localStorage` and redirects to `login.html`.
- **Status Polling:** When a user submits code in `arena.html`, the frontend receives a `submission_id`. It then enters a `setInterval` loop polling `/api/submission/{id}` every 1000ms until the status transitions from `PENDING` to a final verdict (`ACCEPTED`, `WRONG_ANSWER`, etc.), at which point it clears the interval and updates the UI via GSAP animations.

## Design Decisions
- **Why Vanilla JS?** To minimize the Time-to-Interactive (TTI) and avoid bundle compilation steps (Webpack/Vite) during rapid development. The application is small enough that DOM manipulation remains manageable.
- **Why Monaco Editor?** It provides best-in-class syntax highlighting, auto-completion, and minimap features identical to VS Code, making the coding experience feel highly professional compared to simple textareas or older editors like CodeMirror.
