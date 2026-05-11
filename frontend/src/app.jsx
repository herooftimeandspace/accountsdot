import { startTransition, useCallback, useEffect, useMemo, useState } from "react";
import { DevPersonaSwitcher } from "./components/DevPersonaSwitcher";
import { ErrorPage } from "./pages/ErrorPage";
import { LoginPage } from "./pages/LoginPage";
import { DataQualityPage } from "./pages/DataQualityPage";
import { DepartingSeniorsPage } from "./pages/DepartingSeniorsPage";
import { FrequentFliersPage } from "./pages/FrequentFliersPage";
import { OffboardingPage } from "./pages/OffboardingPage";
import { OnboardingPage } from "./pages/OnboardingPage";
import { PhoneDirectoryPage } from "./pages/PhoneDirectoryPage";
import { ReportsPage } from "./pages/ReportsPage";
import { SearchPage } from "./pages/SearchPage";
import { StaticPenPage } from "./pages/StaticPenPage";
import { StudentDataCleanupPage } from "./pages/StudentDataCleanupPage";
import { isRouteAllowed, normalizePath, resolveRoute } from "./lib/routeRegistry";

const DEV_API_BASE = "/api/v1/dev";
const DEFAULT_PERSONA_ID = "it_admin";
const PERSONA_STORAGE_KEY = "wizard-dev-persona";
const APP_TITLE = "The WIZARD";
const PERSONA_SWITCH_OVERLAY_MIN_MS = 450;
const STATIC_ROUTE_TITLES = {
  "dashboard-it-admin": "IT Admin Dashboard",
  "dashboard-hr-lifecycle": "Human Resources Dashboard",
  "dashboard-site-admin": "Site Admin Dashboard",
  onboarding: "Onboarding",
  offboarding: "Offboarding",
  "room-moves": "Room Moves",
  "frequent-fliers": "Frequent Fliers",
  "student-data-cleanup": "Student Data Cleanup",
  reports: "Reports",
  "reports-sync-transparency": "Sync Transparency",
  "reports-ticketing-human-work": "Ticketing Human Work",
  admin: "Admin",
  "my-profile": "My Profile",
};
const PHONE_DIRECTORY_TITLES = {
  person: "Phone Directory by Person",
  room: "Phone Directory by Room",
  department: "Phone Directory by Department",
};

function pageTitleForRoute(route, currentPath) {
  if (currentPath === "/") {
    return "Routing";
  }
  if (!route) {
    return "Page Not Found";
  }
  switch (route.kind) {
    case "login":
      return "Login";
    case "dashboard-redirect":
      return "Dashboard";
    case "data-quality":
      return "Data Quality";
    case "global-search":
      return "Global Search";
    case "offboarding":
      return "Offboarding";
    case "departing-seniors":
      return "Departing Seniors";
    case "frequent-fliers":
      return "Frequent Fliers";
    case "student-data-cleanup":
      return "Student Data Cleanup";
    case "reports":
      return "Reports";
    case "phone-directory":
      return PHONE_DIRECTORY_TITLES[route.mode] || "Phone Directory";
    case "static":
      return STATIC_ROUTE_TITLES[route.artboardKey] || "Page";
    case "error":
      return `Error ${route.code}`;
    default:
      return "Page";
  }
}

function readStoredPersona() {
  if (typeof window === "undefined") {
    return DEFAULT_PERSONA_ID;
  }
  try {
    return window.localStorage.getItem(PERSONA_STORAGE_KEY) || DEFAULT_PERSONA_ID;
  } catch {
    return DEFAULT_PERSONA_ID;
  }
}

function storePersona(personaId) {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.setItem(PERSONA_STORAGE_KEY, personaId);
  } catch {
    // Ignore local storage failures in DEV.
  }
}

function readLocationState() {
  return {
    pathname: normalizePath(window.location.pathname),
    search: window.location.search || "",
  };
}

function PageStatus({ title, message }) {
  return (
    // WCAG 4.1.3: app-level loading and redirect state changes are exposed as polite status updates.
    <main id="main-content" className="page-status" aria-live="polite">
      <section className="page-status__card">
        <h1>{title}</h1>
        <p>{message}</p>
      </section>
    </main>
  );
}

function PersonaSwitchOverlay({ label }) {
  if (!label) {
    return null;
  }

  return (
    <div className="persona-switch-overlay" role="status" aria-live="polite" aria-atomic="true">
      <strong>Switching to {label}...</strong>
    </div>
  );
}

