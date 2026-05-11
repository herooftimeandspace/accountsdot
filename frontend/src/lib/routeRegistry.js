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
  { path: "/room-moves", kind: "static", artboardKey: "room-moves" },
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
  { path: "/student-data-cleanup", kind: "student-data-cleanup", artboardKey: "student-data-cleanup" },
  { path: "/reports", kind: "static", artboardKey: "reports" },
  {
    path: "/reports/sync-transparency",
    kind: "static",
    artboardKey: "reports-sync-transparency",
  },
  {
    path: "/reports/ticketing-human-work",
    kind: "static",
    artboardKey: "reports-ticketing-human-work",
  },
  { path: "/admin", kind: "static", artboardKey: "admin" },
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
  "studentDataCleanup",
  "reports",
  "admin",
];

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
    case "studentDataCleanup":
      return "/student-data-cleanup";
    case "reports":
      return "/reports";
    case "admin":
      return "/admin";
    default:
      return null;
  }
}

export function isRouteAllowed(session, path) {
  const allowedRoutes = session?.allowed_routes ?? [];
  return allowedRoutes.includes(path);
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
      return allowedRoutes.includes("/room-moves");
    case "frequentFliers":
      return allowedRoutes.includes("/frequent-fliers");
    case "studentDataCleanup":
      return allowedRoutes.includes("/student-data-cleanup");
    case "admin":
      return allowedRoutes.includes("/admin");
    default:
      return false;
  }
}

export function buildVisibleNavGroups(session) {
  return NAV_GROUP_ORDER.filter((navKey) => navGroupVisible(navKey, session));
}
