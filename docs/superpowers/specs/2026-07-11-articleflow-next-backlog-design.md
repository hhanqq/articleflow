# Articleflow Next Backlog Design

## Goal

Turn Articleflow from a working parser/search prototype into a usable personalized article feed product. The next track is split into seven implementation stages that can be shipped independently and committed after each stage.

## Approved Stages

1. **Source quality**
   Improve existing parsers and add a source registry foundation. Priority sources are Habr, vc.ru, Dzen, RSS feeds, and later custom sources.

2. **Freshness controls**
   Add date filters to parser jobs and feed queries so users can search by freshness windows such as day, week, month, year, and custom ranges.

3. **Product feed UX**
   Make the web UI closer to a comfortable vertical article feed: better scrolling, article reader flow, save/hide actions, and clearer source/status feedback.

4. **Admin source registry**
   Add internal controls for managing parser sources: enabled state, source strategy, last success, last error, limits, and health status.

5. **Production observability**
   Extend metrics and dashboards around parser quality, source latency, accepted/filtered counts, errors, feed latency, and Kafka lag.

6. **Users and personalization**
   Move from anonymous reaction signals toward real user profiles, user-specific interests, and personalized feed scoring.

7. **Load and end-to-end validation**
   Add Playwright UI coverage, runtime e2e scenarios, and load checks for parser, feed, and long scrolling sessions.

## Execution Order

Stages should be implemented in the listed order. Source quality and freshness controls come first because they directly affect feed usefulness and solve the current problem of irrelevant or stale results. UI and admin controls follow once the API behavior is stable. Observability, user personalization, and load validation harden the product after the core workflow is useful.

## Stage 1 Scope

Stage 1 should avoid a large rewrite. It should improve source quality through focused changes:

- keep Dzen HTML parsing and stale filtering stable;
- improve vc.ru discovery pagination beyond the first HTML page where possible;
- keep Habr working as a baseline source;
- add source-level configuration hooks that later stages can expose in admin UI;
- preserve safe fallback behavior when an external source blocks, redirects, or returns unexpected HTML.

## Stage 2 Scope

Freshness must be represented in the shared query model instead of being only frontend state. Parser jobs and feed queries should accept normalized freshness windows and translate them into `FromDate` and `ToDate` values. Sources that cannot enforce date filters at request time should filter candidates after parsing.

## Stage 3 Scope

The product feed should remain dependency-light for now. It should add practical controls and states rather than a visual redesign: source chips, freshness control, loading/error states, article reader view, and feed actions that can later become personalization signals.

## Non-Goals

- No paid external API dependency is required for this track.
- No scraping bypasses or anti-bot circumvention should be added.
- No full authentication system is required before Stage 6.
- No mobile app is required; the web SPA remains the primary client.

## Success Criteria

- Each stage has tests and a separate git commit.
- `make ci` passes after each stage.
- Live Docker app startup remains valid through `make app-up`.
- Parser job results show source diagnostics that explain what each source returned and filtered.
- The web UI makes it clear which sources and freshness settings produced the feed.

## Self-Review

- No placeholders remain in this spec.
- The seven stages are ordered by product dependency and implementation risk.
- The first two stages are narrow enough to begin implementation without redesigning the whole system.
