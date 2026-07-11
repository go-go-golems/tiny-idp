Staticcheck is a state of the art linter for the [Go programming language](https://go.dev/). Using static analysis, it finds bugs and performance issues, offers simplifications, and enforces style rules.

Each of the [150+](https://staticcheck.dev/docs/checks/) checks has been designed to be fast, precise and useful. When Staticcheck flags code, you can be sure that it isn’t wasting your time with unactionable warnings. Unlike many other linters, Staticcheck focuses on checks that produce few to no false positives. It’s the ideal candidate for running in CI without risking spurious failures.

Staticcheck aims to be trivial to adopt. It behaves just like the official `go` tool and requires no learning to get started with. Just run `staticcheck ./...` on your code in addition to `go vet ./...`.

While checks have been designed to be useful out of the box, they still provide [configuration](https://staticcheck.dev/docs/configuration/) where necessary, to fine-tune to your needs, without overwhelming you with hundreds of options.

Staticcheck can be used from the command line, in CI, and even [directly from your editor](https://github.com/golang/tools/blob/master/gopls/doc/settings.md#staticcheck-bool).

Staticcheck is open source and offered completely free of charge. [Sponsors](https://staticcheck.dev/sponsors/) guarantee its continued development.

---

##### Getting started

Quickly get started using Staticcheck

##### Configuration

Tweak Staticcheck to your requirements

##### Checks

Explanations for all checks in Staticcheck