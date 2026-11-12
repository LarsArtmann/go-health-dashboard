# Decision Batch A — six open calls (parked for the owner)

Written 2026-11-12 (plan T24). Each entry: context, options, a
recommendation, and the blast radius. Nothing here is decided; the
recommendations exist so a one-word reply can unblock a row. Related
session questions are appended so everything rides one reply.

## A1. Release cadence for chore-only deltas

**Context.** The fleet cuts releases when the tree is green and
user-facing content rides. Chore-only deltas (docs, gate scripts, CI
tweaks) currently wait for the next real release, so `Version` const and
the tag drift apart from the tree for days — and the v0.13.3 incident
(three changes riding the train without CHANGELOG entries) showed the
drift's cost. Includes the related question: backfill the three v0.13.3
entries in `[Unreleased]` and ship them with the next cut, or re-cut
v0.13.4 now (the released CHANGELOG is otherwise accurate; the backfill
block already sits in `[Unreleased]`).

**Options.**
1. **Batched (status quo + backfill):** chore deltas ride the next real
   release; the standing `[Unreleased]` block accrues them. Cheap, but
   consumers can't pin "the docs/gates state" to a version.
2. **Cadence rule:** cut a patch release within 48h whenever the tree is
   green AND only chore deltas are pending — keeps tags truthful to the
   tree, costs a release ceremony per chore batch.
3. **CalVer-style chore tags:** tag chore batches as `-chore.N`
   prereleases. Honest but noisy for consumers.

**Recommendation.** Option 1, plus one discipline rule: a chore delta that
changes CONSUMER-VISIBLE behavior (gate scripts other repos copy, flake
outputs, NixOS module) always gets its own patch cut within 48h. The
v0.13.3 backfill rides the next cut.

**Blast radius.** CHANGELOG discipline; tag-protection ruleset untouched.

## A2. Em-dash copy for shipped strings

**Context.** The global source rule bans em dashes in code, but ~200 em
dashes survive in `.go`/`.templ` — including USER-FACING strings (error
messages, badge tooltips) and golden-pinned assertions. A mass rewrite
changes shipped copy and breaks goldens.

**Options.**
1. **Rewrite all shipped strings** to commas/periods/parentheses, update
   goldens in the same change. Clean rule, one mechanical pass + golden
   regen (~1 session).
2. **Exempt shipped copy:** the ban applies to source CODE prose
   (comments), user-facing strings keep their punctuation.
3. **Status quo:** leave as-is; the rule stays aspirational.

**Recommendation.** Option 1 — the fleet rule exists, the copy is better
for it, and goldens make the pass fully verifiable. Schedule as one
dedicated change (not a daemon sweep).

**Blast radius.** Goldens, error-string tests, `TestRender_*` snapshots.

## A3. GOEXPERIMENT=jsonv2 posture

**Context.** Every Go command requires `GOEXPERIMENT=jsonv2` (go-sse uses
`encoding/json/v2`). The experiment carried the module since v0.10; go
1.27 ships jsonv2 closer to stable. Consumers importing this library must
also set the env var — a real adoption tax (file-and-image-renamer rides
it today).

**Options.**
1. **Keep requiring jsonv2** until go-sse drops the dependency; document
   loudly (status quo).
2. **Build-tag split for SSE** (see batch B, B2) so HTML-only consumers
   can import without the jsonv2 graph.
3. **Drop go-sse**, hand-roll the SSE bits we use (broadcaster, stream,
   heartbeat). Weeks of work; rejected — zero-dep is already sacrificed.

**Recommendation.** Option 1 now, option 2 as the structured escape if a
second consumer chokes on the env var.

**Blast radius.** go.mod, CI env, all consumer onboarding docs.

## A4. Compose-stack CI policy

**Context.** The deploy compose stack is manually booted (once verified
2026-09-23, catching a real Dockerfile break). The decision-safe slice
shipped 2026-11-12: `compose-build` CI job builds the dashboard image on
every push. Open: how far boot verification goes in CI. The hub boot-smoke
design (`docs/design/2026-11-12_hub-boot-smoke-ci-design.md`) scopes three
tiers.

**Options.**
1. **Build-only (current):** images compile; boots stay manual.
2. **Tier 1 added:** CI boots the hub binary directly on the runner with a
   dark remote and probes the 503 contract (~15s, no policy weight).
3. **Tier 2 added:** CI boots the NixOS module in a VM (~3–5 min, needs a
   VM-capable runner).

**Recommendation.** Option 2 (Tier 1) — no policy weight, closes the
biggest boot gap; Tier 2 only if the NixOS module becomes a flagship.

**Blast radius.** CI minutes; deploy/README.md boot history.

## A5. Tag-ruleset bypass

**Context.** `protect-release-tags` (id 23637848) matches `refs/tags/v*`
with `bypass: never` — even repository admins cannot re-tag without
deleting the ruleset. The v0.12.1 re-cut needed exactly that dance.

**Options.**
1. **Keep `never` (status quo):** forced re-tags are impossible; a re-cut
   uses a new patch number (v0.10.1, v0.12.1 precedent).
2. **Admin bypass:** repository-admin as a bypass actor for emergencies.

**Recommendation.** Option 1. The no-re-tag discipline has two successful
precedents and the tag-first-then-master procedure avoids the need;
bypass power invites exactly the poison the ruleset prevents.

**Blast radius.** `scripts/set-tag-protection.sh --check` expectations.

## A6. Second-session-during-release policy

**Context.** Two live sessions raced during the v0.13.x train; daemon
commits interleaved with release edits and the release needed daemon-race
recovery recipes. Currently each session self-coordinates via handshake
(`git log`, `ls docs/status/ | tail`) but no rule says STOP.

**Options.**
1. **Release exclusivity:** once a release train starts (gate 0 green →
   tag pushed), other sessions in this repo stand down or stay
   docs-only until the release completes and CI is green.
2. **Lock-free coordination:** status quo — handshakes + recipes.
3. **Mechanical lock:** a repo-level `.release-lock` file the cutting
   session holds; other sessions check it before writes.

**Recommendation.** Option 1 as doctrine (AGENTS release-discipline
extension), option 3 only if doctrine fails twice more.

**Blast radius.** AGENTS.md release-discipline section; session protocols.

---

## Related session questions (answered in the same reply, please)

- **§g1 changelog backfill vs v0.13.4 re-cut** → A1 option 1 (backfill
  rides the next cut).
- **§g2 chromedp patch policy:** current policy = ignore chromedp
  MAJOR+MINOR bumps (API-migration ceremony); PATCH bumps still arrive via
  dependabot. Keep patches open, or fully manual (ignore `patch:` too)?
  Recommendation: keep patches open — they cannot break the API contract
  and the guard's stamp check catches bundle drift.
- **§g3 cqrs-htmx `go.work`:** the 33 stale go-cqrs-lite replaces are
  fixed locally (uncommitted, sibling repo). Push that fix to cqrs-htmx
  master? Recommendation: yes — it unbreaks their battery for every
  contributor; the Playwright-Chromium-on-NixOS shim stays a separate
  question (their `/mnt/buildcache` binary cannot load libglib).
