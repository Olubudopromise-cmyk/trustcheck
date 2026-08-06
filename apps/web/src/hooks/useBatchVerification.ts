'use client';

import { useCallback, useState } from 'react';
import type { VerifyResponse } from '../types';

export const BATCH_MAX_INPUTS = 100;

export interface BatchResultItem {
  input: string;
  success: boolean;
  result: VerifyResponse | null;
  error: string | null;
}

export type BatchStatus = 'idle' | 'verifying' | 'done';

// parseBatchInput normalizes pasted batch text into a deduplicated, capped list
// of inputs. Blank lines are ignored and identical inputs (case-insensitive,
// trimmed) are collapsed so the same thing is never verified twice.
export function parseBatchInput(text: string): string[] {
  const seen = new Set<string>();
  const inputs: string[] = [];
  for (const raw of text.split(/\r?\n/)) {
    const trimmed = raw.trim();
    if (!trimmed) {
      continue;
    }
    const key = trimmed.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    inputs.push(trimmed);
    if (inputs.length >= BATCH_MAX_INPUTS) {
      break;
    }
  }
  return inputs;
}

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// useBatchVerification verifies a list of inputs concurrently against the
// existing /verify endpoint. Promise.allSettled keeps one failed request from
// blocking the rest, and progress is reported live as each request settles.
export function useBatchVerification() {
  const [status, setStatus] = useState<BatchStatus>('idle');
  const [progress, setProgress] = useState({ completed: 0, total: 0 });
  const [results, setResults] = useState<BatchResultItem[]>([]);

  const verifyBatch = useCallback(
    async (inputs: string[], onVerified?: (input: string, result: VerifyResponse) => void) => {
      setStatus('verifying');
      setProgress({ completed: 0, total: inputs.length });
      setResults([]);

      let completed = 0;
      const items: BatchResultItem[] = inputs.map((input) => ({
        input,
        success: false,
        result: null,
        error: null,
      }));

      const tasks = items.map(async (item, index) => {
        try {
          const res = await fetch(`${API_URL}/verify`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ input: item.input }),
          });
          if (!res.ok) {
            const err: { error?: string } = await res.json().catch(() => ({}));
            throw new Error(err.error || `Verification failed (HTTP ${res.status}).`);
          }
          const data = (await res.json()) as VerifyResponse;
          items[index] = { ...item, success: true, result: data };
          onVerified?.(item.input, data);
        } catch (e) {
          items[index] = {
            ...item,
            success: false,
            error: e instanceof Error ? e.message : 'Something went wrong. Is the backend running?',
          };
        } finally {
          completed += 1;
          setProgress({ completed, total: inputs.length });
        }
      });

      await Promise.allSettled(tasks);
      setResults(items);
      setStatus('done');
    },
    [],
  );

  return { status, progress, results, verifyBatch };
}
