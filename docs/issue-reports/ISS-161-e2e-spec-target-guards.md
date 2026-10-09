# ISS-161: four e2e specs read E2E_API_URL without the target guard

- Severity: low (defence in depth). Layer: e2e. Module: tests.
- Symptom: account-recovery, downloads-bearer, exam-lifecycle and loyalty-narrative specs read `E2E_API_URL` directly; global-setup/seed already guard live runs, but a spec should not be point-able at the frozen bilimbaga-test host by itself.
- Fix: each spec now builds its API base through `requireTarget('E2E_API_URL', ..., process.env)` from `scripts/lib/target-guard` (same pattern as seed.ts and certificates.spec.ts).
- Mutation check (statically, no stack): for each of the four specs `E2E_API_URL=http://localhost:8080 playwright test --list` lists the tests; `E2E_API_URL=https://bilimbaga-test.ai-dala.com` is refused with "protected production-class host". Caveat: the refusal can come from global-setup/seed (which import the same guard) before the spec module is loaded, so this proves the combined path, not each spec in isolation.
- Not run: no e2e spec was executed. No separate Code Reviewer subagent (mechanical 1-line-per-file test change under the low-memory one-agent rule).
