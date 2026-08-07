'use client';

import { memo } from 'react';
import type { ChangeEvent } from '../types';

// WhatChanged presents the dated evolution of the story. Dates are only ever
// the ones that were actually observed; when no timeline could be
// reconstructed the honest note is shown instead of a fabricated timeline.
function WhatChanged({ events, note }: { events?: ChangeEvent[]; note?: string }) {
  if (!events?.length) {
    return (
      <p className="text-sm text-slate-500 dark:text-slate-400">
        {note ?? 'No reliable evidence found.'}
      </p>
    );
  }

  return (
    <section aria-label="What changed over time">
      <ol role="list" className="space-y-2">
        {events.map((event, index) => (
          <li key={index} className="flex items-start gap-3">
            <span className="shrink-0 text-sm font-semibold tabular-nums text-slate-900 dark:text-slate-100">
              {event.date}
            </span>
            <span className="text-sm text-slate-700 dark:text-slate-300">{event.event}</span>
          </li>
        ))}
      </ol>
    </section>
  );
}

export default memo(WhatChanged);
