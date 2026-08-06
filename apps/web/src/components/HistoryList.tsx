'use client';

import { memo } from 'react';
import type { VerificationHistoryItem } from '../types';
import { formatRelativeTime } from '../utils/relativeTime';
import StatusBadge from './StatusBadge';
import TypeIcon from './TypeIcon';

interface HistoryListProps {
  history: VerificationHistoryItem[];
  onReopen: (item: VerificationHistoryItem) => void;
  onClear: () => void;
}

function HistoryList({ history, onReopen, onClear }: HistoryListProps) {
  const handleClear = () => {
    if (window.confirm('Clear all verification history?')) {
      onClear();
    }
  };

  return (
    <section aria-label="Recent Activity">
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
          Recent Activity
        </h2>
        {history.length > 0 && (
          <button
            type="button"
            onClick={handleClear}
            className="text-xs font-medium text-slate-500 transition hover:text-red-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 rounded dark:text-slate-400 dark:hover:text-red-400"
          >
            Clear History
          </button>
        )}
      </div>

      {history.length === 0 ? (
        <p className="text-sm text-slate-500 dark:text-slate-400">No recent verifications.</p>
      ) : (
        <ul role="list" className="space-y-2">
          {history.map((item) => (
            <li key={item.id}>
              <div
                role="button"
                tabIndex={0}
                aria-label={`Reopen ${item.input}`}
                onClick={() => onReopen(item)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    onReopen(item);
                  }
                }}
                className="flex cursor-pointer items-center gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3 text-left transition hover:border-cyan-400 hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 dark:border-slate-800 dark:bg-slate-900 dark:hover:border-cyan-500"
              >
                <TypeIcon type={item.result.type} />
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-mono text-sm text-slate-900 dark:text-slate-100">
                    {item.input}
                  </span>
                  <span className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-slate-500 dark:text-slate-400">
                    <StatusBadge status={item.result.status} />
                    <span aria-hidden="true">•</span>
                    <span>
                      <span className="sr-only">score</span>
                      {item.result.trustScore}
                    </span>
                    <span aria-hidden="true">•</span>
                    <span>{formatRelativeTime(item.timestamp)}</span>
                  </span>
                </span>
                <span aria-hidden="true" className="shrink-0 text-slate-400 dark:text-slate-500">
                  ›
                </span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

export default memo(HistoryList);