async function readJSON(response) {
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    const error = new Error(payload?.message || `Request failed with ${response.status}`);
    error.status = response.status;
    error.payload = payload;
    throw error;
  }
  return payload;
}

function resolvePersonaSwitchTarget(payload, pathname) {
  const currentPath = normalizePath(pathname);
  const currentRoute = resolveRoute(currentPath);

  if (currentPath === "/dashboard" || currentPath.startsWith("/dashboard/") || currentPath === "/error/403") {
    return payload?.landing_path || "/dashboard";
  }

  if (!currentRoute || currentRoute.kind === "login" || currentRoute.kind === "error") {
    return null;
  }

  const allowedRoutes = payload?.allowed_routes ?? [];
  return allowedRoutes.includes(currentRoute.path) ? null : payload?.landing_path || "/dashboard";
}

export function App() {
  const [currentLocation, setCurrentLocation] = useState(readLocationState);
  const [session, setSession] = useState(null);
  const [sessionState, setSessionState] = useState("loading");
  const [sessionError, setSessionError] = useState(null);
  const [preferredPersonaId, setPreferredPersonaId] = useState(readStoredPersona);
  const [personaSwitchState, setPersonaSwitchState] = useState(null);

  const currentPath = currentLocation.pathname;
  const currentSearch = currentLocation.search;

  const navigate = useCallback((path, options = {}) => {
    const url = new URL(path, window.location.origin);
    const targetPathname = normalizePath(url.pathname);
    const targetSearch = url.search || "";
    const replace = options.replace ?? false;
    const targetHref = `${targetPathname}${targetSearch}`;
    const currentHref = `${normalizePath(window.location.pathname)}${window.location.search || ""}`;

    startTransition(() => {
      if (replace) {
        window.history.replaceState({}, "", targetHref);
      } else if (currentHref !== targetHref) {
        window.history.pushState({}, "", targetHref);
      }
      setCurrentLocation({ pathname: targetPathname, search: targetSearch });
    });
  }, []);

  const loadSession = useCallback(async () => {
    setSessionState("loading");
    setSessionError(null);
    try {
      const payload = await readJSON(
        await fetch(`${DEV_API_BASE}/session`, {
          credentials: "same-origin",
          headers: { Accept: "application/json" },
        })
      );
      setSession(payload);
      if (payload?.current_persona?.id) {
        setPreferredPersonaId(payload.current_persona.id);
        storePersona(payload.current_persona.id);
      }
      setSessionState("ready");
    } catch (error) {
      setSession(null);
      setSessionError(error);
      setSessionState("error");
    }
  }, []);

  useEffect(() => {
    void loadSession();
  }, [loadSession]);

  useEffect(() => {
    const handlePopState = () => {
      setCurrentLocation(readLocationState());
    };
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  const loginAsPersona = useCallback(
    async (personaId) => {
      const isPersonaSwitch = Boolean(session?.authenticated && session?.authorized);
      const targetPersona = session?.personas?.find((persona) => persona.id === personaId);
      if (isPersonaSwitch) {
        setPersonaSwitchState({
          targetPersonaId: personaId,
          targetLabel: targetPersona?.label || personaId,
          targetPath: null,
        });
      } else {
        setSessionState("updating");
      }
      setSessionError(null);
      try {
        const payload = await readJSON(
          await fetch(`${DEV_API_BASE}/login`, {
            method: "POST",
            credentials: "same-origin",
            headers: {
              "Content-Type": "application/json",
              Accept: "application/json",
            },
            body: JSON.stringify({ persona_id: personaId }),
          })
        );
        const switchTarget = resolvePersonaSwitchTarget(payload, window.location.pathname);
        const targetUrl = switchTarget ? new URL(switchTarget, window.location.origin) : null;
        const targetPathname = targetUrl ? normalizePath(targetUrl.pathname) : null;
        const targetSearch = targetUrl?.search || "";
        const targetHref = targetPathname ? `${targetPathname}${targetSearch}` : null;

        if (isPersonaSwitch) {
          setPersonaSwitchState((current) =>
            current?.targetPersonaId === personaId
              ? { ...current, targetPath: targetHref }
              : current
          );
        }

        startTransition(() => {
          if (targetHref) {
            window.history.replaceState({}, "", targetHref);
            setCurrentLocation({ pathname: targetPathname, search: targetSearch });
          }
          setSession(payload);
          setPreferredPersonaId(personaId);
          storePersona(personaId);
          setSessionState("ready");
        });

        if (isPersonaSwitch) {
          window.setTimeout(() => {
            window.requestAnimationFrame(() => {
              window.requestAnimationFrame(() => {
                setPersonaSwitchState((current) =>
                  current?.targetPersonaId === personaId ? null : current
                );
              });
            });
          }, PERSONA_SWITCH_OVERLAY_MIN_MS);
        }
      } catch (error) {
        setPersonaSwitchState(null);
        setSessionError(error);
        setSessionState("error");
      }
    },
    [session]
  );

  const logout = useCallback(async () => {
    setSessionState("updating");
    setSessionError(null);
    try {
      const payload = await readJSON(
        await fetch(`${DEV_API_BASE}/logout`, {
          method: "POST",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
        })
      );
      setSession(payload);
      setSessionState("ready");
      navigate("/login", { replace: true });
    } catch (error) {
      setSessionError(error);
      setSessionState("error");
    }
  }, [navigate]);

  const currentRoute = useMemo(() => resolveRoute(currentPath), [currentPath]);
  const currentSearchQuery = useMemo(() => {
    const params = new URLSearchParams(currentSearch);
    return params.get("q") ?? "";
  }, [currentSearch]);
  const authenticated = Boolean(session?.authenticated && session?.authorized);

  useEffect(() => {
    const routeTitle = pageTitleForRoute(currentRoute, currentPath);
    // WCAG 2.4.2: route changes update the document title for screen reader and tab users.
    document.title = `${routeTitle} | ${APP_TITLE}`;
  }, [currentPath, currentRoute]);

  const handleSharedSearch = useCallback(
    (query) => {
      const trimmed = query.trim();
      if (!trimmed) {
        if (currentRoute?.kind === "global-search") {
          navigate("/dashboard");
          return;
        }
        const params = new URLSearchParams(currentSearch);
        params.delete("q");
        const nextSearch = params.toString();
        navigate(`${currentPath}${nextSearch ? `?${nextSearch}` : ""}`);
        return;
      }
      navigate(`/search?q=${encodeURIComponent(trimmed)}`);
    },
    [currentPath, currentRoute?.kind, currentSearch, navigate]
  );

  const redirectTarget = useMemo(() => {
    if (sessionState !== "ready") {
      return null;
    }

    if (currentPath === "/") {
      return authenticated ? "/dashboard" : "/login";
    }

    if (currentRoute?.kind === "login") {
      return authenticated ? "/dashboard" : null;
    }

    if (!authenticated) {
      return currentPath === "/error/401" ? null : "/error/401";
    }

    if (!currentRoute) {
      return "/error/404";
    }

    if (currentRoute.kind === "dashboard-redirect") {
      return session?.landing_path || "/error/403";
    }

    if (currentRoute.kind === "error") {
      return null;
    }

    if (!isRouteAllowed(session, currentRoute.path)) {
      return "/error/403";
    }

    return null;
  }, [authenticated, currentPath, currentRoute, session, sessionState]);

  useEffect(() => {
    if (!redirectTarget || redirectTarget === currentPath) {
      return;
    }
    navigate(redirectTarget, { replace: true });
  }, [currentPath, navigate, redirectTarget]);

  const handleUnauthorized = useCallback(() => {
    navigate("/error/401", { replace: true });
  }, [navigate]);

  const handleForbidden = useCallback(() => {
    navigate("/error/403", { replace: true });
  }, [navigate]);

  const showDevPersonaSwitcher = Boolean(import.meta.env.DEV && authenticated);
  const activePersonaId = session?.current_persona?.id || preferredPersonaId;
  const visiblePersonaId = personaSwitchState?.targetPersonaId || activePersonaId;

  let page = null;
  if (sessionState === "loading") {
    page = <PageStatus title="Loading" message="Preparing the DEV session." />;
  } else if (sessionState === "error") {
    page = (
      <ErrorPage
        code={500}
        session={session}
        onNavigate={navigate}
        onSearch={handleSharedSearch}
        searchQuery={currentSearchQuery}
        details={sessionError?.message || "The DEV session could not be loaded."}
      />
    );
  } else if (redirectTarget) {
    page = <PageStatus title="Redirecting" message="Routing to the correct page." />;
  } else if (currentRoute?.kind === "login") {
    page = (
      <LoginPage
        personaId={preferredPersonaId}
        onLogin={loginAsPersona}
      />
    );
  } else if (currentRoute?.kind === "error") {
    page = (
      <ErrorPage
        code={currentRoute.code}
        session={session}
        onNavigate={navigate}
        onSearch={handleSharedSearch}
        searchQuery={currentSearchQuery}
      />
    );
  } else if (currentRoute?.kind === "data-quality") {
    page = (
      <DataQualityPage
        session={session}
        onNavigate={navigate}
        onSearch={handleSharedSearch}
        searchQuery={currentSearchQuery}
        currentSearch={currentSearch}
        onUnauthorized={handleUnauthorized}
        onForbidden={handleForbidden}
      />
    );
  } else if (currentRoute?.kind === "global-search") {
    page = (
      <SearchPage
        session={session}
        onNavigate={navigate}
        onSearch={handleSharedSearch}
        searchQuery={currentSearchQuery}
        onUnauthorized={handleUnauthorized}
        onForbidden={handleForbidden}
      />
    );
  } else if (currentRoute?.kind === "phone-directory") {
    page = (
      <PhoneDirectoryPage
        session={session}
        mode={currentRoute.mode}
        artboardKey={currentRoute.artboardKey}
        onNavigate={navigate}
        onSearch={handleSharedSearch}
        searchQuery={currentSearchQuery}
        currentSearch={currentSearch}
        onUnauthorized={handleUnauthorized}
        onForbidden={handleForbidden}
      />
    );
  } else if (currentRoute?.kind === "onboarding") {
    page = (
      <OnboardingPage
        session={session}
        onNavigate={navigate}
        onSearch={handleSharedSearch}
        searchQuery={currentSearchQuery}
        onUnauthorized={handleUnauthorized}
        onForbidden={handleForbidden}
      />
    );
  } else if (currentRoute?.kind === "offboarding") {
    page = (
      <OffboardingPage
        session={session}
        onNavigate={navigate}
        onSearch={handleSharedSearch}
        searchQuery={currentSearchQuery}
        onUnauthorized={handleUnauthorized}
        onForbidden={handleForbidden}
      />
    );
  } else if (currentRoute?.kind === "departing-seniors") {
    page = (
      <DepartingSeniorsPage
        session={session}
        onNavigate={navigate}
        onSearch={handleSharedSearch}
        searchQuery={currentSearchQuery}
        onUnauthorized={handleUnauthorized}
        onForbidden={handleForbidden}
      />
    );
  } else if (currentRoute?.kind === "frequent-fliers") {
    page = (
      <FrequentFliersPage
        session={session}
        onNavigate={navigate}
        onSearch={handleSharedSearch}
        searchQuery={currentSearchQuery}
      />
    );
  } else if (currentRoute?.kind === "student-data-cleanup") {
    page = (
      <StudentDataCleanupPage
        session={session}
        onNavigate={navigate}
        onSearch={handleSharedSearch}
        searchQuery={currentSearchQuery}
      />
    );
  } else if (currentRoute?.kind === "reports") {
    page = (
      <ReportsPage
        session={session}
        onNavigate={navigate}
        onSearch={handleSharedSearch}
        searchQuery={currentSearchQuery}
      />
    );
  } else if (currentRoute?.kind === "static") {
    page = (
      <StaticPenPage
        artboardKey={currentRoute.artboardKey}
        session={session}
        onNavigate={navigate}
        onSearch={handleSharedSearch}
        searchQuery={currentSearchQuery}
      />
    );
  } else {
    page = <PageStatus title="Loading" message="Preparing the requested page." />;
  }

  return (
    <>
      {/* WCAG 2.4.1/2.4.7: keyboard users can bypass shared chrome and see the focused link. */}
      <a className="skip-link" href="#main-content">
        Skip to main content
      </a>
      {showDevPersonaSwitcher ? (
        <DevPersonaSwitcher
          session={session}
          personaId={visiblePersonaId}
          pendingPersonaId={personaSwitchState?.targetPersonaId}
          pendingLabel={personaSwitchState?.targetLabel}
          sessionState={sessionState}
          onChange={(personaId) => {
            if (personaId === visiblePersonaId) {
              return;
            }
            void loginAsPersona(personaId);
          }}
        />
      ) : null}
      {personaSwitchState ? <PersonaSwitchOverlay label={personaSwitchState.targetLabel} /> : null}
      {page}
    </>
  );
}
