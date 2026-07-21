# Security Policy

pgscope is a database monitoring tool, so a vulnerability here has a different blast radius than most projects: it could mean unauthorized access to session data, query text, or the authentication mechanism protecting the dashboard. Please report responsibly rather than opening a public issue.

## Reporting a vulnerability

Use GitHub's private vulnerability reporting for this repository (Security tab → "Report a vulnerability") rather than filing a public issue or discussing it in a PR. If that's not available to you, open an issue titled only "Security issue, please contact me" with no details, and a maintainer will reach out for a private channel.

Please include:
- What you found and where (file, endpoint, or behavior)
- Steps to reproduce
- What you think the actual impact is (what could an attacker do with this)

## What's in scope

- Authentication and session handling (`internal/presentation/http/auth_*.go`)
- Rate limiting and IP banning (`internal/presentation/http/rate_limit_middleware.go`, `ban_middleware.go`)
- Anything that would let pgscope execute something beyond a read-only `SELECT` against a statistics view — see the root [README](README.md)'s "What pgscope is, and is not" section for why this boundary matters
- SQL injection in any collector under `internal/infrastructure/postgres`

## What's likely not a vulnerability

- The lack of a full user/role system — this is a deliberate, documented design choice for a single-operator tool (see [go/README.md](go/README.md)'s security model section), not an oversight
- Behavior when pgscope is run without a reverse proxy / without HTTPS — the docs are explicit that this is required for the `Secure` session cookie to work correctly, and that responsibility sits with the deployer, not the application

## Response time

This is a single-maintainer open-source project, not a company with an SLA. I'll do my best to acknowledge a report within a few days and follow up with a fix or a clear explanation of why something isn't a vulnerability.
