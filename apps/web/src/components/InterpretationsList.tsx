'use client';

import { memo } from 'react';
import type { Interpretation } from '../types';

// confidenceColor maps an interpretation confidence to a bar color.
const confidenceColor = (value: number): string =>
  value >= 70 ? 'bg-green-500' : value >= 45 ? 'bg-yellow-400' : 'bg-red-400';

// InterpretationsList presents the 2-3 alternative readings of the input so a
// single meaning is never assumed.
function InterpretationsList({ interpretations }: { interpretations?: Interpretation[] }) {
  if (!interpretations?.length) {
    return null;
  }

  return (
    <section aria-label="Possible interpretations" className="space-y-3">
      {interpretations.map((interpretation) => (
        <article
          key={interpretation.title}
          className="rounded-lg border border-slate-200 p-3 dark:border-slate-700"
        >
          <div className="flex items-center justify-between gap-3">
            <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
              {interpretation.title}
            </h4>
            <span className="flex items-center gap-2">
              <span className="h-1.5 w-16 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700">
                <span
                  className={`block h-full rounded-full ${confidenceColor(interpretation.confidence)}`}
                  style={{ width: `${Math.min(100, Math.max(0, interpretation.confidence))}%` }}
                />
              </span>
              <span className="text-xs font-medium tabular-nums text-slate-500 dark:text-slate-400">
                {interpretation.confidence}%
              </span>
            </span>
          </div>
          <p className="mt-1.5 text-sm text-slate-700 dark:text-slate-300">
            {interpretation.explanation}
          </p>
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
            {interpretation.reasoning}
          </p>
        </article>
      ))}
    </section>
  );
}

export default memo(InterpretationsList);
