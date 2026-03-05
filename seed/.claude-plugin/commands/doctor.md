---
name: {{.Name}}-doctor
description: Check {{.Name}} plugin health — CLI, auth, API connectivity, project context.
invocable: true
---

# /{{.Name}}-doctor

Run the {{.Name}} CLI health check and report results.

```bash
{{.Name}} doctor --json
```

Interpret the output:
- **pass**: Working correctly
- **warn**: Non-critical issue (e.g., shell completion not installed)
- **skip**: Check not run (e.g., unauthenticated or not applicable)
- **fail**: Broken — needs attention

For any failures, follow the `hint` field in the check output. Common fixes:
- Authentication failed → `{{.Name}} auth login`
- API unreachable → check network / VPN
- Plugin not installed → `{{.Name}} setup claude`

Report results concisely: list failures and warnings with their hints. If everything passes, say so.
