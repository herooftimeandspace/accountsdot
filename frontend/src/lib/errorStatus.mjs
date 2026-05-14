/**
 * errorStatusCodeFor keeps the app-level error surface aligned with failed DEV
 * API requests. The React app calls this before rendering ErrorPage for session,
 * login, and logout failures; it accepts the thrown fetch error plus the status
 * to use when no valid HTTP status was attached. The returned integer drives
 * the visible error headline, so a 404 request failure cannot be mislabeled as
 * a 500 application error.
 */
export function errorStatusCodeFor(error, fallbackCode = 500) {
  const status = Number(error?.status);
  if (Number.isInteger(status) && status >= 100 && status <= 599) {
    return status;
  }

  return fallbackCode;
}
