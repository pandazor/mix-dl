# Languages Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans.

**Goal:** Five-language UI and user guides, donation setup ready.
**Architecture:** Shared local JSON catalog, explicit UI translation calls, persisted language endpoint, native labels.
**Tech Stack:** Go, TypeScript, Playwright.
**Spec:** docs/superpowers/specs/2026-11-09-languages-design.md

## Global Constraints
fr/en/es/zh-CN/ja; French legacy default; no user data or raw engine log translation; preserve data/downloads; no live donation recipient without user link.

## Review Focus
- Switching language must preserve unsaved input, quota, paths and account associations.
- HTML-like names, paths and tokens remain literal and escaped.
- Long Spanish labels and CJK text fit the minimum window.
- Unsupported language and old Settings are handled without corruption.
- Catalog placeholders and documentation links remain complete in archives.

### Task 1: language persistence and UI
Files: internal/model, internal/queue, internal/web, internal/i18n; frontend/i18n.ts, app.ts; main.go.
Interfaces: Settings.Language; Manager.SaveLanguage(string); POST settings/language; t(source,...params), setLanguage, locale.
- [x] Write and run failing API test for five values, rejection and preservation.
- [x] Implement persistence and catalog, translate static UI and known messages, native labels.
- [x] Browser test switch/reload/all pages/unsaved values/escaped user text; validate catalog coverage.

### Task 2: guides and delivery
Files: README.md; docs/README.{en,es,zh-CN,ja}.md; docs/images; .github/FUNDING.yml; scripts/package.py; version files.
- [x] Complete five guides with tagline, cookies example, honest compatibility table and screenshots.
- [x] Configure GitHub funding and translated guide links with the public Revolut.me URL supplied by the maintainer.
- [x] Compile TypeScript, run suite/UI, vet, build both platforms; review final diff.
Delivery: commit, package and verify archives; push branch, no release publication.
