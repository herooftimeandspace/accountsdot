export const APP_ROUTES = [
  { path: "/login", kind: "login", public: true },
  { path: "/dashboard", kind: "dashboard-redirect" },
  { path: "/dashboard/it-admin", kind: "static", artboardKey: "dashboard-it-admin" },
  { path: "/dashboard/hr-lifecycle", kind: "static", artboardKey: "dashboard-hr-lifecycle" },
  { path: "/dashboard/site-admin", kind: "static", artboardKey: "dashboard-site-admin" },
  { path: "/search", kind: "global-search" },
  { path: "/onboarding", kind: "onboarding", artboardKey: "onboarding" },
  { path: "/offboarding", kind: "offboarding", artboardKey: "offboarding" },
  { path: "/departing-seniors", kind: "departing-seniors", artboardKey: "offboarding" },
  { path: "/room-moves", kind: "room-moves", artboardKey: "room-moves" },
  { path: "/room-moves/bulk-draft", kind: "room-moves-bulk-draft", artboardKey: "room-moves-bulk-draft" },
  {
    path: "/phone-directory/by-person",
    kind: "phone-directory",
    mode: "person",
    artboardKey: "phone-directory-by-person",
  },
  {
    path: "/phone-directory/by-department",
    kind: "phone-directory",
    mode: "department",
    artboardKey: "phone-directory-by-department",
  },
  {
    path: "/phone-directory/by-room",
    kind: "phone-directory",
    mode: "room",
    artboardKey: "phone-directory-by-room",
  },
  { path: "/data-quality", kind: "data-quality", artboardKey: "data-quality" },
  { path: "/frequent-fliers", kind: "frequent-fliers", artboardKey: "frequent-fliers" },
  { path: "/meraki-last-seen", kind: "meraki-last-seen", artboardKey: "reports" },
  { path: "/student-data-cleanup", kind: "student-data-cleanup", artboardKey: "student-data-cleanup" },
  { path: "/reports", kind: "reports", artboardKey: "reports" },
  { path: "/reports/security-issues", kind: "security-issues-report", artboardKey: "reports" },
  { path: "/reports/zoom-desk-phone-renames", kind: "zoom-desk-phone-renames-report", artboardKey: "reports" },
  {
    path: "/reports/sync-transparency",
    kind: "static",
    artboardKey: "reports-sync-transparency",
  },
  { path: "/admin", kind: "static", artboardKey: "admin" },
  { path: "/admin/auth-settings", kind: "auth-settings", artboardKey: "admin-feature-flags" },
  { path: "/admin/feature-flags", kind: "feature-flags", artboardKey: "admin-feature-flags" },
  { path: "/my-profile", kind: "static", artboardKey: "my-profile" },
];

const ROUTE_ALIASES = new Map([["/invalid-student-names", "/student-data-cleanup"]]);

export const NAV_GROUP_ORDER = [
  "dashboard",
  "onboarding",
  "offboarding",
  "departingSeniors",
  "roomMoves",
  "phoneDirectory",
  "dataQuality",
  "frequentFliers",
  "merakiLastSeen",
  "studentDataCleanup",
  "reports",
  "admin",
];

const NAV_CHILD_ROUTE_GROUPS = {
  reports: [
    { path: "/reports/security-issues", label: "Security Issues" },
    { path: "/reports/zoom-desk-phone-renames", label: "Zoom Desk Phone Renames" },
    { path: "/reports/sync-transparency", label: "Sync Transparency" },
  ],
  admin: [
    { path: "/admin/auth-settings", label: "Auth Settings" },
    { path: "/admin/feature-flags", label: "Feature Flags" },
  ],
};

export function normalizePath(pathname) {
  if (!pathname || pathname === "/") {
    return "/";
  }
  const normalized = pathname.endsWith("/") ? pathname.slice(0, -1) : pathname;
  return ROUTE_ALIASES.get(normalized) ?? normalized;
}

export function resolveRoute(pathname) {
  const normalized = normalizePath(pathname);
  const errorMatch = normalized.match(/^\/error\/(\d{3})$/);
  if (errorMatch) {
    return {
      path: normalized,
      kind: "error",
      public: true,
      code: Number.parseInt(errorMatch[1], 10),
    };
  }

  return APP_ROUTES.find((route) => route.path === normalized) ?? null;
}

