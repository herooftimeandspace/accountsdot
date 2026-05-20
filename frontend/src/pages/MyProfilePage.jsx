import { useEffect, useMemo, useState } from "react";

import { RuntimeDrawer } from "../components/RuntimeDrawer";
import { PenArtboard } from "../lib/PenArtboard";
import { useGeneratedArtboard } from "../lib/generatedArtboards";
import { buildArtboardSemanticSummary } from "../lib/artboardSemantics";
import {
  buildSharedShellHiddenNodeIds,
  buildSharedShellImageOverrides,
  buildSharedShellTextOverrides,
  createSharedShellRenderOverlay,
} from "../lib/sharedShellPresentation";

const MY_PROFILE_ENDPOINT = "/api/v1/dev/my-profile";
const ARTBOARD_KEY = "my-profile";
const PROFILE_ROUTE = "/my-profile";
const MY_PROFILE_STATIC_NODE_IDS = [
  "my-profile__f62",
  "my-profile__t63",
  "my-profile__l91",
  "my-profile__t92",
  "my-profile__f93",
  "my-profile__t94",
  "my-profile__t95",
  "my-profile__f96",
  "my-profile__t97",
  "my-profile__p100",
  "my-profile__p101",
  "my-profile__t103",
  "my-profile__f104",
  "my-profile__t105",
  "my-profile__t106",
  "my-profile__t107",
  "my-profile__t108",
  "my-profile__t109",
  "my-profile__t110",
  "my-profile__t111",
  "my-profile__t112",
  "my-profile__t113",
  "my-profile__t114",
  "my-profile__t115",
  "my-profile__t116",
  "my-profile__t117",
  "my-profile__t118",
  "my-profile__t119",
  "my-profile__f120",
  "my-profile__t121",
  "my-profile__t122",
];

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

function profileFormFromPayload(profile) {
  return {
    preferred_first_name: profile?.preferred_first_name ?? "",
    preferred_last_name: profile?.preferred_last_name ?? "",
    pronouns: profile?.pronouns ?? "",
  };
}

function buildProfileTextOverrides(profile) {
  if (!profile) {
    return {};
  }
  return {
    "my-profile__t60": "View your account information and manage your display name and pronouns.",
    "my-profile__t69": profile.legal_name ?? "",
    "my-profile__t72": profile.display_name ?? "",
    "my-profile__t75": profile.email ?? "",
    "my-profile__t78": profile.site ?? "",
    "my-profile__t81": profile.department ?? "",
    "my-profile__t84": profile.manager ?? "",
    "my-profile__t87": profile.room ?? "",
    "my-profile__t90": profile.phone_extension ?? "",
  };
}

/**
 * MyProfileEditDrawer renders the shared right-drawer direct-edit form for /my-profile.
 * MyProfilePage owns the payload and save callback; this component keeps form state local until
 * the operator clicks Save, then reports the server-returned mock profile back to the page.
 */
function MyProfileEditDrawer({ profile, onClose, onSave }) {
  const [form, setForm] = useState(() => profileFormFromPayload(profile));
  const [state, setState] = useState("idle");
  const [message, setMessage] = useState("");
  const [errors, setErrors] = useState({});
  const profileIdentity = `${profile?.email ?? ""}|${profile?.legal_name ?? ""}`;

  useEffect(() => {
    setForm(profileFormFromPayload(profile));
    setState("idle");
    setMessage("");
    setErrors({});
  }, [profileIdentity]);

  function updateField(field, value) {
    setForm((current) => ({ ...current, [field]: value }));
    setErrors((current) => ({ ...current, [field]: "" }));
  }

  async function submitProfile(event) {
    event.preventDefault();
    setState("saving");
    setMessage("");
    setErrors({});
    try {
      await onSave(form);
      setState("saved");
      setMessage("Profile saved.");
    } catch (error) {
      setState("error");
      setMessage(error.message);
      setErrors(error.payload?.errors || {});
    }
  }

  return (
    <RuntimeDrawer title="Edit My Profile" onClose={onClose} variant="modal">
      <form className="my-profile-runtime__drawer-form" onSubmit={submitProfile}>
        <p className="my-profile-runtime__drawer-note">
          Update the display name and pronouns used by your profile. Legal-name records stay unchanged.
        </p>
        <label>
          <span>Preferred first name</span>
          <input
            value={form.preferred_first_name}
            maxLength={50}
            required
            onChange={(event) => updateField("preferred_first_name", event.target.value)}
          />
          {errors.preferred_first_name ? <small role="alert">{errors.preferred_first_name}</small> : null}
        </label>
        <label>
          <span>Preferred last name</span>
          <input
            value={form.preferred_last_name}
            maxLength={50}
            required
            onChange={(event) => updateField("preferred_last_name", event.target.value)}
          />
          {errors.preferred_last_name ? <small role="alert">{errors.preferred_last_name}</small> : null}
        </label>
        <label>
          <span>Pronouns</span>
          <input
            value={form.pronouns}
            maxLength={40}
            onChange={(event) => updateField("pronouns", event.target.value)}
          />
          {errors.pronouns ? <small role="alert">{errors.pronouns}</small> : null}
        </label>
        {message ? (
          <p className="my-profile-runtime__drawer-message" role={state === "error" ? "alert" : "status"}>
            {message}
          </p>
        ) : null}
        <button type="submit" disabled={state === "saving"}>
          {state === "saving" ? "Saving" : "Save Profile"}
        </button>
      </form>
    </RuntimeDrawer>
  );
}

