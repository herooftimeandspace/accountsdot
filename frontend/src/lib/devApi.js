export class DevApiError extends Error {
  constructor(message, status, payload) {
    super(message);
    this.name = "DevApiError";
    this.status = status;
    this.payload = payload;
  }
}

export async function readDevApiJSON(response) {
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new DevApiError(payload?.message || `Request failed with ${response.status}`, response.status, payload);
  }
  return payload;
}

export async function fetchDevApiJSON(input, { signal, ...init } = {}) {
  const response = await fetch(input, {
    credentials: "same-origin",
    headers: { Accept: "application/json", ...(init.headers || {}) },
    ...init,
    signal,
  });
  return readDevApiJSON(response);
}

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
