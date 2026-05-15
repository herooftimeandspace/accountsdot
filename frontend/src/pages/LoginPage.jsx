import { PenArtboard } from "../lib/PenArtboard";
import loginArtboard from "../generated/login.artboard.json";

const LOGIN_BUTTON_NODE_ID = "f5";

/**
 * LoginPage renders the public `/login` route after `App` resolves a logged-out route.
 * It receives the preferred DEV persona id plus the caller-owned login callback, renders
 * the static three-element PEN artboard without an intermediate loading card, and exposes
 * the visual Google button as one named native button. The page itself does not mutate
 * state; activating the hotspot delegates to `App.loginAsPersona`, which writes the
 * DEV mock session cookie through `POST /api/v1/dev/login`.
 */
export function LoginPage({ personaId, onLogin }) {
  const hotspots = {
    [LOGIN_BUTTON_NODE_ID]: {
      label: "Log in with Google",
      onClick: () => onLogin?.(personaId),
    },
  };

  return (
    <main id="main-content" className="page-canvas page-canvas--login" aria-labelledby="login-page-title">
      {/* WCAG 2.4.2/2.4.6: visual PEN login art is aria-hidden, so the page keeps a semantic heading. */}
      <h1 id="login-page-title" className="sr-only">
        The WIZARD Login
      </h1>
      <div className="page-canvas__frame">
        <PenArtboard artboard={loginArtboard} hotspots={hotspots} />
      </div>
    </main>
  );
}