export function defaultPhoneDirectoryRoute(personaId) {
  return personaId === "site_secretary" ? "/phone-directory/by-room" : "/phone-directory/by-person";
}

export function navDestinationForKey(navKey, session) {
  const personaId = session?.current_persona?.id || "it_admin";
  const allowedRoutes = session?.allowed_routes ?? [];
  switch (navKey) {
    case "dashboard":
      return session?.landing_path?.startsWith("/dashboard/") ? session.landing_path : null;
    case "onboarding":
      return "/onboarding";
    case "offboarding":
      return "/offboarding";
    case "departingSeniors":
      return "/departing-seniors";
    case "roomMoves":
      return "/room-moves";
    case "phoneDirectory":
      return defaultPhoneDirectoryRoute(personaId);
    case "dataQuality":
      return "/data-quality";
    case "frequentFliers":
      return "/frequent-fliers";
    case "merakiLastSeen":
      return "/meraki-last-seen";
    case "studentDataCleanup":
      return "/student-data-cleanup";
    case "reports":
      return "/reports";
    case "admin":
      return allowedRoutes.includes("/admin") ? "/admin" : "/admin/auth-settings";
    default:
      return null;
  }
}

export function isRouteAllowed(session, path) {
  const allowedRoutes = session?.allowed_routes ?? [];
  return allowedRoutes.includes(path);
}

/**
 * artboardKeysForAllowedRoutes converts the backend-issued session.allowed_routes
 * list into generated artboard keys that may be warmed in the current browser
 * session. App calls this before prefetching, so direct-route authorization stays
 * tied to the same route registry used for rendering and unauthorized personas
 * never warm artboards for hidden sidebar destinations.
 */
export function artboardKeysForAllowedRoutes(session) {
  const allowedRoutes = session?.allowed_routes ?? [];
  const keys = new Set();
  allowedRoutes.forEach((path) => {
    const route = resolveRoute(path);
    if (route?.artboardKey) {
      keys.add(route.artboardKey);
    }
  });
  return [...keys];
}

export function navGroupVisible(navKey, session) {
  const allowedRoutes = session?.allowed_routes ?? [];
  switch (navKey) {
    case "dashboard":
      return allowedRoutes.some((route) => route.startsWith("/dashboard/"));
    case "phoneDirectory":
      return allowedRoutes.some((route) => route.startsWith("/phone-directory/"));
    case "reports":
      return allowedRoutes.some((route) => route === "/reports" || route.startsWith("/reports/"));
    case "dataQuality":
      return allowedRoutes.includes("/data-quality");
    case "onboarding":
      return allowedRoutes.includes("/onboarding");
    case "offboarding":
      return allowedRoutes.includes("/offboarding");
    case "departingSeniors":
      return allowedRoutes.includes("/departing-seniors");
    case "roomMoves":
      return allowedRoutes.includes("/room-moves") || allowedRoutes.includes("/room-moves/bulk-draft");
    case "frequentFliers":
      return allowedRoutes.includes("/frequent-fliers");
    case "merakiLastSeen":
      return allowedRoutes.includes("/meraki-last-seen");
    case "studentDataCleanup":
      return allowedRoutes.includes("/student-data-cleanup");
    case "admin":
      return allowedRoutes.includes("/admin") || allowedRoutes.includes("/admin/auth-settings") || allowedRoutes.includes("/admin/feature-flags");
    default:
      return false;
  }
}

export function buildVisibleNavGroups(session) {
  return NAV_GROUP_ORDER.filter((navKey) => navGroupVisible(navKey, session));
}

/**
 * visibleNavChildrenForKey returns documented child route buttons for a visible
 * sidebar parent. Some route-backed modes, such as Phone Directory's person,
 * room, and department routes, intentionally stay in-page controls rather than
 * sidebar children. The shared shell calls this with the same
 * session.allowed_routes list used for direct-route authorization, so
 * role-filtered nested buttons cannot advertise routes that would resolve to
 * the app-level 403 page.
 */
export function visibleNavChildrenForKey(navKey, session) {
  const allowedRoutes = new Set(session?.allowed_routes ?? []);
  return (NAV_CHILD_ROUTE_GROUPS[navKey] ?? []).filter((child) => allowedRoutes.has(child.path));
}
