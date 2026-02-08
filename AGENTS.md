# AGENTS.md

## Code style
- Prefer guard clauses and avoid nesting.
- Don't include comments unless they add meaningful context.
- Use the `errors` package for sentinel errors.
- Don't ignore errors by assigning to `_`.
- Ensure any defaults are the same in deploy/ and charts/

## Testing instructions
- Tests are defined in:
  - `charts/binarylane-cloud-controller-manager/tests`
  - `scripts/e2e-tests.sh`
  - `internal/**/*_test.go`
- Add or update tests for the code you change, even if nobody asked.
- Run deploy-cluster.sh before e2e-tests.sh, and then delete-cluster.sh after.
- Run golangci-lint locally to ensure no linting errors.
