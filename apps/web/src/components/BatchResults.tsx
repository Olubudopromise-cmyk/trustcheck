'use client';

import { useMemo, useState } from 'react';
import type { BatchResultItem, BatchStatus } from '../hooks/useBatchVerification';
import type { VerifyResponse } from '../types';
import { downloadBatchJSON } from '../utils/report';
import StatusBadge from './StatusBadge';
import TrustScore from './TrustScore';
import TypeIcon, { typeLabel } from './TypeIcon';

type Filter = 'all' | 'verified' | 'warning' | 'invalid' | 'unreachable';
type SortKey = 'input' | 'type' | 'status' | 'score';
type SortDir = 'asc' | 'desc';

const FILTERS: { key: Filter; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'verified', label: 'Verified' },
  { key: 'warning', label: 'Warnings' },
  { key: 'invalid', label: 'Invalid' },
  { key: 'unreachable', label: 'Unreachable' },
];

const SORTABLE_COLUMNS: { key: SortKey; label: string }[] = [
  { key: 'input', label: 'Input' },
  { key: 'type', label: 'Type' },
  { key: 'status', label: 'Status' },
  { key: 'score', label: 'Trust Score' },
];

function matchesFilter(item: BatchResultItem, filter: Filter): boolean {
  if (filter === 'all') {
    return true;
  }
  if (!item.success) {
    return filter === 'unreachable';
  }
  return item.result?.status === filter;
}

function sortItems(items: BatchResultItem[], key: SortKey, dir: SortDir): BatchResultItem[] {
  const factor = dir === 'asc' ? 1 : -1;
  return [...items].sort((a, b) => {
    let cmp = 0;
    switch (key) {
      case 'input':
        cmp = a.input.localeCompare(b.input);
        break;
      case 'type':
        cmp = typeLabel(a.result?.type ?? '').localeCompare(typeLabel(b.result?.type ?? ''));
        break;
      case 'status':
        cmp = (a.result?.status ?? 'error').localeCompare(b.result?.status ?? 'error');
        break;
      case 'score':
        cmp =
          (a.result?.trustScore ?? -1) - (b.result?.trustScore ?? -1) ||
          a.input.localeCompare(b.input);
        break;
    }
    return cmp * factor;
  });
}

interface BatchResultsProps {
  status: BatchStatus;
  progress: { completed: number; total: number };
  results: BatchResultItem[];
}

