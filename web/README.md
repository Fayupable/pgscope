# pgscope, frontend

React + Vite + TypeScript dashboard that consumes the backend's live SSE stream, its advisory Insights endpoint, and its authentication endpoints. Feature-based folder structure: each screen or concern lives in its own `features/*` folder with its own components and hooks; shared infrastructure (SSE client, API clients, types, design tokens) lives in `shared/`.

Backend setup is documented in the [root README](../README.md) and [go/README.md](../go/README.md); the backend must be running for this to show anything, and it must be reachable with a valid `PGSCOPE_API_KEY`, since every screen except the login screen requires an authenticated session.

## Prerequisites

- Node.js 18+
- The Go backend running (see [go/README.md](../go/README.md)), with `PGSCOPE_API_KEY` set

## Setup

    npm install
    npm run dev

Open `http://localhost:5173`. Vite proxies every `/api/*` request to `http://localhost:8090` (see `vite.config.ts`), so no CORS configuration is needed in dev, and the frontend code never hardcodes the backend's origin anywhere.

## Directory structure

    src/
      features/
        auth/
          components/    — LoginScreen
        sessions/
          components/    — SessionTable, SessionList
          hooks/           — useSessionStream
        lock-graph/
          components/     — SessionGraph, SessionDetailPanel, EdgeDetailPanel, ClusterHulls, GraphLegend
        db-stats/
          components/     — DbStatsBar
          hooks/            — useDbStatsStream
        replay/
          components/     — ReplayControls
          hooks/            — useSnapshotPlayer
        insights/
          components/     — InsightsPanel, TopQueriesTable, IndexCandidatesTable, DuplicateIndexesTable, UnusedIndexesTable, FunctionCostsTable, PaginationWarningsTable, HealthPanel and its cards (ConnectionSaturationCard, DatabaseSizeCard, SequenceOverflowTable, InvalidObjectsCard, VacuumHealthTable, IdleInTransactionTable, CheckpointHealthCard, ReplicationLagTable, PhysicalIOHotspotsTable)
          hooks/            — useInsights
          utils/            — formatBytes, formatDuration
        dashboard/
          components/     — Dashboard (composition root), MonitoringControls
      shared/
        api/
          authClient.ts                — login, logout, checkAuthStatus
          sseClient.ts                  — low-level EventSource wrapper, generic event dispatch
          monitoringStreamContext.ts    — React context + useMonitoringStream() hook
          MonitoringStreamProvider.tsx  — opens exactly one SSE connection for the whole app
          controlClient.ts              — fetch wrappers for /monitor, /record, /history endpoints
          insightsClient.ts             — fetch wrapper for /insights
        types/
          session.ts     — mirrors the backend's Session/LockedObject JSON shape
          snapshot.ts     — mirrors the backend's Snapshot JSON shape
          insights.ts      — mirrors the backend's Insights JSON shape and every sub-type inside it
        utils/
          sessionStatus.ts       — derives a normalized status (active/blocked/idle/...) from a Session, shared by table and graph views
          parseSnapshotFile.ts    — validates and parses an uploaded history JSON file
          graph/                   — pure layout math for the blocking graph (bridge detection, per-cluster force layout, packing, hull computation), see "How the graph layout works" below
        styles/
          tokens.css     — every color, spacing value, and radius used anywhere in the app, as CSS custom properties

## Authentication

`App.tsx` is the entry point for the whole login flow. On mount it calls `checkAuthStatus()`, which hits `GET /api/v1/auth/status`, a backend route that requires a valid session cookie and returns `200` if one is present or `401` otherwise. If it returns `401`, `App.tsx` renders `LoginScreen` instead of the dashboard; if it returns `200`, it renders the dashboard directly, meaning a reload does not force the user to log in again as long as the session cookie is still valid.

