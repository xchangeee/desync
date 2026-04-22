# Claude Instructions

## Code Generation

## Build

This project uses a Makefile. Key targets:

- `make build` — Build all packages
- `make fmt` — Format code
- `make vet` — Run static analysis
- `make test` — Run all tests
- `make coverage` — Run all tests with coverage
- `make clean` — Remove build artifacts

To see source files from a dependency, or to answer questions about a dependency, run `go mod download -json MODULE` and use the returned `Dir` path to read the files.

Use `go doc foo.Bar` or `go doc -all foo` to read documentation for packages, types, functions, etc.

Use `go run .` or `go run ./cmd/foo` instead of `go build` to run programs, to avoid leaving behind build artifacts.

After you are done, verify changes with `make check coverage`.

## Code

Document not only what a piece of logic does, but also what its responsibilities are in the bigger picture and how its supposed to interact with other components.
