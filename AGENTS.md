# AGENTS.md

## Code style
- Prefer guard clauses and avoid nesting.
- Don't include comments unless they add meaningful context.
- Use the `errors` package for sentinel errors.

## Testing instructions
- Fix any test or type errors until everything succeeds.
- Add or update tests for the code you change, even if nobody asked.
- Update Go tests and scripts/e2e-tests.sh as needed.
- Run deploy-cluster.sh before e2e-tests.sh, and then delete-cluster.sh after.
