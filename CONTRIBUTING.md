# Contributing

plugpin is a small Go module. Changes should keep the default scan offline,
read-only, and free of process execution.

## Build and test

Go 1.27.3 or newer (the version in `go.mod`):

```sh
go test ./...
go test -race ./...
go vet ./...
go tool staticcheck ./...
```

`golangci-lint` v2.15.0 matches CI:

```sh
golangci-lint run ./...
```

Optional version override when building a binary:

```sh
go build -trimpath -ldflags '-X github.com/alikesk222/plugpin/internal/cli.Version=0.1.0' -o plugpin ./cmd/plugpin
```

Tests use fixtures and temporary git repositories created with go-git. They
must not call the `git` binary, start an agent, or open the network. Do not
add exploit payloads or proof-of-concept checkouts.

## Fuzzing

Fuzz tests live next to the parsers they exercise:

- `internal/gitread/fuzz_test.go` (`FuzzParseHEAD`, `FuzzParsePackedRefs`)
- `internal/cfgparse/fuzz_test.go` (`FuzzParseHooksDocument`, `FuzzParseMCPDocument`, `FuzzParseCodexTOML`)

Seed corpora live beside those tests, under `testdata/fuzz/<FuzzName>/`, so
`go test` loads them. A corpus file looks like this:

```
go test fuzz v1
[]byte("ref: refs/heads/main\n")
```

```sh
go test -fuzz=FuzzParseHEAD -fuzztime=20s ./internal/gitread
```

## Layout

| Path | Role |
| --- | --- |
| `cmd/plugpin` | Binary entrypoint |
| `data/` | Embedded rule and floor catalogs |
| `internal/discover` | Per-agent and project readers |
| `internal/gitread` | `.git` file parser; go-git is secondary |
| `internal/checks` | Rule evaluation |
| `internal/report` | Text, JSON, and SARIF |
| `docs/` | Architecture, agents, rules, formats, threat model |

Paths marked `UNVERIFIED` in source comments are not confirmed against a live
install. Do not invent a path or a pin file to fill one of those gaps. If the
file is absent, skip it.

## Commit identity

New commits must use this author and committer, with no attribution trailer
whose line matches `*-by:` (including a Co-authored-by line):

```
Ali Kesik <alikesk222@users.noreply.github.com>
```

The parentless root commit predates this policy and is skipped by
`scripts/check-commit-identity.sh`. Do not rewrite it.

```sh
git config user.name "Ali Kesik"
git config user.email "alikesk222@users.noreply.github.com"
export GIT_AUTHOR_NAME="Ali Kesik"
export GIT_AUTHOR_EMAIL="alikesk222@users.noreply.github.com"
export GIT_COMMITTER_NAME="Ali Kesik"
export GIT_COMMITTER_EMAIL="alikesk222@users.noreply.github.com"
```

## Content check

`policy/content-token.sha256` stores only the SHA-256 of a banned substring.
The substring itself is not in the repository. Before pushing, install the
local hook (it writes under `.git/hooks` and is not committed):

```sh
PLUGPIN_CONTENT_TOKEN='…' sh scripts/install-local-hook.sh
```

The same variable drives `scripts/check-content-token.sh`. Do not commit the
token, the hook, or any file whose name or text contains the substring.

## Reports and baselines

Rule ids are `PP001` through `PP040` as listed in `data/rules.yaml`. A behavior
change needs a unit test and, when the text or SARIF shape changes, a golden
update:

```sh
UPDATE_GOLDEN=1 go test ./internal/scan -run TestGoldenSHABranch
```

Review the diff. Do not refresh goldens to hide a real change.

## Scope that will be declined

- Executing scanned code, or running `git` as a subprocess
- Network access on the default scan path
- Guessed marketplace or pin paths for fields marked unverified
- Turning a finding into an exploit or a remote scanner
- A package-manager tap or a signed release until the maintainer asks
