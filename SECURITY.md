> **Disclaimer:** `semctl` is an independent, open-source command line interface for Semaphore UI. It is **not affiliated with, endorsed by, sponsored by, or officially connected to the Semaphore UI project or its creators**. This tool is intended for personal use, educational purposes, and operational convenience at your own risk. All product names, logos, and brands are property of their respective owners.

# Security Policy

## Supported Versions

Only the latest release on the `main` branch is actively supported with security updates. Please keep your installation up to date.

## Reporting a Vulnerability

If you discover a security vulnerability in `semctl`, please report it **privately** so we can fix it before details are disclosed publicly.

### How to report

1. **GitHub Security Advisories** (preferred):
   - Go to [Security → Advisories → New draft advisory](https://github.com/moep90/semaphore-cli/security/advisories/new).
   - Provide a clear description, steps to reproduce, and impact assessment.
   - We will triage within 7 days and coordinate a fix and disclosure timeline.

2. **Email** (alternative):
   - If you cannot use GitHub advisories, email the maintainers directly. Contact information can be found in the repository maintainer list or `CODEOWNERS`.

### What to include

- Affected version(s)
- Steps to reproduce
- Expected vs. actual behavior
- Impact assessment (e.g., token exposure, privilege escalation)
- Suggested fix (if any)

### Our commitment

- We will acknowledge receipt within 7 days.
- We will work on a fix and release on a timeline proportional to severity.
- We will credit you in the advisory unless you prefer to remain anonymous.
- We will request a CVE if warranted.

## Dependency CVEs

If a vulnerability is reported in a Go module dependency:

1. We use `govulncheck` to determine whether vulnerable symbols are actually reachable from `semctl`'s code.
2. If the vulnerability is reachable, we will upgrade the dependency and cut a patch release.
3. If it is not reachable, we will document the assessment and consider upgrading opportunistically.

## Security Best Practices for Users

- Prefer the OS keyring for token storage (`semctl auth login`).
- Avoid passing tokens as shell arguments or persisting them in shell history.
- Use `SEMAPHORE_TOKEN` only in ephemeral CI environments.
- Review `--debug` output carefully; secrets are redacted, but redaction is not a guarantee.
