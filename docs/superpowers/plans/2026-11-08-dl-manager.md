# dl-manager Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a tested portable Windows download manager with a persistent queue, protected credentials and a resettable volume budget.

**Architecture:** A Go core exposes a same-origin JSON API to an embedded Wails interface. SQLite owns state; process adapters isolate gallery-dl and yt-dlp. Windows Job Objects and DPAPI own process lifetime and credential protection.

**Tech Stack:** Go 1.26+, Wails 3, TypeScript, modernc SQLite, Windows APIs.

**Spec:** ../../design.md

## Global Constraints
- Explicit resume after application restart. Resetting the volume counter never resumes downloads.
- No shell interpolation, credentials in logs, automatic browser control or automatic transcoding.
- Portable package; 1–3 workers; independent gallery-dl/yt-dlp archives; images/videos/extensions.
- User authorized execution in the conversation; execute inline without another approval round.

## Review Focus
- Concurrent jobs targeting the same archive must not duplicate files (task 2).
- Reset while work is active must establish a safe accounting boundary without resuming paused work (task 2).
- Invalid/missing credentials must not corrupt saved profiles or leak in stdout (task 3).
- Malicious URL/template and reserved Windows names must not escape destinations (task 1/3).
- Crash/restart and child processes must preserve partials and never autostart (task 2/4).

### Task 1: Persistent model and validation
**Files:** internal/model/model.go, internal/storage/storage.go and tests.
**Interfaces:** Open(path) -> *Store; Snapshot; SaveSettings; AddJobs; UpdateJob; SaveProfile; ResetBudget.
- [ ] Write/run failing tests for URL rejection, filter validation, Windows paths, SQLite persistence and restart-to-paused.
- [ ] Implement validated models and atomic SQLite transactions, with errors surfaced.
- [ ] Run the package tests; document results in docs/progress.md.

### Task 2: Queue and quota
**Files:** internal/queue/queue.go and tests.
**Interfaces:** Runner.Run(context.Context, Job, func(Event)) error; Manager.Start/Pause/Action/ResetBudget/Close.
- [ ] Write/run failing tests for worker cap, no autostart, per-resource exclusion, budget pause/reset and job terminal states.
- [ ] Implement dispatcher with single state owner, cancellation and persisted budget.
- [ ] Run queue + storage suite.

### Task 3: Engines, accounts and secrets
**Files:** internal/downloader, internal/auth, internal/platform, internal/logging and tests.
**Interfaces:** ResolveEngine, BuildInvocation, Vault.Save/Load/Delete, Runner.Run.
- [ ] Write/run tests for safe arguments, secret redaction, DPAPI, classified errors and configured engine selection.
- [ ] Implement adapters, profile guides, private temporary configuration, progress and archive handling.
- [ ] Exercise a real local HTTP download including interruption/resume and no duplicates.

### Task 4: Desktop interface and package
**Files:** main.go, internal/web, frontend, scripts/package.py, README.md.
- [ ] Exercise API requests for queue/settings/profiles in HTTP tests before implementing handlers.
- [ ] Build four views, native file pickers, tray, status/errors, quota reset and engine diagnostics.
- [ ] Compile TypeScript + Go Windows executable, validate actual window, package official dependencies.
- [ ] Run full suite and independent review, fix material findings, save source/package deliverables.
