import { PenArtboard } from "../lib/PenArtboard";
import { generatedArtboards } from "../generated/artboards.generated.js";

const LOGIN_ARTBOARD_KEY = "login";
const LOGIN_BUTTON_NODE_ID = "f5";

export function LoginPage({ personaId, onLogin }) {
  const artboard = generatedArtboards[LOGIN_ARTBOARD_KEY];

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
        <PenArtboard artboard={artboard} hotspots={hotspots} />
      </div>
    </main>
  );
}
