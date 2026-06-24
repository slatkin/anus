# Task completion checklist

After any coding change, run the appropriate subset:

## Go changes
```bash
make vet          # go vet — catch common mistakes
make test         # go test ./... — full test suite
```

## Frontend changes
```bash
cd frontend && npm test    # Vitest unit tests
```
No TypeScript compiler (plain JS), no ESLint configured — vet + tests are the gate.

## Before committing
- Ensure no `Co-Authored-By` in commit message (project rule).
- `git pull --rebase` before pushing (user workflow rule).
- Fix any `vite-plugin-svelte` a11y warnings before committing.