`LoginScreen` collects the API key and posts it to `POST /api/v1/auth/login`. On success, the backend sets a session cookie and the frontend never touches the key again, it does not store it in `localStorage`, `sessionStorage`, or any application state. This was a deliberate choice over the more common pattern of keeping an auth token in browser storage and attaching it to each request manually: any storage API that JavaScript can read (`localStorage`, `sessionStorage`) can also be read by an attacker's script if the page ever has a cross-site-scripting vulnerability, whereas a cookie marked `HttpOnly` by the backend is invisible to JavaScript entirely, including to an attacker's injected script. Because the frontend and backend are same-origin (the Vite dev proxy and, in production, a single reverse-proxied domain), the browser attaches this cookie automatically to every subsequent `fetch` and to the `EventSource` connection without any code needing to read or forward it, which is also why `authClient.ts`, `controlClient.ts`, and `insightsClient.ts` all pass `credentials: 'include'` explicitly rather than relying on the default, so the same code keeps working correctly if the frontend and backend are ever split across different subdomains later.

The Dashboard's "Log out" button calls `POST /api/v1/auth/logout`, which clears the cookie, and returns the user to `LoginScreen`.

## How the app is wired

- `App.tsx` gates everything behind the authentication check described above, then wraps the dashboard in `MonitoringStreamProvider`, which opens a **single** SSE connection (`/api/v1/sessions/stream`) for the entire app and fans `sessions`/`db_stats` events out via React context. Every component that needs live data (`useSessionStream`, `useDbStatsStream`) reads from this one connection instead of each opening its own; this was a deliberate fix, since the naive approach of each hook opening its own `EventSource` works but wastes a connection per consuming component for no benefit.
- `SessionList` and `SessionGraph` both accept an optional `sessionsOverride` prop. When absent, they read from the live stream via their own hook. When present (Replay mode passes the currently-playing snapshot's sessions), they render that instead. This is the entire mechanism behind Live/Replay switching; neither component needed to be duplicated or forked.
- `Dashboard.tsx` is the composition root: it owns the Live/Replay/Insights toggle, the List/Graph toggle, and decides what gets passed down as `sessionsOverride`.

## Features

### Monitor / Record controls

`MonitoringControls` calls the backend's `/monitor/start`, `/monitor/stop`, `/record/start`, `/record/stop` endpoints. Both are off by default; the dashboard shows nothing until you pick a duration and click **Start** under Monitor. Recording is disabled in the UI until monitoring is active, mirroring the backend rule that stopping monitoring also stops recording.

### Download / Replay

**Download JSON** fetches `/api/v1/history` and triggers a browser download via a `Blob` and a temporary `<a download>` element. The downloaded file can later be loaded back in via the **Replay** tab's **Load JSON** button, parsed and validated by `parseSnapshotFile`. `useSnapshotPlayer` then exposes play/pause/seek/speed controls (0.5x to 4x) that step through the loaded snapshots on an interval, effectively a video player for a captured monitoring window.

### List / Graph views

- **List** (`SessionTable`), a dense, fixed-column table: state (as a colored left border), PID, duration, user/app, query (masked, monospace, truncated), wait event, lock severity (colored dot and count), and who's blocking whom.
- **Graph** (`SessionGraph`), built with [`@xyflow/react`](https://reactflow.dev/) (React Flow). Each session is a node colored by status; each "waiting on" relationship is a directed, animated edge. Clicking a node opens a detail panel (query, locks, wait event, duration) positioned right next to the node you clicked, not fixed to a corner. Clicking an edge opens a similar panel explaining that specific blocking relationship.

### Insights

`InsightsPanel` (`features/insights/components/InsightsPanel.tsx`) is a tabbed view over the backend's `GET /api/v1/insights` response, fetched once on mount and again on demand via a **Refresh** button, since this endpoint is deliberately rate-limited and expensive on the backend (see [go/README.md](../go/README.md)'s Insights section), so it is not polled automatically the way the live session stream is.

Each tab renders one category returned by the backend, with its own table or card component: Top Queries, Index Candidates, Duplicate Indexes, Unused Indexes, Functions and Triggers, Pagination Warnings, and a combined Health tab. The Health tab (`HealthPanel.tsx`) groups several smaller, related signals together: connection saturation, database size, sequence overflow risk, invalid indexes and unvalidated constraints, vacuum health, idle-in-transaction sessions, checkpoint health, replication lag, and physical I/O hotspots (the last one only shown as available if the optional `pg_stat_kcache` extension is installed on the target database, mirrored from the `physicalIOEnabled` flag the backend already computes).

Every row rendered in these tables includes the same plain-language explanation string the backend generated, rather than the frontend trying to re-derive or rephrase it. This keeps the wording (always phrased as something to verify, never as a certainty) consistent regardless of which screen it's shown on, and means a threshold change in the backend's `domain` layer is reflected everywhere automatically without a matching frontend change.

## How the graph layout works

With more than a handful of sessions, simply placing every node evenly around one circle produces long edges crossing the whole canvas and no visual grouping between unrelated clusters of activity. The layout pipeline in `shared/utils/graph/` avoids that in four steps, run once per render of the graph view (React Flow itself has no physics engine; this pipeline hands it a finished, static `{x, y}` position for every node):

1. **Bridge (cut-edge) detection** (`edges.ts`), before laying anything out, finds edges whose removal would split the graph into two pieces that are each genuinely their own cluster (more than one node on each side, and each side has its own high-degree "hub" node). An edge meeting that bar is treated as a bridge: the two clusters it connects are laid out independently, and the bridge itself is drawn afterward as one thin connecting line between them, instead of a naive connected-components pass merging everything on either side of it into one tangled blob. A single stray cross-link on an otherwise unrelated plain chain does not qualify as a bridge on its own, which is why the rule requires a hub on both sides rather than just one; see the comment above `findBridges` in `edges.ts` for the exact reasoning and a concrete counterexample.
2. **Per-cluster, hub-centered layout** (`clusterLayout.ts`), within each remaining cluster, the node with the highest degree (most connections, for example a session blocking many others) is pinned toward the center, with the rest seeded on a jittered circle around it and relaxed with `d3-force` (`forceLink`, `forceManyBody`, `forceCollide`) for a fixed number of ticks.
3. **Non-overlapping tiling** (`packing.ts`), once each cluster has its own internal layout, clusters are placed on the canvas so their bounding boxes don't overlap, using simple row-based packing rather than a full bin-packing library, since the graph sizes involved here don't need one.
4. **Isolated nodes** (`singletonLayout.ts`), sessions with zero relationships are scattered organically in a reserved area below the clusters using repulsion and collision only (no links, since there's nothing to connect), rather than being run through the same force simulation as connected clusters, since a force simulation with only charge and no links has nothing to arrange them around.

`index.ts` composes all four steps into `computeClusteredLayout()`, the single function `SessionGraph.tsx` calls.

## Design system

All colors, spacing, and radii are CSS custom properties defined once in `shared/styles/tokens.css` and referenced everywhere else via `var(--...)`; no component's CSS file should ever contain a raw hex color or hardcoded pixel spacing value. This was a deliberate refactor after the first pass of components had colors duplicated across multiple files; centralizing them means a palette change only ever touches one file. `stylelint` (`.stylelintrc.json`) enforces the no-raw-hex-color rule automatically, with `tokens.css` itself the one exception, since that's where the actual hex values have to live.

Lock severity and session status both use a small, colorblind-safe palette (green to amber to red by severity, not just hue) so the graph and table read correctly at a glance without relying on color alone; shape and position (left-border stripe, dot size) carry some of that signal too.

## Testing and linting

    npx tsc -b
    npm run lint
    npm run lint:css

`eslint.config.js` enforces one architecture rule beyond the standard React/TypeScript recommendations: code under `src/features/**` cannot call `fetch()` or construct `EventSource` directly, every network call must go through a wrapper in `shared/api/`. This is what keeps credentials handling (`credentials: 'include'`), error shaping, and the request pattern itself in one place instead of duplicated per caller.

## Known limitations

- The graph does not yet handle 100+ node graphs gracefully; the clustering pipeline above helps considerably but has not been tuned or tested at that scale.
- `useSessionStream`/`useDbStatsStream` currently render even when a component only needs replay data. This is harmless (the shared SSE connection has near-zero idle cost) but slightly redundant, and not worth optimizing before the graph-scaling work above.