# Security policy

## Supported versions

v0.1.0 is the version string in source. No release tag has been published.
Until a tag exists, the supported build is the `main` branch of
`github.com/alikesk222/plugpin`.

## Reporting a vulnerability

Report vulnerabilities in plugpin through
[GitHub Security Advisories](https://github.com/alikesk222/plugpin/security/advisories/new).
Please include the version string (`plugpin version`), the command you ran,
and enough detail to reproduce the bug on a fixture. Do not include secrets
from a real home directory.

This channel is for defects in plugpin: unsafe file handling, unexpected
process execution, secret leakage into reports, or a check that contradicts
the documented rules. Findings about third-party agents belong with those
vendors. Do not attach exploit code.

## What this project will not accept

Patches that add network access to the default scan, that execute scanned
hooks or plugins, or that turn a check into an exploit, will be rejected.
See [RESPONSIBLE_USE.md](RESPONSIBLE_USE.md).

## Supply chain

The default scan is offline. Dependencies are the Go module graph in
`go.mod`. CI runs `go vet`, `staticcheck`, `golangci-lint`, `go test -race`,
`govulncheck`, and CodeQL. An unreachable vulnerability in a required module
does not fail the build when `govulncheck` reports it as unreachable.

Release signing and a package-manager tap are not part of v0.1. See the
roadmap in [README.md](README.md).
