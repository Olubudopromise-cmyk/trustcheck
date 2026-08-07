'use client';

import { memo } from 'react';
import type { MissingInfo } from '../types';

// MissingInformation presents the "What is missing?" section. Each item is
// grounded in a check that could not be run, an unknown fact, or a detected
// warning — nothing is invented.
function MissingInformation({ items }: { items?: MissingInfo[] }) {
  if (!items?.length) {
    return (
      <p className="text-sm text-slate-500 dark:text-slate-400">
        No missing information identified in the checks performed.
      </p>
    );
  }

  return (
    <section aria-label="Missing information" className="space-y-2">
      <ul role="list" className="space-y-2">
        {items.map((item) => (
          <li
            key={item.item}
            className="rounded-lg border border-slate-200 p-3 dark:border-slate-700"
          >
            <span className="block text-sm font-medium text-slate-900 dark:text-slate-100">
              {item.item}
            </span>
            <span className="mt-0.5 block text-sm text-slate-500 dark:text-slate-400">
              {item.whyItMatters}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

export default memo(MissingInformation);
