import type { AnalysisMode, ApiError, VerifyResponse } from '../types';

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

// userFacingMessage turns HTTP failures into clear, non-alarming copy. The
// user's saved research lives in localStorage and is never touched by an API
// failure, so the message says so explicitly instead of implying data loss.
function userFacingMessage(status: number, bodyError?: string): string {
  if (bodyError && status < 500) {
    return bodyError;
  }
  if (status === 429) {
    return 'Too many requests. Please wait a moment and try again.';
  }
  if (status === 502 || status === 504 || status >= 500) {
    return 'Verification could not be completed. The verification service returned an error. Your saved research has not been deleted. Try again.';
  }
  return bodyError ?? `Verification failed (HTTP ${status}).`;
}

// parseApiBody safely reads a JSON body. If the response is not JSON at all
// (e.g. the platform returned HTML for a missing route), null is returned so
// the caller can fall back to a friendly message instead of crashing on a
// syntax error.
async function parseApiBody(res: Response): Promise<ApiError | VerifyResponse | null> {
  try {
    return (await res.json()) as ApiError | VerifyResponse;
  } catch {
    return null;
  }
}

// verify submits a single input to the TrustCheck API and returns the typed
// verification result. Network failures are surfaced as a clear, actionable
// message instead of a raw fetch error.
export async function verify(input: string, mode: AnalysisMode = 'quick'): Promise<VerifyResponse> {
  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}/verify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ input, mode }),
    });
  } catch {
    throw new Error('Could not reach the TrustCheck API. Is it running?');
  }

  if (!res.ok) {
    const body = await parseApiBody(res);
    const bodyError = body && typeof body === 'object' && 'error' in body ? body.error : undefined;
    throw new ApiRequestError(userFacingMessage(res.status, bodyError), res.status);
  }

  const body = await parseApiBody(res);
  if (body === null) {
    throw new ApiRequestError(
      'Verification could not be completed. The service returned an unreadable response. Your saved research has not been deleted. Try again.',
      res.status,
    );
  }
  return body as VerifyResponse;
}