export default function BatchResults({ status, progress, results }: BatchResultsProps) {
  const [filter, setFilter] = useState<Filter>('all');
  const [sortKey, setSortKey] = useState<SortKey>('input');
  const [sortDir, setSortDir] = useState<SortDir>('asc');

  const successful = useMemo(() => results.filter((item) => item.success), [results]);

  const exportable = useMemo<VerifyResponse[]>(
    () =>
      successful
        .map((item) => item.result)
        .filter((result): result is VerifyResponse => result !== null),
    [successful],
  );

  const verifiedCount = successful.filter((item) => item.result?.status === 'verified').length;
  const warningCount = successful.filter((item) => item.result?.status === 'warning').length;
  const invalidCount = successful.filter((item) => item.result?.status === 'invalid').length;
  const averageScore = successful.length
    ? Math.round(
        successful.reduce((sum, item) => sum + (item.result?.trustScore ?? 0), 0) /
          successful.length,
      )
    : null;

  const visible = useMemo(
    () =>
      sortItems(
        results.filter((item) => matchesFilter(item, filter)),
        sortKey,
        sortDir,
      ),
    [results, filter, sortKey, sortDir],
  );

  if (status === 'idle' && results.length === 0) {
    return null;
  }

  const toggleSort = (key: SortKey) => {
    if (key === sortKey) {
      setSortDir((dir) => (dir === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortKey(key);
      setSortDir('asc');
    }
  };

  const ariaSortFor = (key: SortKey): 'none' | 'ascending' | 'descending' => {
    if (key !== sortKey) {
      return 'none';
    }
    return sortDir === 'asc' ? 'ascending' : 'descending';
  };

  const summaryCards = [
    { label: 'Verified', value: verifiedCount, color: 'text-green-600 dark:text-green-400' },
    { label: 'Warnings', value: warningCount, color: 'text-yellow-600 dark:text-yellow-400' },
    { label: 'Invalid', value: invalidCount, color: 'text-red-600 dark:text-red-400' },
    {
      label: 'Average Score',
      value: averageScore === null ? '—' : `${averageScore} / 100`,
      color: 'text-slate-900 dark:text-slate-100',
    },
  ];

  return (
    <div className="space-y-4">
      {status === 'verifying' && (
        <div
          role="status"
          aria-live="polite"
          className="rounded-xl border border-slate-200 bg-white p-4 shadow dark:border-slate-800 dark:bg-slate-900"
        >
          <p className="text-sm font-medium text-slate-700 dark:text-slate-300">
            Verifying...{' '}
            <span className="font-semibold text-slate-900 dark:text-slate-100">
              {progress.completed} / {progress.total}
            </span>{' '}
            completed
          </p>
          <div
            className="mt-3 h-2 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700"
            aria-hidden="true"
          >
            <div
              className="h-full rounded-full bg-cyan-500 transition-all duration-150"
              style={{
                width: `${progress.total ? (progress.completed / progress.total) * 100 : 0}%`,
              }}
            />
          </div>
        </div>
      )}

      {status === 'done' && (
        <>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            {summaryCards.map((card) => (
              <div
                key={card.label}
                className="rounded-xl border border-slate-200 bg-white p-4 shadow dark:border-slate-800 dark:bg-slate-900"
              >
                <p className="text-xs font-medium text-slate-500 dark:text-slate-400">
                  {card.label}
                </p>
                <p className={`mt-1 text-2xl font-semibold tabular-nums ${card.color}`}>
                  {card.value}
                </p>
              </div>
            ))}
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex flex-wrap gap-2" role="group" aria-label="Filter results">
              {FILTERS.map((item) => {
                const active = filter === item.key;
                return (
                  <button
                    key={item.key}
                    type="button"
                    aria-pressed={active}
                    onClick={() => setFilter(item.key)}
                    className={`rounded-lg border px-3 py-1.5 text-xs font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 ${
                      active
                        ? 'border-cyan-500 bg-cyan-500 text-white'
                        : 'border-slate-300 text-slate-700 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-800'
                    }`}
                  >
                    {item.label}
                  </button>
                );
              })}
            </div>
            <button
              type="button"
              onClick={() => downloadBatchJSON(exportable)}
              disabled={exportable.length === 0}
              className="rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-800"
            >
              Export JSON
            </button>
          </div>

          <div className="overflow-x-auto rounded-xl border border-slate-200 shadow dark:border-slate-800">
            <table className="w-full min-w-[640px] border-collapse bg-white text-sm dark:bg-slate-900">
              <caption className="sr-only">Batch verification results</caption>
              <thead>
                <tr className="border-b border-slate-200 text-left dark:border-slate-800">
                  {SORTABLE_COLUMNS.map((col) => (
                    <th
                      key={col.key}
                      scope="col"
                      aria-sort={ariaSortFor(col.key)}
                      className="px-4 py-3"
                    >
                      <button
                        type="button"
                        onClick={() => toggleSort(col.key)}
                        className="flex items-center gap-1 text-xs font-semibold uppercase tracking-wide text-slate-500 transition hover:text-cyan-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 dark:text-slate-400 dark:hover:text-cyan-400"
                      >
                        {col.label}
                        <span aria-hidden="true" className="text-slate-400 dark:text-slate-500">
                          {sortKey === col.key ? (sortDir === 'asc' ? '↑' : '↓') : '↕'}
                        </span>
                        <span className="sr-only">
                          {sortKey === col.key
                            ? `, sorted ${sortDir === 'asc' ? 'ascending' : 'descending'}`
                            : ', not sorted'}
                        </span>
                      </button>
                    </th>
                  ))}
                  <th
                    scope="col"
                    className="px-4 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400"
                  >
                    Summary
                  </th>
                </tr>
              </thead>
              <tbody>
                {visible.map((item) => (
                  <tr
                    key={item.input}
                    className="border-b border-slate-200 last:border-0 dark:border-slate-800"
                  >
                    <td className="px-4 py-3 font-mono break-all text-slate-900 dark:text-slate-100">
                      {item.input}
                    </td>
                    <td className="px-4 py-3">
                      {item.success && item.result ? (
                        <span className="flex items-center gap-2">
                          <TypeIcon type={item.result.type} />
                          <span className="text-xs font-medium text-slate-700 dark:text-slate-300">
                            {typeLabel(item.result.type)}
                          </span>
                        </span>
                      ) : (
                        <span className="text-slate-400 dark:text-slate-500">—</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {item.success && item.result ? (
                        <StatusBadge status={item.result.status} verdict={item.result.verdict} />
                      ) : (
                        <span className="inline-flex items-center gap-1 rounded-full bg-red-100 px-3 py-1 text-xs font-semibold text-red-800 dark:bg-red-900/30 dark:text-red-300">
                          <span aria-hidden="true">✗</span>
                          ERROR
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {item.success && item.result ? (
                        <TrustScore score={item.result.trustScore} compact />
                      ) : (
                        <span className="text-slate-400 dark:text-slate-500">—</span>
                      )}
                    </td>
                    <td className="max-w-md px-4 py-3 break-words text-slate-700 dark:text-slate-300">
                      {item.success && item.result ? item.result.summary : item.error}
                    </td>
                  </tr>
                ))}
                {visible.length === 0 && (
                  <tr>
                    <td
                      colSpan={5}
                      className="px-4 py-8 text-center text-sm text-slate-500 dark:text-slate-400"
                    >
                      No results match this filter.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}
