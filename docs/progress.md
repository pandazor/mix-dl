# dl-manager execution ledger

Plan: docs/superpowers/plans/2026-11-08-dl-manager.md

- 2026-11-08: User approved starting; browser actions explicitly manual; resettable quota confirmed.
- Toolchain: portable Go installed in session work directory; Python HTTPS works while Windows curl/PowerShell TLS do not in this sandbox.
- Ruling: Build in writable workspace then deliver an archive for G:; this session does not have write permission at the requested G: path.
- Pre-flight: storage Job/Settings/Profile/Event definitions shared by queue, engine and UI; one model package is the source of truth.

## 2026-11-09 completion
- Ruling: no Git repository exists; preserve the existing isolated workspace and deliver portable + source ZIPs instead of creating unrelated Git history. Cost: no commit-based history, source snapshot included.
- Task 1 complete: model/storage package tests passed; added transactional clearing of deleted default profiles.
- Task 2 complete: concurrency/quota/restart tests passed; pending cancellation now blocks Start, and credential mutations serialize against dispatch.
- Task 3 complete: real bundled gallery-dl local HTTP interrupted/resumed/archive replay verified; two same-basename sources retained; windows path-strip corrected. DPAPI test remains explicitly skipped in restricted terminal after .NET independently reproduces unavailable profile.
- Task 4: native window launched from clean data and observed via accessibility; browser-preview settings/add/start/history/accounts tested, public python.org image downloaded successfully. Native capture/coordinate input unavailable. Portable/source ZIP integrity and engine SHA-256 verified.
- Final independent review: four Important findings, none Critical. All four reproduced RED then fixed GREEN; storage default-profile removal also fixed. Full suite with DLM_RESTRICTED_TESTS=1 passes; go vet ./... passes; TypeScript compilation passes.
- Ruling: preserve directlink domain/path naming under custom filename mode to prevent collisions; user-selected custom title format is not applied to direct links. Cost: naming exception documented.
- Ruling: WebView2 Runtime remains a clearly documented prerequisite instead of adding a large fixed runtime. It is installed on the test PC. Cost: another PC without WebView2 needs the official runtime.
- Final minor deferred: both YouTube naming modes still use id + title. Documented in README/VALIDATION.
- Declined review scopes resolved: native DPAPI/private accounts and native picker/tray cycle not claimed validated; alias/junction destination overlap and hostile metadata guarantees are outside current coverage and documented. Engine filenames remain sanitized by the official engines.
- Verification: 26 test functions including one subprocess helper; DPAPI roundtrip is the only skipped check under restricted mode. Public test writes kept outside deliverables; no private credentials read or imported.

- Task 4 complete: final executable launched natively with its portable WebView profile; preview restart retained the completed public job and volume but left the queue paused. Package byte checks and CRC checks passed. Deliverables exclude all test data.