/**
 * MyProfilePage combines the generated My Profile artboard with runtime-owned direct editing.
 * Static profile geometry remains sourced from the `.pen` artboard, while React hides obsolete
 * request/status regions, fetches DEV mock profile data, opens the shared drawer, and writes only
 * to `/api/v1/dev/my-profile` for eligible employee and contractor personas.
 */
export function MyProfilePage({ session, onNavigate, onSearch, searchQuery = "", onUnauthorized, onForbidden }) {
  const { artboard, status: artboardStatus } = useGeneratedArtboard(ARTBOARD_KEY);
  const [payload, setPayload] = useState(null);
  const [pageState, setPageState] = useState("loading");
  const [message, setMessage] = useState("");
  const [drawerOpen, setDrawerOpen] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    async function loadProfile() {
      setPageState("loading");
      setMessage("");
      try {
        const nextPayload = await readJSON(
          await fetch(MY_PROFILE_ENDPOINT, {
            credentials: "same-origin",
            headers: { Accept: "application/json" },
            signal: controller.signal,
          })
        );
        setPayload(nextPayload);
        setPageState("ready");
      } catch (error) {
        if (controller.signal.aborted) {
          return;
        }
        if (error.status === 401) {
          onUnauthorized?.();
          return;
        }
        if (error.status === 403) {
          onForbidden?.();
          return;
        }
        setMessage(error.message);
        setPageState("error");
      }
    }
    void loadProfile();
    return () => controller.abort();
  }, [onForbidden, onUnauthorized, session?.current_persona?.id]);

  const profile = payload?.profile;
  const textOverrides = useMemo(
    () => ({
      ...buildSharedShellTextOverrides(session),
      ...buildProfileTextOverrides(profile),
    }),
    [profile, session]
  );
  const imageNodeOverrides = buildSharedShellImageOverrides(session);
  const hiddenNodeIds = useMemo(
    () => [
      ...buildSharedShellHiddenNodeIds(session, {
        hideNavHighlight: true,
        hideSearchPlaceholder: true,
        hideAllNavGroups: true,
      }),
      ...MY_PROFILE_STATIC_NODE_IDS,
    ],
    [session]
  );
  const sharedShellRenderOverlay = createSharedShellRenderOverlay({
    session,
    onNavigate,
    onSearch,
    searchQuery,
    activeRoutePath: PROFILE_ROUTE,
    refreshMetadata: null,
  });

  async function saveProfile(form) {
    const nextPayload = await readJSON(
      await fetch(MY_PROFILE_ENDPOINT, {
        method: "PUT",
        credentials: "same-origin",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: JSON.stringify(form),
      })
    );
    setPayload(nextPayload);
  }

  const renderOverlay = (overlayProps) => (
    <>
      {sharedShellRenderOverlay(overlayProps)}
      <button
        type="button"
        className="my-profile-runtime__edit-entry"
        aria-label="Edit My Profile"
        onClick={() => setDrawerOpen(true)}
        disabled={pageState !== "ready" || !profile?.editable}
      />
      <button
        type="button"
        className="my-profile-runtime__directory-link"
        onClick={() => onNavigate("/phone-directory/by-person")}
      >
        Go to Phone Directory
      </button>
      {pageState === "loading" ? (
        <p className="my-profile-runtime__status" role="status">
          Loading profile...
        </p>
      ) : null}
      {message ? (
        <p className="my-profile-runtime__status my-profile-runtime__status--error" role="alert">
          {message}
        </p>
      ) : null}
      {drawerOpen && profile ? (
        <MyProfileEditDrawer profile={profile} onClose={() => setDrawerOpen(false)} onSave={saveProfile} />
      ) : null}
    </>
  );

  if (artboardStatus === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading My Profile</h1>
          <p>Preparing the generated page artboard.</p>
        </section>
      </main>
    );
  }
  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>My Profile unavailable</h1></main>;
  }

  const semanticSummary = buildArtboardSemanticSummary(artboard, {
    fallbackTitle: "My Profile",
    textOverrides,
  });

  return (
    <main id="main-content" className="page-canvas page-canvas--static" aria-labelledby="my-profile-title">
      <section className="sr-only" aria-labelledby="my-profile-title">
        <h1 id="my-profile-title">{semanticSummary.title}</h1>
        <p>Use Edit My Profile to update display name and pronouns for the current session.</p>
        {semanticSummary.items.length > 0 ? (
          <ul>
            {semanticSummary.items.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        ) : null}
      </section>
      <div className="page-canvas__frame">
        <PenArtboard
          artboard={artboard}
          textOverrides={textOverrides}
          hiddenNodeIds={hiddenNodeIds}
          imageNodeOverrides={imageNodeOverrides}
          renderOverlay={renderOverlay}
        />
      </div>
    </main>
  );
}
