# Security exceptions

Sometimes a scanner flags something we decide to accept for a while.
Write that decision down here as a short dated note.

Do not hide findings forever inside CI config.

## Add an exception

1. Add a file named like `2026-08-02-trivy-CVE-2024-12345.md`
2. Fill in the top fields (example below)
3. Run `task security:exceptions`
4. Include the file in your PR

When `expires` passes, CI fails until you fix the issue or add a new note with a new date and reason.

## Example

```yaml
---
tool: trivy
finding: CVE-2024-12345
kind: cve
expires: 2026-11-01
owner: "@acctbl/maintainers"
reason: Fix not released yet. We block the risky path another way.
residual_risk: Only affects CI machines, not production.
---
```

| Field | Meaning |
| --- | --- |
| `tool` | Which scanner (`gitleaks`, `govulncheck`, `trivy`, or `trivy-iac`) |
| `finding` | What to ignore (CVE id, path, regex, and so on) |
| `kind` | How to read `finding` (`cve`, `path`, `regex`, `vuln_id`, or `misconfig_id`) |
| `expires` | Last day this is allowed (`YYYY-MM-DD`) |
| `owner` | Who owns removing it |
| `reason` | Why we allow it |
| `residual_risk` | What risk remains until then |
