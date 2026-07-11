# Articleflow Product Stages

## Saved Stages

8. **Auth and User Profiles**
   Replace manual `user_id` entry with a gateway-managed user context, anonymous sessions, `/api/v1/me`, and persisted user profiles.

9. **Personal Ranking**
   Use reactions, sources, tags, and article text to rank the feed per user.

10. **Source Registry V2**
   Persist source settings, health, last success/error, limits, and custom RSS sources in Postgres.

11. **More Sources**
   Add more article sources after the registry can manage them safely.

12. **Better Infinite Feed**
   Improve cursor pagination, prefetch, reader flow, saved articles, and long-scroll UX.

13. **Production Observability**
   Add dashboards, alerts, Kafka lag visibility, and source-level runtime health.

14. **CI/CD and Deploy**
   Build images, run stage deploys, manage secrets, run migrations, and execute runtime e2e in CI.

15. **Parser Quality and Anti-Duplicates**
   Improve canonical URLs, deduplication, stale filters, full text extraction, and per-source reliability.

## Current Execution

Stage 8 is the active stage. It should provide an anonymous-session foundation without adding password auth or OAuth yet.
