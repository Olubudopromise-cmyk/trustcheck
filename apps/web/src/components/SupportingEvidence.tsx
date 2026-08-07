'use client';

import { memo } from 'react';
import type { SourceGroup } from '../types';

// credibilityColor styles the credibility badge.
const credibilityColor: Record<string, string> = {
  high: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
  medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
  low: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
};

// SupportingEvidence presents evidence grouped by the category of source it
// came from. Each item carries a credibility level, its source, and a summary.
function SupportingEvidence({ groups }: { groups?: SourceGroup[] }) {
  if (!groups?.length) {
    return (
      <p className="text-sm text-slate-500 dark:text-slate-400">
        No reliable supporting evidence found.
      </p>
    );
  }

  return (
    <section aria-label="Supporting evidence by source" className="space-y-4">
      {groups.map((group) => (
        <div key={group.category}>
          <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
            {group.category}
          </h4>
          <ul role="list" className="mt-2 space-y-2">
            {group.items.map((item) => (
              <li
                key={`${group.category}-${item.title}`}
                className="rounded-lg border border-slate-200 p-3 dark:border-slate-700"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <span className="block text-sm font-medium text-slate-900 dark:text-slate-100">
                      {item.title}
                    </span>
                    <span className="block text-xs text-slate-500 dark:text-slate-400">
                      {item.source}
                    </span>
                  </div>
                  <span
                    className={`shrink-0 rounded-full px-2 py-0.5 text-xs font-medium capitalize ${credibilityColor[item.credibility] ?? credibilityColor.medium}`}
                  >
                    {item.credibility}
                  </span>
                </div>
                <p className="mt-2 text-sm text-slate-700 dark:text-slate-300">{item.summary}</p>
                {item.publicationDate && (
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Published: {item.publicationDate}
                  </p>
                )}
              </li>
            ))}
          </ul>
        </div>
      ))}
    </section>
  );
}

export default memo(SupportingEvidence);
