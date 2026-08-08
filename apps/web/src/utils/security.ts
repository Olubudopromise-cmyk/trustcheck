import type { SecurityResponse } from '../types';
import { API_BASE_URL, ApiRequestError } from './api';

// securityAnalyze submits source code to the TrustCheck security analysis endpoint.
export async function securityAnalyze(
  code: string,
  filename: string,
  language?: string,
): Promise<SecurityResponse> {
  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}/security`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code, filename, language }),
    });
  } catch {
    throw new Error('Could not reach the TrustCheck API. Is it running?');
  }

  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string } | null;
    throw new ApiRequestError(
      body?.error ?? `Security analysis failed (HTTP ${res.status}).`,
      res.status,
    );
  }

  return (await res.json()) as SecurityResponse;
}
