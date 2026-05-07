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
