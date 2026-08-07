'use client';

import type { Verdict } from '../types';

export const VERDICT_META: Record<string, { symbol: string; label: string; color: string }> = {
  High: {
    symbol: '\u2713',
    label: 'HIGH TRUST',
    color: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
  },
  Medium: {
    symbol: '\u26a0',
    label: 'MEDIUM TRUST',
    color: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
  },
  Low: {
    symbol: '\u2717',
    label: 'LOW TRUST',
    color: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
  },
};

const statusMeta: Record<string, { symbol: string; label: string; color: string }> = {
  verified: {
    symbol: '\u2713',
    label: 'VERIFIED',
    color: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
  },
  warning: {
    symbol: '\u26a0',
    label: 'WARNING',
    color: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
  },
  invalid: {
    symbol: '\u2717',
    label: 'INVALID',
    color: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
  },
  private: {
    symbol: '\U0001f512',
    label: 'PRIVATE',
    color: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
  },
  local: {
    symbol: '\U0001f4f1',
    label: 'LOCAL',
    color: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300',
  },
  unreachable: {
    symbol: '\U0001f4f4',
    label: 'UNREACHABLE',
    color: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300',
  },
  not_implemented: {
    symbol: '\u29f6',
    label: 'NOT IMPLEMENTED',
    color: 'bg-slate-200 text-slate-800 dark:bg-slate-700 dark:text-slate-300',
  },
  classified: {
    symbol: '\u2713',
    label: 'CLASSIFIED',
    color: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-300',
  },
};

export default function StatusBadge({ status, verdict }: { status: string; verdict?: Verdict }) {
  const meta =
    (verdict && VERDICT_META[verdict]) ?? statusMeta[status] ?? statusMeta.not_implemented;
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-semibold ${meta.color}`}
      aria-label={`status: ${meta.label.toLowerCase()}`}
    >
      <span aria-hidden="true">{meta.symbol}</span>
      {meta.label}
    </span>
  );
}
