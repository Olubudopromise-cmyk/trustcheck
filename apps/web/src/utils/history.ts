import type { VerificationHistoryItem, VerifyResponse } from '../types';

const HISTORY_KEY = 'trustcheck:verification-history';
const MAX_ENTRIES = 20;

// makeId prefers crypto.randomUUID (spec); falls back for non-secure contexts.
function makeId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

// isHistoryItem guards against corrupt or foreign data written to the same key.
function isHistoryItem(value: unknown): value is VerificationHistoryItem {
  if (typeof value !== 'object' || value === null) {
    return false;
  }
  const item = value as Record<string, unknown>;
  if (
    typeof item.id !== 'string' ||
    typeof item.timestamp !== 'number' ||
    typeof item.input !== 'string'
  ) {
    return false;
  }
  const result = item.result as VerifyResponse | null;
  return (
    typeof result === 'object' &&
    result !== null &&
    typeof result.input === 'string' &&
    typeof result.type === 'string' &&
    typeof result.status === 'string' &&
    typeof result.trustScore === 'number' &&
    typeof result.summary === 'string' &&
    Array.isArray(result.evidence)
  );
}

export function loadHistory(): VerificationHistoryItem[] {
  if (typeof window === 'undefined') {
    return [];
  }
  try {
    const raw = window.localStorage.getItem(HISTORY_KEY);
    if (!raw) {
      return [];
    }
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed.filter(isHistoryItem).slice(0, MAX_ENTRIES);
  } catch {
    return [];
  }
}

export function saveHistory(items: VerificationHistoryItem[]): void {
  if (typeof window === 'undefined') {
    return;
  }
  try {
    window.localStorage.setItem(HISTORY_KEY, JSON.stringify(items.slice(0, MAX_ENTRIES)));
  } catch {
    // Quota or serialization errors must never break a verification.
  }
}

// addHistoryItem prepends a new entry, drops any duplicate for the same input
// (case-insensitive, trimmed) and keeps the newest MAX_ENTRIES. It persists the
// result to localStorage and returns the updated list.
export function addHistoryItem(input: string, result: VerifyResponse): VerificationHistoryItem[] {
  const normalized = input.trim().toLowerCase();
  const item: VerificationHistoryItem = {
    id: makeId(),
    timestamp: Date.now(),
    input: input.trim(),
    result,
  };
  const next = [
    item,
    ...loadHistory().filter((entry) => entry.input.trim().toLowerCase() !== normalized),
  ].slice(0, MAX_ENTRIES);
  saveHistory(next);
  return next;
}

export function clearHistory(): void {
  if (typeof window === 'undefined') {
    return;
  }
  try {
    window.localStorage.removeItem(HISTORY_KEY);
  } catch {
    // Ignore; the in-memory list is cleared regardless by the caller.
  }
}
