<!--
The title, one short line, becomes the commit message (squash and merge);
the details go here. Target the dev branch.
-->

## What and why



## Checks

- [ ] The pull request targets `dev`.
- [ ] `go test -race ./...` and `golangci-lint run` pass.
- [ ] The change has tests (testify/require, `t.Parallel()`).
- [ ] A change of behaviour is described in `README.md` and `README.fr.md`.
- [ ] A line under "Unreleased" in `CHANGELOG.md`.
