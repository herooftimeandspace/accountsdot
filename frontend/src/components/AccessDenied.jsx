/**
 * AccessDenied renders the UI surface for frontend/src/components/AccessDenied.jsx. Page components call this shared component/helper to keep repeated runtime UI behavior consistent; debug it through props, callbacks, and rendered DOM state. Inputs are the parameters or props in the signature; output is the returned value, rendered JSX, or state transition consumed by the caller.
 */
export function AccessDenied({ title = "Access denied", message, role = "alert" }) {
  const liveRegion = role === "status" ? "polite" : "assertive";

  return (
    // WCAG 4.1.3: async loading, denial, and error states are exposed as live status messages.
    <div className="overlay-card" role={role} aria-live={liveRegion}>
      <h2>{title}</h2>
      <p>{message}</p>
    </div>
  );
}
