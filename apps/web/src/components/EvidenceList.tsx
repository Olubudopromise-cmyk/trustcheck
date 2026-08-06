'use client';

import type { Evidence, EvidenceResult } from '../types';

const resultMeta: Record<EvidenceResult, { symbol: string; label: string; color: string }> = {
  pass: { symbol: '\u2713', label: 'passed', color: 'text-green-500' },
  warning: { symbol: '\u26a0', label: 'warning', color: 'text-yellow-500 dark:text-yellow-400' },
  fail: { symbol: '\u2717', label: 'failed', color: 'text-red-500' },
  info: { symbol: '\u24d8', label: 'info', color: 'text-blue-500' },
};

const formatPoints = (points: number): string => (points > 0 ? `+${points}` : String(points));

export default function EvidenceList({ evidence }: { evidence: Evidence[] }) {
  if (evidence.length === 0) {
    return (
      <p className="text-sm text-slate-500 dark:text-slate-400">
        No verification details available.
      </p>
    );
  }

  return (
    <section aria-label="Evidence breakdown">
      <h3 className="text-sm text-slate-500 dark:text-slate-400">Evidence</h3>
      <div className="mt-2 border-t border-slate-200 dark:border-slate-800" />
      <ul role="list" className="mt-3 space-y-2">
        {evidence.map((item, index) => {
          const meta = resultMeta[item.result];
          return (
            <li
              key={`${item.label}-${index}`}
              role="listitem"
              className="flex min-w-0 items-baseline justify-between gap-3"
            >
              <span className="flex min-w-0 items-baseline gap-2">
                <span aria-hidden="true" className={`shrink-0 ${meta.color}`}>
                  {meta.symbol}
                </span>
                <span className="min-w-0 break-words text-sm text-slate-700 dark:text-slate-300">
                  {item.label}
                </span>
              </span>
              <span className="sr-only">{meta.label}</span>
              <span className={`shrink-0 text-sm font-semibold tabular-nums ${meta.color}`}>
                {formatPoints(item.points)}
              </span>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
