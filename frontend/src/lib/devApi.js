export class DevApiError extends Error {
  constructor(message, status, payload) {
    super(message);
    this.name = "DevApiError";
    this.status = status;
    this.payload = payload;
  }
}

/**
 * readDevApiJSON is the shared decoder for DEV-only frontend JSON endpoints.
 * Read-only page query functions use it so HTTP status, structured error
 * payloads, and auth failures are handled consistently across routes.
 */
export async function readDevApiJSON(response) {
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new DevApiError(
      payload?.message || `Request failed with ${response.status}`,
      response.status,
      payload
    );
  }
  return payload;
}

/**
 * fetchDevApiJSON wraps same-origin DEV API fetches for React Query callers.
 * It preserves the query-provided AbortSignal so route changes, persona
 * switches, and refreshes can cancel stale requests before they update UI.
 */
export async function fetchDevApiJSON(input, { signal, ...init } = {}) {
  const response = await fetch(input, {
    credentials: "same-origin",
    headers: { Accept: "application/json", ...(init.headers || {}) },
    ...init,
    signal,
  });
  return readDevApiJSON(response);
}

/**
 * handleDevApiAuthError forwards DEV API 401/403 failures to app-level route
 * guards. Returning true lets page components keep ordinary network failures
 * on their local error surfaces without duplicating status checks.
 */
export function handleDevApiAuthError(error, { onUnauthorized, onForbidden } = {}) {
  if (error?.status === 401) {
    onUnauthorized?.();
    return true;
  }
  if (error?.status === 403) {
    onForbidden?.();
    return true;
  }
  return false;
}
