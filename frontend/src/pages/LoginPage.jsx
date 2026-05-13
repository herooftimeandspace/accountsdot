import { PenArtboard } from "../lib/PenArtboard";
import { useGeneratedArtboard } from "../lib/generatedArtboards";

const LOGIN_ARTBOARD_KEY = "login";
const LOGIN_BUTTON_NODE_ID = "f5";

/**
 * LoginPage renders the UI surface for frontend/src/pages/LoginPage.jsx. The React router renders this page/helper after route resolution in frontend/src/app.jsx; debug it by following props, fetch calls, overlay state, and matching /api/v1/dev backend handlers. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller. Pay special attention to side effects: this path may update React state, browser storage, cookies, or DEV mock APIs and should stay aligned with docs/external-write-inventory.md when it triggers mutations.
 */
export function LoginPage({ personaId, onLogin }) {
  const { artboard, status } = useGeneratedArtboard(LOGIN_ARTBOARD_KEY);

  const hotspots = {
    [LOGIN_BUTTON_NODE_ID]: {
      label: "Log in with Google",
      onClick: () => onLogin?.(personaId),
    },
  };

  if (status === "loading") {
    return (
      <main id="main-content" className="page-status" aria-live="polite">
        <section className="page-status__card">
          <h1>Loading Login</h1>
          <p>Preparing the DEV login page.</p>
        </section>
      </main>
    );
  }

  if (!artboard) {
    return <main id="main-content" className="page-status"><h1>Login unavailable</h1></main>;
  }

  return (
    <main id="main-content" className="page-canvas page-canvas--login" aria-labelledby="login-page-title">
      {/* WCAG 2.4.2/2.4.6: visual PEN login art is aria-hidden, so the page keeps a semantic heading. */}
      <h1 id="login-page-title" className="sr-only">
        The WIZARD Login
      </h1>
      <div className="page-canvas__frame">
        <PenArtboard artboard={artboard} hotspots={hotspots} />
      </div>
    </main>
  );
}
