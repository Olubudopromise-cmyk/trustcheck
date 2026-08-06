'use client';

import { useCallback, useSyncExternalStore } from 'react';
import type { VerificationHistoryItem, VerifyResponse } from '../types';
import { addHistoryItem, clearHistory, loadHistory } from '../utils/history';

// useVerificationHistory mirrors localStorage-backed verification history into
// React state. localStorage is treated as an external store via
// useSyncExternalStore: the server snapshot is empty so SSR HTML and the
// client's first hydration always match, then the real history is surfaced
// after mount. The module-level `items` cache keeps the snapshot reference
// stable between writes (required by useSyncExternalStore).

type Listener = () => void;

let items: VerificationHistoryItem[] | null = null;
const listeners = new Set<Listener>();

function getSnapshot(): VerificationHistoryItem[] {
  if (items === null) {
    items = loadHistory();
  }
  return items;
}

function getServerSnapshot(): VerificationHistoryItem[] {
  return [];
}

function subscribe(listener: Listener): () => void {
  listeners.add(listener);
  // Cross-tab sync: a 'storage' event in another tab invalidates the cache and
  // re-reads from localStorage.
  const onStorage = () => {
    items = null;
    listener();
  };
  window.addEventListener('storage', onStorage);
  return () => {
    listeners.delete(listener);
    window.removeEventListener('storage', onStorage);
  };
}

function emit(): void {
  listeners.forEach((listener) => listener());
}

export function useVerificationHistory() {
  const history = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);

  const remember = useCallback((input: string, result: VerifyResponse) => {
    items = addHistoryItem(input, result);
    emit();
  }, []);

  const clear = useCallback(() => {
    clearHistory();
    items = [];
    emit();
  }, []);

  return { history, remember, clear };
}
