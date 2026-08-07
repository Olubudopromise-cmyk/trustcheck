'use client';

import { memo } from 'react';
import type { ConfidenceBreakdown as ConfidenceBreakdownType } from '../types';

// metricColor maps a metric score to a bar color (green/yellow/red).
const metricColor = (value: number): string =>
  value >= 70 ? 'bg-green-500' : value >= 45 ? 'bg-yellow-400' : 'bg-red-400';

// ConfidenceBreakdown shows the user-facing metrics that explain the overall
// confidence. Only user-friendly metrics are exposed — never the hidden scoring
// algorithm.
function ConfidenceBreakdown({ breakdown }: { breakdown?: ConfidenceBreakdownType }) {
  if (!breakdown || breakdown.metrics.length === 0) {
    return null;
  }

  return (
    <section aria-label="Confidence breakdown">
      <div className="mb-4 flex items-center gap-3">
        <span className="text-3xl font-bold tabular-nums text-slate-900 dark:text-slate-100">
          {breakdown.overall}%
        </span>
        <span className="text-sm text-slate-500 dark:text-slate-400">Overall confidence</span>
      </div>
      <ul role="list" className="space-y-3">
        {breakdown.metrics.map((metric) => (
          <li key={metric.name}>
            <div className="flex items-center justify-between gap-3">
              <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
                {metric.name}
              </span>
              <span className="text-sm font-semibold tabular-nums text-slate-900 dark:text-slate-100">
                {metric.score}%
              </span>
            </div>
            <div
              role="img"
              aria-label={`${metric.name}: ${metric.score}%`}
              className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700"
            >
              <span
                className={`block h-full rounded-full ${metricColor(metric.score)}`}
                style={{ width: `${Math.min(100, Math.max(0, metric.score))}%` }}
              />
            </div>
            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{metric.note}</p>
          </li>
        ))}
      </ul>
    </section>
  );
}

export default memo(ConfidenceBreakdown);
