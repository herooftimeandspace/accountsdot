# Shared Shell Help Content Walkthrough

Shared-shell help content is the route-aware help drawer rendered on implemented pages. It is high risk because it appears across staff-only pages, depends on persona-visible navigation, and must stay product-aligned without adding undocumented controls or mock-policy text.

## Frontend Entrypoint

- Route registry: `frontend/src/lib/routeRegistry.js`
- Route help content: `frontend/src/lib/routeHelpContent.js`
- App dispatch: `frontend/src/app.jsx`
- Shared helper: `frontend/src/lib/sharedShellPresentation.jsx`
- Page-level callers: implemented pages under `frontend/src/pages/`

The app resolves the URL to a route, checks whether it is public or allowed for the current session, then renders a page component. Implemented pages call `createSharedShellRenderOverlay` and pass it to `PenArtboard` or the generated view. The helper overlays runtime shell behavior over `.pen`-derived artboards.

Shared help resolves in this order:

- A page may still pass explicit `helpContent` to `createSharedShellRenderOverlay` when it has a documented special case.
- Otherwise `createSharedShellRenderOverlay` calls `helpContentForRoute(activeRoutePath, activeNavKey)` from `frontend/src/lib/routeHelpContent.js`.
- `activeRoutePath` wins over `activeNavKey` so child routes such as `/reports/sync-transparency`, `/reports/security-issues`, and `/admin/feature-flags` do not inherit generic parent copy.
- The nav-key fallback is only a safety net for legacy callers that have not yet supplied a route path. New implemented pages should pass `activeRoutePath`.

`routeHelpContent.js` also stores a short source note per route. The notes identify the PRD and implementation-plan areas used to author the drawer copy. Keep these notes current when page behavior changes so reviewers can trace help text back to the source-of-truth documents without reading every section again.

## Handler, Store, And Helper Chain

There is no backend handler for help drawer content. The route/page data chain still matters because the help overlay only renders for authenticated and authorized sessions:

- `frontend/src/app.jsx` calls `/api/v1/dev/session` through `loadSession` and stores the session payload.
- `frontend/src/lib/routeRegistry.js` uses `allowed_routes` to decide route access and visible nav groups.
- `frontend/src/lib/sharedShellPresentation.jsx` `createSharedShellRenderOverlay` exits early when the session is not authenticated or authorized.
- `createSharedShellRenderOverlay` locates shell node bounds, resolves route-specific help content, and renders `SharedShellHelpOverlay`.
- `SharedShellHelpOverlay` renders the invisible hit target over the artboard help icon and opens `RuntimeDrawer`.

The DEV session and persona payload is served from `internal/web/dev_frontend.go`, especially `handleDevSession`, `handleDevLogin`, `buildDevSessionPayload`, `resolveAuthenticatedDevPersona`, and `routeAllowed`.

## Payload Shape

Help content is a frontend object, not an HTTP payload:

```js
{
  title: "Onboarding help",
  sections: [
    {
      heading: "What this page shows",
      paragraphs: [
        "This page shows upcoming staff onboarding work."
      ]
    }
  ]
}
```

`SharedShellHelpOverlay` also accepts legacy `section.body` and converts it to a single-paragraph section. The session payload that gates the overlay includes:

```json
{
  "authenticated": true,
  "authorized": true,
  "current_persona": { "id": "it_admin" },
  "allowed_routes": ["/onboarding", "/room-moves", "/data-quality"],
  "shell": {
    "scope_title": "All Sites",
    "search_placeholder": "Search"
  }
}
```

## Authorization And Persona Behavior

Help content follows page and shell authorization rather than having its own permission layer.

`createSharedShellRenderOverlay` returns `null` unless `session.authenticated` and `session.authorized` are true and at least one of `onNavigate` or `onSearch` is callable. Visible sidebar rows are computed by `buildVisibleNavGroups(session)`, which reads `allowed_routes`. The help icon overlay uses the current artboard's shared shell help icon bounds; if the node is not present or cannot be located, `SharedShellHelpOverlay` returns `null`.

Unauthorized users should never receive reduced help content on the same URL. App-level route checks should send them through login, unauthorized, or forbidden behavior before the implemented page renders.

## Mutation Boundary

There is no provider, database, or DEV mock store mutation for opening help. The only runtime state mutation is local React state:

- `SharedShellHelpOverlay` stores `isOpen`.
- Clicking the help hit target toggles `isOpen`.
- Closing `RuntimeDrawer` resets `isOpen` to false.

Because there is no external write, this path is not listed as a DEV mock mutation in `docs/external-write-inventory.md`. It still depends on the DEV session cookie path documented there because session login/logout and route visibility determine whether the shell overlay appears.

## Tests

Relevant coverage:

- `internal/web/dev_frontend_test.go` covers DEV session payloads, `allowed_routes`, persona access, and implemented-page API authorization.
- `frontend/src/lib/routeRegistry.test.mjs` covers route help content completeness for implemented logged-in routes, source-note coverage, child-route specificity, and required correction-path warnings.
- UI-heavy changes to this path should run the implemented-page checks documented in `README.md`: `npm run build:web` and `npm run a11y:check`; `.pen` layout changes should additionally run `npm run pen:check` and `npm run pen:lint`.

## Debugging Breakpoints

Frontend breakpoints:

- `frontend/src/app.jsx` `loadSession`, route resolution, and page dispatch.
- `frontend/src/lib/routeRegistry.js` `isRouteAllowed`, `navGroupVisible`, and `buildVisibleNavGroups`.
- `frontend/src/lib/sharedShellPresentation.jsx` `createSharedShellRenderOverlay`, `defaultHelpContent`, and `SharedShellHelpOverlay`.
- `frontend/src/lib/routeHelpContent.js` `helpContentForRoute`, route entries, and source notes.
- Page-specific `helpContent` constants only when a page intentionally overrides the route-level model.
- `frontend/src/components/RuntimeDrawer.jsx`, if the drawer does not close or focus/pointer behavior is wrong.

Backend breakpoints:

- `internal/web/dev_frontend.go` `handleDevSession`, `handleDevLogin`, `buildDevSessionPayload`, `resolveAuthenticatedDevPersona`, and `routeAllowed`.
- `internal/web/app.go` `NewAppHandler`, to confirm DEV session routes are registered.

Useful symptoms:

- Help button missing on an implemented page usually means the session is unauthenticated/unauthorized, `onNavigate` and `onSearch` are both missing, or the generated artboard does not expose the shared help icon node.
- Wrong help text usually means the page passed stale `helpContent`, omitted `activeRoutePath`, or has a stale entry in `routeHelpContent.js`.
- Sidebar/help mismatch usually means `allowed_routes`, `activeRoutePath`, or `activeNavKey` changed without corresponding route-registry or shared-shell updates.
