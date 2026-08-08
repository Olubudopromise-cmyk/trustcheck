'use client';

import { memo, useCallback } from 'react';
import type { VerificationHistoryItem } from '../types';
import { formatRelativeTime } from '../utils/relativeTime';
import StatusBadge from './StatusBadge';
import TypeIcon from './TypeIcon';

interface ResearchSidebarProps {
  history: VerificationHistoryItem[];
  activeId: string | null;
  onSelect: (item: VerificationHistoryItem) => void;
  onNewResearch: () => void;
  onDelete: (id: string) => void;
  isOpen: boolean;
  onClose: () => void;
}

function ResearchSidebar({
  history,
  activeId,
  onSelect,
  onNewResearch,
  onDelete,
  isOpen,
  onClose,
}: ResearchSidebarProps) {
  const handleDelete = useCallback(
    (e: React.MouseEvent, id: string) => {
      e.stopPropagation();
      onDelete(id);
    },
    [onDelete],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent, item: VerificationHistoryItem) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        onSelect(item);
      }
    },
    [onSelect],
  );

  return (
    <>
      {/* Mobile overlay */}
      {isOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={onClose}
          aria-hidden="true"
        />
      )}

      {/* Sidebar */}
      <aside
        className={`
          fixed inset-y-0 left-0 z-50 w-72 flex flex-col
          border-r border-slate-200 bg-white
          dark:border-slate-800 dark:bg-slate-900
          transform transition-transform duration-200 ease-out
          lg:relative lg:translate-x-0 lg:z-auto
          ${isOpen ? 'translate-x-0' : '-translate-x-full'}
        `}
        aria-label="Research history"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-slate-800">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
            Research Sessions
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 lg:hidden dark:hover:bg-slate-800 dark:hover:text-slate-300"
            aria-label="Close sidebar"
          >
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* New Research button */}
        <div className="border-b border-slate-200 px-3 py-3 dark:border-slate-800">
          <button
            type="button"
            onClick={onNewResearch}
            className="flex w-full items-center justify-center gap-2 rounded-xl border-2 border-dashed border-slate-300 bg-slate-50 px-4 py-3 text-sm font-medium text-slate-600 transition hover:border-cyan-400 hover:bg-cyan-50 hover:text-cyan-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-400 dark:hover:border-cyan-500 dark:hover:bg-slate-800 dark:hover:text-cyan-400"
          >
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 4v16m8-8H4"
              />
            </svg>
            New Research
          </button>
        </div>

        {/* History list */}
        <nav className="flex-1 overflow-y-auto px-2 py-2">
          {history.length === 0 ? (
            <div className="px-3 py-8 text-center">
              <p className="text-sm text-slate-500 dark:text-slate-400">
                No research sessions yet.
              </p>
              <p className="mt-1 text-xs text-slate-400 dark:text-slate-500">
                Start a verification to begin.
              </p>
            </div>
          ) : (
            <ul role="list" className="space-y-1">
              {history.map((item) => {
                const isActive = item.id === activeId;
                return (
                  <li key={item.id}>
                    <div
                      role="button"
                      tabIndex={0}
                      aria-label={`Select research: ${item.input}`}
                      aria-current={isActive ? 'true' : undefined}
                      onClick={() => onSelect(item)}
                      onKeyDown={(e) => handleKeyDown(e, item)}
                      className={`
                        group flex cursor-pointer items-start gap-3 rounded-xl px-3 py-2.5
                        transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500
                        ${
                          isActive
                            ? 'bg-cyan-50 border border-cyan-200 dark:bg-cyan-950/30 dark:border-cyan-800'
                            : 'border border-transparent hover:bg-slate-50 dark:hover:bg-slate-800/60'
                        }
                      `}
                    >
                      <div className="mt-0.5 shrink-0">
                        <TypeIcon type={item.result.type} />
                      </div>
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium text-slate-900 dark:text-slate-100">
                          {item.input.length > 30 ? item.input.slice(0, 30) + '...' : item.input}
                        </p>
                        <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1">
                          <StatusBadge status={item.result.status} verdict={item.result.verdict} />
                          <span className="text-xs tabular-nums text-slate-500 dark:text-slate-400">
                            {item.result.trustScore}
                          </span>
                          <span className="text-xs text-slate-400 dark:text-slate-500">
                            {formatRelativeTime(item.timestamp)}
                          </span>
                        </div>
                      </div>
                      <button
                        type="button"
                        onClick={(e) => handleDelete(e, item.id)}
                        className="shrink-0 rounded-lg p-1 text-slate-300 opacity-0 transition hover:bg-red-50 hover:text-red-500 group-hover:opacity-100 dark:text-slate-600 dark:hover:bg-red-950/30 dark:hover:text-red-400"
                        aria-label={`Delete research: ${item.input}`}
                      >
                        <svg
                          className="h-4 w-4"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                          />
                        </svg>
                      </button>
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </nav>

        {/* Footer */}
        <div className="border-t border-slate-200 px-4 py-3 dark:border-slate-800">
          <p className="text-xs text-slate-400 dark:text-slate-500">
            {history.length} session{history.length !== 1 ? 's' : ''}
          </p>
        </div>
      </aside>
    </>
  );
}

export default memo(ResearchSidebar);
