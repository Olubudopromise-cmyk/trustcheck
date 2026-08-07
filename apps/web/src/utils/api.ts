import type { ApiError, VerifyResponse } from '../types';

// API_BASE_URL resolves where the browser reaches the TrustCheck API.
// Production (Netlify) serves the API as a Function mounted at /api on the
// same origin as the frontend, so a relative path is used and no absolute URL
// leaks into the client bundle. Local development defaults to the Go server
// on :8080 and can be overridden with NEXT_PUBLIC_API_URL (e.g. when the API
// is hosted elsewhere).
export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ??
  (process.env.NODE_ENV === 'production' ? '/api' : 'http://localhost:8080');

// ApiRequestError is thrown when the API responds with a non-2xx status.
// status carries the HTTP status code so callers can react to specific errors
// (e.g. 429 for rate limiting).
export class ApiRequestError extends Error {
  readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'ApiRequestError';
    this.status = status;
  }
}

// verify submits a single input to the TrustCheck API and returns the typed
// verification result. Network failures are surfaced as a clear, actionable
// message instead of a raw fetch error.
export async function verify(input: string): Promise<VerifyResponse> {
  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}/verify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ input }),
    });
  } catch {
    throw new Error('Could not reach the TrustCheck API. Is it running?');
  }

  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ApiError | null;
    throw new ApiRequestError(
      body?.error ?? `Verification failed (HTTP ${res.status}).`,
      res.status,
    );
  }

  return (await res.json()) as VerifyResponse;
}
