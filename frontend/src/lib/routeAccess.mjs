/**
 * redirectTargetForRoute centralizes the app-level route guard used by
 * frontend/src/app.jsx. Public login and explicit error routes remain directly
 * reachable while protected implemented routes require an authenticated,
 * authorized DEV session.
 */
export function redirectTargetForRoute({ sessionState, authenticated, currentPath, currentRoute, session }) {
  if (sessionState !== "ready") {
    return null;
  }

  if (currentPath === "/") {
    return authenticated ? "/dashboard" : "/login";
  }

  if (currentRoute?.kind === "login") {
    return authenticated ? "/dashboard" : null;
  }

  if (currentRoute?.kind === "error") {
    return null;
  }

  if (!authenticated) {
    return "/error/401";
  }

  if (!currentRoute) {
    return "/error/404";
  }

  if (currentRoute.kind === "dashboard-redirect") {
    return session?.landing_path || "/error/403";
  }

  if (!session?.allowed_routes?.includes(currentRoute.path)) {
    return "/error/403";
  }

  return null;
}
