'use client';

import { memo } from 'react';
import type { WarningSeverity } from '../types';

const severityMeta: Record<WarningSeverity, { symbol: string; color: string }> = {
  high: { symbol: '\u26a0', color: 'bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300' },
  medium: {
    symbol: '\u26a0',
    color: 'bg-yellow-50 text-yellow-700 dark:bg-yellow-950/40 dark:text-yellow-300',
  },
  low: {
    symbol: '\u24d8',
    color: 'bg-blue-50 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300',
  },
};

// WarningSignals presents the structured misinformation indicators detected in
// the input, each with its severity.
function WarningSignals({
  signals,
}: {
  signals?: { label: string; severity: WarningSeverity; description: string }[];
}) {
  if (!signals?.length) {
    return null;
  }

  return (
    <section aria-label="Warning signals">
      <ul role="list" className="space-y-2">
        {signals.map((signal) => {
          const meta = severityMeta[signal.severity] ?? severityMeta.low;
          return (
            <li
              key={signal.label}
              className={`flex items-start gap-2 rounded-lg px-3 py-2 text-sm ${meta.color}`}
            >
              <span aria-hidden="true" className="mt-0.5 shrink-0">
                {meta.symbol}
              </span>
              <span className="min-w-0">
                <span className="font-semibold">{signal.label}</span>
                <span className="block text-xs opacity-80">{signal.description}</span>
              </span>
            </li>
          );
        })}
      </ul>
    </section>
  );
}

export default memo(WarningSignals);
