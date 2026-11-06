# Contributing

Thank you for your interest! You may write in English or in French.

## Report a problem, suggest an idea

- With a GitHub account: [open an issue](https://github.com/FrederickFrance/igiveyouarendezvous/issues/new/choose),
  with the problem file or the link of the online demo that shows the
  problem. Issues are public: put no real names in them (`Candidate 1`,
  initials…).
- Without one: write to me on
  [LinkedIn](https://www.linkedin.com/in/frederic-sauvage-rueangpanichpiban),
  with the file attached.
- A security problem: see the [security policy](SECURITY.md), not a public
  issue.

Do not hesitate: every idea is welcome, even a small one, even unfinished,
and telling where you got stuck is as useful as a bug report. For a large
change, open an issue first, so that we agree on it before you spend time
on the code.

## Change the code

You need Go, at the version of `go.mod`, and
[golangci-lint](https://golangci-lint.run/) 2.15 or later. Before a pull
request:

```sh
go test -race ./...
golangci-lint run
```

The CI runs them too, with `osv-scanner`, and the tests compiled to
WebAssembly for the online demo (`.github/workflows/ci.yml`).

The conventions of the project:

- Code, comments and messages in English, with the standard Go naming.
- Tests with [testify/require](https://pkg.go.dev/github.com/stretchr/testify/require),
  and `t.Parallel()` in every test.
- A new key of the problem file goes into the JSON schema
  (`schema/problem.schema.json`) too: the tests check that the schema and
  the code know the same keys. Every file of `example/` is run by the tests.
- A change of behaviour is described in `README.md` and `README.fr.md` (the
  other READMEs are short and do not need it), and gets a line under
  "Unreleased" in [CHANGELOG.md](CHANGELOG.md).

## Pull requests

- Start from the `dev` branch, and target `dev`: `main` only receives
  `dev`, at each release.
- One subject per pull request. Its title, one short line, becomes the
  commit message (squash and merge); the details go in its description.

By contributing, you agree that your contribution is released under the
licences of the project, [0BSD](LICENSE) or [WTFPL](WTFPL.txt), at the
choice of whoever uses it.