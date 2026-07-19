# Changelog

All notable changes to the NicheCP project will be documented in this file.

## [Unreleased]

### Added
- **AGENTS.md**: Persistent AI instruction manual outlining repository structures, conventions, and context rules.
- **.ai/**: Directory containing `context.md`, `tasks.md`, and `session.md` to ensure continuous AI synchronization.
- **docs/**: Comprehensive documentation directory containing architecture guidelines, setup instructions, decision logs, and this changelog.

### Changed
- Refactored `RequireAdmin` middleware to natively query PostgreSQL for granular database roles (`admin`, `superadmin`), moving away from purely hardcoded superadmin strings.
- Updated `index.html` Hero Section for improved vertical spacing, refined typography, and standardized padding on Bento grid components.
- Modified global `fetch` calls across all dynamic pages (`arena.html`, `contests.html`, `profile.html`, `admin.html`) to enforce strict `try/catch` wrapping and null-safe array checks (`Array.isArray`).
- Updated data tables in admin and profile pages to utilize `overflow-x: auto;` for improved mobile responsiveness.
- Redesigned profile and registration roll-number extraction logic. Roll numbers are no longer input manually but strictly derived from verified `.amrita.edu` OTP responses.

### Fixed
- Fixed Google OAuth URL redirection which was previously failing on non-standard development ports. Redirection now utilizes dynamic frontend URL extraction via the `oauth_referer` cookie.
- Fixed 404 dead links on the global navigation by creating placeholder HTML files (`practice.html`, `learn.html`, `rankings.html`, `blogs.html`) using the existing `coming-soon.html` template.
- Eliminated JavaScript crashing errors inside GSAP animation logic caused by unexpected `null` payloads from API responses.
- Fixed Solved Count duplication bug by using `COUNT(DISTINCT problem_id)` in profile metrics queries.

### Security
- Implemented a secure two-step OTP validation flow for email updates (validates current email first, then new email).
- Enforced a 24-hour cooldown period on password resets following a successful email address change to prevent account hijacking.
- Restricted the frontend Admin Panel navigation logic exclusively to the verified superadmin email.
