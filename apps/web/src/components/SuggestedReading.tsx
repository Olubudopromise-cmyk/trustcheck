'use client';

import { memo } from 'react';
import type { SuggestedReading as SuggestedReadingType } from '../types';

// SuggestedReading presents recommended material to consult. Links are never
// fabricated; when no specific articles could be identified an honest note is
// shown alongside the search-target guidance.
function SuggestedReading({ items, note }: { items?: SuggestedReadingType[]; note?: string }) {
  if (!items?.length) {
    return (
      <p className="text-sm text-slate-500 dark:text-slate-400">
        {note ?? 'No reliable reading suggestions identified.'}
      </p>
    );
  }

  return (
    <section aria-label="Suggested reading" className="space-y-3">
      {note && <p className="text-xs text-slate-500 dark:text-slate-400">{note}</p>}
      <ul role="list" className="space-y-2">
        {items.map((item) => (
          <li
            key={item.title}
            className="rounded-lg border border-slate-200 p-3 dark:border-slate-700"
          >
            <span className="block text-sm font-medium text-slate-900 dark:text-slate-100">
              {item.title}
            </span>
            <span className="mt-0.5 block text-xs text-slate-500 dark:text-slate-400">
              {item.publisher}
            </span>
            <span className="mt-1 block text-sm text-slate-700 dark:text-slate-300">
              {item.whyItHelps}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

export default memo(SuggestedReading);
