# LaundryPro Admin — Frontend

Premium SaaS admin console for the LaundryPro backend. Built with
**React 19 + Vite 8 + Redux Toolkit + TanStack Query + Tailwind CSS v4 + Recharts**.

## Quick start

```bash
cd frontend
npm install
cp .env.example .env.local   # adjust VITE_API_BASE_URL / VITE_API_PROXY if needed
npm run dev                  # http://localhost:5173
```

The Vite dev server proxies `/api` to the backend (default
`http://localhost:9090`), so no CORS configuration is needed locally.

## Scripts

| Command            | Purpose                                  |
| ------------------ | ---------------------------------------- |
| `npm run dev`      | Dev server with HMR + API proxy          |
| `npm run typecheck`| TypeScript strict check (`tsc -b`)       |
| `npm run build`    | Production build (typecheck + bundle)    |
| `npm run preview`  | Preview the production build             |

## API alignment

The frontend **strictly** follows the backend contracts:

- **Envelope** — every response is
  `{ code, message, data, pagination, summary, request_id }`; the axios
  interceptor unwraps it and throws a typed `ApiError` on failures.
- **Methods** — `POST`, `PATCH`, `DELETE` only (per the backend). All list
  endpoints are `POST /<resource>/list` with the body carrying the page
  request: `{ cursor, limit, sort_by, sort_order, search, filters }`.
- **Pagination** — cursor-based. `useListQuery` keeps a per-page cursor cache
  so you can move forward/backward between pages; `pagination.total` powers
  the *"Showing 1–20 of 145"* summaries.
- **Filters** — enum filters send `eq:`-prefixed values; date ranges send
  `created_at_from` / `created_at_to` keys — exactly what
  `backend/internal/pkg/pagination` parses.
- **Snake case** — all request/response fields are snake_case, typed 1:1
  in `src/services/*.ts` from the Go DTOs.

## Architecture

```
src/
  services/        # API service layer (one file per backend module)
  lib/             # types (Envelope/PageRequest), axios client, formatters
  store/           # Redux: auth session + UI state (sidebar etc.)
  hooks/           # useListQuery (cursor paging), useAuth, usePermissions
  components/
    ui/            # design-system primitives (Button, Modal, Toast, …)
    data-table/    # generic cursor-paginated DataTable + toolbar + pagination
    charts/        # Recharts wrappers (revenue, donut, top services)
  layouts/         # app shell, sidebar, header, route guards
  features/        # feature modules: <feature>/<page>.tsx (+ forms)
  router/          # lazy-loaded routes with guards
```

### Conventions

- Max ~300 lines per file; shared logic lives in hooks and components.
- UI components contain no business logic; data fetching lives in feature
  hooks/pages via TanStack Query.
- Every list page reuses `<DataTable>`: server-side search (debounced),
  column sorting, enum filters, date-range filters, skeleton loaders,
  friendly empty states, error states with retry, and summary chips built
  from the backend `summary` payload.
- Mutations invalidate their query keys and toast success/failure — no
  manual refresh needed.

## Auth & RBAC

- Token is stored in `localStorage`, synced to the Redux `auth` slice, and
  attached by the axios request interceptor.
- A 401 anywhere triggers a single global logout + redirect to `/login`.
- `usePermissions()` gates navigation and row actions. The backend is the
  real enforcement point; the UI hides actions the user's roles can't
  perform (`is_system` roles like `super_admin` see everything).

## Production build

The built `dist/` is meant to be served behind the Caddy reverse proxy
(next to the backend) so `/api/v1` stays same-origin. No build-time env is
required beyond `VITE_API_BASE_URL` (default `/api/v1`).
