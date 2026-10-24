# Responsible use

plugpin is a defensive, read-only auditor. It inspects configuration and git
metadata that is already on a machine. It does not exploit missing client
fixes, and it does not change the system it scans.

Use it only on machines and repositories you own, or that you are explicitly
authorized to inspect. Scanning a machine, home directory, or project you are
not allowed to access is outside the purpose of this tool.

plugpin does not:

- run `git`, the coding agents, or any scanned hook, plugin, or MCP command
  (the only optional process is `--exec-version`, which runs a version flag
  and is off by default)
- open network connections during a scan
- write plugin, hook, or MCP configuration
- ship exploit code, proof-of-concept payloads, or weaponized checks

A finding means the on-disk state matches a rule in `data/rules.yaml`. It is
not, by itself, proof that code ran or that data left the machine. Update the
agent, turn off untrusted auto-update, and reinstall plugins from commits you
have reviewed. plugpin does not replace that work.

Report vulnerabilities in plugpin itself through the process in
[SECURITY.md](SECURITY.md). Do not file exploit details for the agents in a
public issue.
