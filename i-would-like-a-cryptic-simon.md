# GoCs — Status Summary & Feature Roadmap

## Context

This is a CS2 demo-analysis web app: Go/Fiber backend (`server/`, split into `auth`, `matches`, `team`, `season`, `player`, `parser` packages) + React/TS/Vite frontend (`client/src/`), MySQL schema in `init.sql`. The last commit (`2d3e0dc`) split the monolithic `main.go` into per-domain route packages, added the Season UI (`SeasonsTab.tsx`, `SeasonMatches.tsx`), added a standalone `Whiteboard.tsx` drawing page, and expanded team stats. The user asked for (1) a status check — what's built, what's stubbed/half-wired, especially since that last commit — and (2) a phased roadmap for 8 requested features. Three Explore agents audited auth/cookies, upload/season/stats, and playback/drawing/notes/search; findings below are drawn directly from that audit (file:line references throughout).

Two scope decisions were confirmed with the user up front:
- **2D playback drawings persist per round** (saved to backend, revisitable later) rather than being session-only.
- **Roadmap order**: bug fixes first, then the **season system fix** next (it unblocks accurate season-scoped data used by later features), then stats, then notes/search, then drawing/heatmaps.

A late addition to scope: the user wants the 2D playback to **eventually** support multiple teammates watching the same session together, in sync. This is flagged as its own phase (Phase 6) at the end of the roadmap, since — unlike everything else here — it requires real-time infrastructure (websockets, live session state) that doesn't exist anywhere in the app today.

---

## Current State Summary

**Solid and working:** per-match advanced stats (entries/utility/plants&retakes/clutches — `AdvancedStatsSubTab.tsx` + `server/matches/handlers.go`), 2D tick-by-tick playback via `react-konva` (`DemoPlayback.tsx`, bulk-fetches a round's data from `/api/match/2DPlayback/:matchRound`, pre-transforms map coords), team career roster stats and round-economy stats (`TeamTab.tsx`/`AdvancedTab.tsx` + `server/team/handlers.go` + `auth.GetTeamRoundStats`), demo upload/save, Steam OAuth login.

**Half-wired since the last commit:**
- `SeasonsTab.tsx` (season create/edit UI) calls `POST /api/team/seasons` and `PUT /api/seasons/:id` — **neither route exists server-side**. The UI renders but every create/edit action 404s.
- `MATCHES.SEASON_ID` (FK → `SEASONS.SEASON_ID`) is declared and *read* by `server/season/handlers.go`, but **nothing ever writes it**. Every match's `SEASON_ID` is NULL, so `SeasonPage.tsx` (`/season/:id`) always shows zero matches even though seasons list fine.
- `AdvancedStatsSubTab.tsx` / `client/src/helpers/types.ts` (new, untracked) are **not stubs** — fully wired to real per-match endpoints. The gap is one level up: there's no team-wide (career-spanning) equivalent for entries/utility/clutches, only per-match.
- `auth.GetTeamRoundStats` already computes team-wide bomb-plant-win%, retake%, and anti-eco% (`server/auth/handlers.go:303-365`) but `team/handlers.go`'s `Summary`/`Advanced` never include those fields in the JSON response — dead data, one-line-per-field wiring fix.
- `ROUNDS.WIN_REASON` is parsed and stored but never read anywhere in Go or TS.

**Stubs with no backend at all:**
- `DemoPlayback.tsx`'s "Drawing mode" and "Settings" buttons (lines ~1633-1634) have no handlers. Its "Notes" drawer (line 1693) is a bare `<textarea>` with no `value`/`onChange`/save.
- `PlayerPage.tsx`'s "Team Notes" section (`GameNote`/`UnparsedNote`, lines 30-48/84-101/417-434) renders UI for notes the backend (`server/player/handlers.go: MeStats`) never populates.
- `Whiteboard.tsx` is a fully working freeform Konva canvas (pen, rect/circle, eraser, color picker, map background) but is 100% client-local — no save/load, not tied to any match or round.
- No dropped-weapon/kit tracking exists in the parser at all; the underlying `demoinfocs-golang` library exposes `ItemDrop`/`ItemPickup`/`ItemEquip` events (noting they're "not available in all demos") but none are wired up — only the bomb's drop/pickup is tracked.
- No round search/filter feature exists anywhere, though the schema (`ROUNDS.SITE/BUY_TYPE_T/BUY_TYPE_CT/BOMB_PLANT/WINNING_SIDE`, `MATCHES.MAP`) already supports the filters requested without any migration.

**Auth bugs (confirmed root causes, detailed in Phase 0):** no logout exists anywhere; `PlayerPage.tsx` shows a generic broken-looking error (not an onboarding prompt) when `STEAMID` is null; a cookie-name-blind bug in `Login.tsx` misroutes returning users to the Register panel.

---

## Phase 0 — Auth/session bug fixes

**Goal:** fix the three reported login/cookie problems.

1. **Add real logout.** New `POST /auth/logout` in `server/auth/handlers.go` + `server/auth/routes.go` that re-sets `auth_token` with an expired `MaxAge`/past `Expires` (mirror the cookie config already used in `Login`/`Register`, `handlers.go:198-205,272-279`). Add a logout action in `Navbar.tsx` (currently has no logout UI at all) that calls it and also clears the stray non-HttpOnly `steamLinked` cookie (set in `PlayerPage.tsx:206`, never cleared today).
2. **Fix empty/broken profile page when no SteamID.** `server/player/handlers.go:61-63` currently returns HTTP 400 for `/api/player/me/stats` when `STEAMID` is NULL (true for every email/password `Register` account, since `Register` never sets `STEAMID` — `handlers.go:173-176`). Change it to return 200 with a "no steam linked" flag instead of a hard error, and in `PlayerPage.tsx` route that state to the existing (currently dead) `OnboardingModal` component (lines 170-286) instead of falling into the generic `catch` → error screen (line ~304-330). Wire up `showModal` (currently hardcoded `useState(false)`, the intended trigger logic is commented out at lines 313-320) so first-time/unlinked users see a "link your Steam account" prompt instead of a broken page.
3. **Fix stale-cookie mis-routing to Create Account.** `Login.tsx`'s `getCookies(name)` (lines 25-33) ignores its `name` argument and just reads whatever cookie is first in `document.cookie` — fix it to actually look up the named cookie. Since there's no real `new_account` cookie/signal today, replace this whole mechanism: after Phase 0.1 ships real logout (which will also clear `steamLinked`), have `Login.tsx` decide the login vs. register panel from the auth state via `useAuth()` (or a `?register=1` query param from an explicit "Create account" link) rather than cookie-sniffing. Also add an "already logged in" guard in `App.tsx`'s `Home()` (lines 12-19) so it redirects to `/accountHome` instead of always rendering `<Login/>`.
4. Minor: remove or implement the "Continue as Guest" button in `Login.tsx` (`handleGuest`, posts to unregistered `/auth/guest`, 404s today).

**Critical files:** `server/auth/handlers.go`, `server/auth/routes.go`, `client/src/components/Navbar.tsx`, `client/src/components/PlayerPage.tsx`, `client/src/components/Login.tsx`, `client/src/App.tsx`, `client/src/context/AuthContext.tsx`.

---

## Phase 1 — Season system fix (dropdown + working CRUD)

**Goal:** make season selection at upload time a real dropdown, backed by seasons the user actually created, and make season-scoped pages work.

1. **Add the missing season CRUD endpoints** the frontend already expects: `POST /api/team/seasons` (create) and `PUT /api/seasons/:id` (edit) — `SeasonsTab.tsx` (lines 23, 35) already calls these and 404s today. Implement in `server/season/handlers.go`/`server/team/handlers.go` following the existing `Seasons()` read handler pattern (`team/handlers.go:613-650`).
2. **Write `MATCHES.SEASON_ID` on upload.** Change `UploadDemoModal.tsx`'s free-text "Season" `<input>` (lines 93-101) to a `<select>` populated from `GET /api/team/seasons`, enforcing "create a season first." Submit `season_id` instead of a raw string. Update `UploadDemo` in `server/team/handlers.go` (lines 508-526) to `SET SEASON_ID = ?` (keep writing legacy `SEASON` text too, or backfill/deprecate it — recommend keeping both in sync short-term to avoid breaking `Demos()`'s existing read of `SEASON`).
3. Note the upload handler is an `UPDATE ... WHERE DEMO_NAME` that assumes a `MATCHES` row already exists — verify this still holds once season becomes a required pre-upload step, since a season must exist but a match row's creation timing is unrelated.

**Critical files:** `client/src/components/UploadDemoModal.tsx`, `client/src/components/SeasonsTab.tsx`, `server/season/handlers.go`, `server/season/routes.go`, `server/team/handlers.go` (`UploadDemo`, `Seasons`).

---

## Phase 2 — Team UI emphasis: plants, retakes, gun/eco rounds, impact rating

**Goal:** surface the stats the user called out (bomb plants, retake conversion, gun rounds, T/CT round win rates, an "impact round win rate %") on the team advanced stats page.

1. **Wire up already-computed fields.** `auth.GetTeamRoundStats` (`server/auth/handlers.go:303-365`, struct in `server/model/server_structs.go:10-20`) already computes `TPlants/TPlantWins/TPlantWinPct`, `RetakeOpportunities/RetakeWins/RetakeSuccessPct`, and `AntiEcoRounds/AntiEcoWins` — none of these appear in the JSON returned by `Summary` (lines 196-216) or `Advanced` (lines 296-306) in `server/team/handlers.go`. Add them to both response maps and render new stat cards in `AdvancedTab.tsx` (plant conversion %, retake conversion %, anti-eco win %).
2. **Add "gun round" framing.** "Gun round" = non-pistol round (force or full buy); this is directly derivable from the existing `BUY_TYPE_T/BUY_TYPE_CT` breakdown already computed — add a combined force+full "gun round win %" stat alongside the existing separate force/full numbers.
3. **Use `ROUNDS.WIN_REASON`** (parsed, stored, never read) to add an elimination-vs-bomb-vs-time win-condition breakdown.
4. **Reconcile the "impact round win rate %" ask** with the two existing, inconsistent rating formulas: a simplified one in `matches/handlers.go`'s `buildPlayer` (lines 198-214, explicitly not HLTV 2.0) and the proper HLTV-2.0-style one in `team/handlers.go` (`Summary`/`PlayerStats`, lines 145-152/427-428). Standardize on one formula (recommend the HLTV-2.0-style one) and surface team-level "% of rounds where [player/team] had positive impact" using it.
5. **Optional stretch (flagged, not required for this phase):** team-level per-site (A/B) plant/retake breakdown and team-level entries/utility/clutch aggregation — these only exist per-match today (`matches/handlers.go`'s `EntryStats`/`UtilityStats`/`ClutchStats`); a team-wide version would mean adapting those queries to `SUM`/aggregate across all of a team's matches, analogous to how `GetTeamRoundStats` already aggregates rounds. Worth a follow-up phase once Phase 2's core wiring ships.

**Critical files:** `server/auth/handlers.go` (`GetTeamRoundStats`), `server/team/handlers.go` (`Summary`, `Advanced`), `client/src/components/AdvancedTab.tsx`, `client/src/components/TeamTab.tsx`.

---

## Phase 3 — Round notes (categorized, timestamped) + round search

**Goal:** per-round notes with user-defined categories and optional tick ranges, plus a round-search page supporting queries like "postplant losses on Mirage" or "eco round losses."

1. **New schema** (append to `init.sql`, additive — no changes to existing tables besides is unnecessary since `ROUNDS`/`MATCHES` already carry everything needed for scoping):
   - `NOTE_CATEGORIES(CATEGORY_ID PK, TEAMNAME FK, NAME, COLOR, CREATED_AT)` — user-created, scoped per team (matches the user's ask: "categorization system for the user to create").
   - `ROUND_NOTES(NOTE_ID PK, MATCHID, ROUND_NO FK→ROUNDS, AUTHOR_ACCOUNTID, CATEGORY_ID FK NULL, TICK_START INT NULL, TICK_END INT NULL, NOTE_TEXT, CREATED_AT)` — `TICK_START`/`TICK_END` support the "range of time where the comment is highlighted" ask; both NULL means a whole-round note.
2. **Backend:** new `server/notes/` package (handlers + routes, following the existing per-domain package pattern) — CRUD for categories and notes, plus `GET /api/notes/round/:matchId/:round` and a `GET /api/notes/category/:categoryId` "all notes in this category" query (the user's explicit ask).
3. **Frontend:** replace the two existing dead stubs rather than building fresh UI from scratch — wire the "Notes" drawer already present in `DemoPlayback.tsx` (line 1693) to save/load real notes scoped to the current match+round, with tick-range selection tied to the existing playback timeline/scrubber. Populate the already-built "Team Notes" section in `PlayerPage.tsx` (`GameNote`/`UnparsedNote`, currently always empty because `MeStats` never sets those fields) or repurpose it into a category-browsing view.
4. **Round search:** new backend endpoint, e.g. `GET /api/team/rounds/search`, with query params for map, side, buy type, bomb-plant/postplant-loss, win reason, site, etc. — all directly filterable off existing columns (`MATCHES.MAP`, `ROUNDS.WINNING_SIDE/BOMB_PLANT/BUY_TYPE_T/BUY_TYPE_CT/SITE/WIN_REASON`) with no schema changes needed. "Postplant" is a derived phase (join `ROUND_EVENTS` where `EVENT_TYPE = BombPlanted` for the plant tick, same pattern `DemoPlayback.tsx` already uses client-side at lines ~1856-1874), not a stored flag. New dedicated frontend page (e.g. `RoundSearch.tsx`) with a filter builder and a results table linking each match into `SeasonMatches`/`DemoPlayback` at the specific round.

**Critical files (new):** `server/notes/handlers.go`, `server/notes/routes.go`, new client page for search, `init.sql` additions. **Existing files touched:** `client/src/components/DemoPlayback.tsx` (Notes drawer), `client/src/components/PlayerPage.tsx` (Team Notes section), `server/player/handlers.go` (`MeStats`).

---

## Phase 4 — 2D playback drawing (persisted) + toggleable dropped weapons/kits

**Goal:** telestrator-style drawing on the 2D map (pen w/ adjustable color, predefined smoke/molly stickers, explicit toggle for draw mode), saved per round; plus optional rendering of dropped weapons/kits.

1. **Drawing engine reuse.** `Whiteboard.tsx` already has a working Konva-based pen/shape/color-picker/eraser implementation — lift that drawing logic into a shared component/hook usable by both `Whiteboard.tsx` and a new drawing overlay `Layer` in `DemoPlayback.tsx`, rather than reimplementing. Wire `DemoPlayback.tsx`'s existing but inert "Drawing mode" button (line ~1633) to toggle this overlay (per the user's explicit ask: drawing must be behind a toggle, not always-on, since it would otherwise interfere with tick scrubbing/interaction).
2. **Predefined objects:** add a small palette of placeable icons (smoke, molotov, plus reuse existing grenade icon assets from `DemoPlayback.tsx`'s grenade rendering if available) as draggable Konva `Image`/`Group` nodes, distinct from the freehand pen tool.
3. **Persistence** (per the confirmed decision): new table `ROUND_DRAWINGS(DRAWING_ID PK, MATCHID, ROUND_NO, AUTHOR_ACCOUNTID, SHAPE_DATA JSON, CREATED_AT)` — store the Konva shape/line/sticker list as JSON (mirrors how `Whiteboard.tsx` already holds `lines`/`shapes` as serializable state). New `server/drawings/` package (or extend `server/notes/` from Phase 3 since both are round-scoped annotations) with save/load endpoints keyed by `(matchId, round)`.
4. **Dropped weapons/kits (toggleable layer):** requires new parser work — register `ItemDrop`/`ItemPickup` handlers in `server/parser/parsed_2d.go` mirroring the existing `BombDropped`/`BombPickup` pattern (lines 283-325), a new model/table (e.g. `WEAPON_DROP_EVENTS`, parallel to `GRENADE_EVENTS`) since reusing grenade machinery would be a naming/semantics mismatch, and a new `RedrawAtTick` rendering branch in `DemoPlayback.tsx` gated behind a toggle button (separate from draw mode) to avoid clutter, per the user's ask. **Caveat to flag to the user before starting:** the underlying `demoinfocs-golang` library's own docs note `ItemDrop` "is not available in all demos" — this feature may render incompletely on some older/edge-case demo files; worth prototyping against a couple of real demos early to confirm coverage before committing to full UI polish.

**Critical files:** `client/src/components/Whiteboard.tsx`, `client/src/components/DemoPlayback.tsx`, new `server/drawings/` (or extended `server/notes/`) package, `server/parser/parsed_2d.go`, `server/model/parser_structs.go`, `init.sql` additions.

---

## Phase 5 — Heatmaps (per-match and accumulated)

**Goal:** heatmaps for a single match and "accumulative" heatmaps across many rounds/matches (e.g. a season, or a filtered set from Phase 3's round search).

1. **Data source:** `PLAYER_EVENTS` already stores per-tick `XPOS/YPOS/ZPOS` for every player in every round (`init.sql` lines 194-223) — no new parsing needed for a positional/death heatmap. Kill/death locations specifically can be derived from `ROUND_EVENTS` (`EVENT_TYPE = PlayerKilled`) joined to the killer/victim's position at that tick.
2. **Per-match heatmap:** new endpoint aggregating `PLAYER_EVENTS`/death positions for one `matchId`, rendered as a density overlay `Layer` on top of the existing map-image + coordinate-transform logic already built in `DemoPlayback.tsx` (`newX`/`newY` closures, ~lines 1480-1485) — reuse that transform rather than rebuilding it.
3. **Accumulated heatmap:** same aggregation but parameterized over a set of matches/rounds — natural to build on top of Phase 3's round-search filters (map, side, season, outcome) so a user can generate e.g. "heatmap of all T-side deaths in postplant losses on Mirage this season." Recommend sequencing this after Phase 3 lands so the filter engine can be reused rather than duplicated.
4. Performance note: aggregating raw tick-level `XPOS/YPOS` across many matches server-side (SQL `GROUP BY` binned coordinates, or stream-and-bin in Go) will scale better than shipping raw points to the client for large date ranges — worth deciding the binning resolution/approach when this phase is actually scoped in detail.

**Critical files:** new heatmap endpoint(s) in `server/matches/` or a new `server/heatmaps/` package, `client/src/components/DemoPlayback.tsx` (coordinate transform reuse), new accumulated-heatmap view (likely alongside the Phase 3 search page).

---

## Phase 6 — Synchronized multi-viewer playback ("watch together")

**Goal:** let multiple teammates watch the same 2D playback session simultaneously, in sync (same round/tick), for remote coaching review. Flagged by the user as an "eventually" feature — included here so the requirements are understood in advance, not necessarily to build immediately.

This is a different category of work from everything else in this roadmap. Confirmed during the earlier audit: `DemoPlayback.tsx` only does a one-shot bulk HTTP fetch per round (`fetch(...api/match/2DPlayback/...)`, then walks the data tick-by-tick locally via a `Konva.Animation` loop that mutates refs directly for performance) — there is **no websocket or any real-time transport anywhere** in `server/` or `client/` today. Breaking down what's actually needed:

1. **Transport: WebSockets.** Fiber supports this via `github.com/gofiber/contrib/websocket`. Needs a new package (e.g. `server/live/`) with a `/ws/playback/:sessionId` upgrade endpoint — net-new infrastructure, nothing like it exists in `main.go`'s current route registration.
2. **Session/room concept.** Define a "watch session": which match+round it's watching, and who's allowed to join. Simplest model: scope to `TEAM_MEMBERS` (reuse the same team-membership check as `middleware.RequireTeam`) — any teammate can join a session for a match their team owns. Needs a "Watch together" action in `DemoPlayback.tsx` that starts/generates a session, and a lightweight in-memory registry on the server (sessionId → connected clients + current state; a mutex-guarded map is enough for a single server instance — flag that this won't work across multiple server instances without a shared broker like Redis pub/sub if horizontal scaling is ever needed).
3. **Control model — recommend host-driven (leader/follower), not symmetric.** One participant (session starter, or whoever takes control) drives round selection/play-pause/scrubbing; everyone else's client receives state updates and mirrors them (their scrub bar becomes read-only/follows). This avoids conflict resolution (two people scrubbing simultaneously causing jittery playback) and matches the realistic use case — a coach or IGL narrating while teammates watch. Symmetric multi-control is possible but adds real complexity for little benefit here.
4. **What state actually syncs (kept minimal):** current round, current tick/playhead (broadcast on an interval, e.g. every 200-500ms while playing — not every frame, since each client already has the full round's tick data loaded locally per the earlier audit; you're syncing *position in the timeline*, not the underlying data), and play/pause/speed. If Phase 4's persisted drawing has landed by the time this is built, live drawing strokes could piggyback on the same socket as an additional event type, upgrading persisted-drawing into live-collaborative-drawing.
5. **Client refactor needed.** The current animation loop conflates "read local user input" with "advance the tick" (see the `Konva.Animation` loop and direct ref mutation noted in the earlier audit). A follower client needs tick-advancement driven by incoming socket messages instead of local controls — this means decoupling "what drives the tick" from the rendering code so the same playback rendering serves both host and follower roles. This is a moderate refactor of `DemoPlayback.tsx`'s control logic, not a rewrite of the rendering itself.
6. **Auth/access control.** Reuse the existing cookie-based session auth (`middleware.RequireAuth`) during the websocket upgrade handshake rather than inventing new auth, plus a team-membership check mirroring `middleware.RequireTeam`.
7. **Presence UI (nice-to-have):** show who's currently in the session and who has control, in the existing HUD chrome.

**Sequencing note:** the core round/tick sync (points 1-6, minus drawing) doesn't strictly depend on Phase 4 and could be pulled forward if desired. It's placed last here because it's the most architecturally novel piece of work in the roadmap (first real-time/multi-client feature in an otherwise request/response app) and was flagged by the user as a lower-priority "eventually."

**Critical files (new):** `server/live/` (websocket handler + in-memory session registry), `client/src/components/DemoPlayback.tsx` (host/follower control split), `server/main.go` (websocket route registration).

---

## Verification approach (all phases)

- Run via `./dev.sh` (Go server on :4000, Vite client on :5173) for manual end-to-end testing of each phase's UI flow.
- Phase 0: manually test login → logout → reopen home page (should land on Login panel, not Register); create an email/password account with no Steam link and confirm the profile page shows an onboarding prompt, not a generic error.
- Phase 1: create a season, upload a demo against it, confirm it appears under `/season/:id`.
- Phase 2: cross-check new stat cards against manually-tallied numbers from a known demo/match.
- Phase 3: create categories, add timestamped notes to a round, confirm category-filtered query returns them; run a few round-search queries against known matches and manually verify results.
- Phase 4: draw on a round, reload the page, confirm drawings persist and reload correctly; test toggle on/off for both drawing mode and dropped-items layer.
- Phase 5: sanity-check heatmap density visually against known hot spots (site chokepoints) for a familiar map/demo.
- Phase 6: open the same session in two browser sessions (or two browsers), confirm round/tick/play-pause state mirrors from host to follower within the sync interval, and that a follower's local controls don't fight the host's state.
