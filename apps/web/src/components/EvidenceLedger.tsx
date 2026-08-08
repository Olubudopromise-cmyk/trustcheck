'use client';

import { memo } from 'react';
import type {
  EvidenceLedger as EvidenceLedgerType,
  LedgerEntry,
  SourceIntelligence,
} from '../types';

interface EvidenceLedgerProps {
  ledger: EvidenceLedgerType;
}

function SourceBadge({ source }: { source: SourceIntelligence }) {
  const typeColors: Record<string, string> = {
    official: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300',
    institutional: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300',
    academic: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300',
    journalism: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300',
    community: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
    commercial: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300',
    unknown: 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400',
  };

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <span
        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
          typeColors[source.sourceType] ?? typeColors.unknown
        }`}
      >
        {source.sourceType}
      </span>
      {source.isOfficial && (
        <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-300">
          Official
        </span>
      )}
      {source.isIndependent && (
        <span className="rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700 dark:bg-green-900/30 dark:text-green-300">
          Independent
        </span>
      )}
      <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-800 dark:text-slate-400">
        {source.relation}
      </span>
    </div>
  );
}

function LedgerEntryCard({
  entry,
  type,
}: {
  entry: LedgerEntry;
  type: 'supporting' | 'contradicting';
}) {
  const borderColor = type === 'supporting' ? 'border-l-green-400' : 'border-l-red-400';
  const bgColor =
    type === 'supporting'
      ? 'bg-green-50/50 dark:bg-green-950/10'
      : 'bg-red-50/50 dark:bg-red-950/10';

  return (
    <div className={`rounded-r-lg border-l-4 ${borderColor} ${bgColor} p-3`}>
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium text-slate-700 dark:text-slate-300">
            {entry.source.title}
          </p>
          {entry.summary && (
            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{entry.summary}</p>
          )}
        </div>
        <div className="shrink-0 text-right">
          <span className="text-xs tabular-nums text-slate-500 dark:text-slate-400">
            {entry.strength}%
          </span>
        </div>
      </div>
      <div className="mt-2">
        <SourceBadge source={entry.source} />
      </div>
    </div>
  );
}

function EvidenceLedger({ ledger }: EvidenceLedgerProps) {
  // Safe defaults for every field: persisted ledgers from older app versions
  // may be missing counters or buckets, and rendering must never crash.
  const supporting = ledger.supporting ?? [];
  const contradicting = ledger.contradicting ?? [];
  const unknown = ledger.unknown ?? [];
  const totalSources = ledger.totalSources ?? supporting.length + contradicting.length;
  const independentCount = ledger.independentCount ?? 0;

  return (
    <section aria-label="Evidence Ledger" className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
          Evidence Ledger
        </h3>
        <div className="flex items-center gap-3 text-xs text-slate-500 dark:text-slate-400">
          <span>{totalSources} sources</span>
          <span>{independentCount} independent</span>
        </div>
      </div>

      {/* Claim */}
      <div className="rounded-lg border border-slate-200 bg-slate-50 p-3 dark:border-slate-800 dark:bg-slate-800/50">
        <p className="text-sm font-medium text-slate-700 dark:text-slate-300">
          &ldquo;{ledger.claim ?? ''}&rdquo;
        </p>
      </div>

      {/* Supporting evidence */}
      <div>
        <h4 className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-green-700 dark:text-green-400">
          <span className="h-2 w-2 rounded-full bg-green-500" />
          Supporting ({supporting.length})
        </h4>
        {supporting.length === 0 ? (
          <p className="text-xs text-slate-500 dark:text-slate-400">
            No supporting evidence found.
          </p>
        ) : (
          <div className="space-y-2">
            {supporting.map((entry, i) => (
              <LedgerEntryCard key={`s-${i}`} entry={entry} type="supporting" />
            ))}
          </div>
        )}
      </div>

      {/* Contradicting evidence */}
      <div>
        <h4 className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-red-700 dark:text-red-400">
          <span className="h-2 w-2 rounded-full bg-red-500" />
          Contradicting ({contradicting.length})
        </h4>
        {contradicting.length === 0 ? (
          <p className="text-xs text-slate-500 dark:text-slate-400">
            No contradicting evidence found.
          </p>
        ) : (
          <div className="space-y-2">
            {contradicting.map((entry, i) => (
              <LedgerEntryCard key={`c-${i}`} entry={entry} type="contradicting" />
            ))}
          </div>
        )}
      </div>

      {/* Unknown */}
      {unknown.length > 0 && (
        <div>
          <h4 className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
            <span className="h-2 w-2 rounded-full bg-slate-400" />
            Unknown
          </h4>
          <ul className="space-y-1">
            {unknown.map((item, i) => (
              <li key={i} className="text-xs text-slate-500 dark:text-slate-400">
                {item}
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  );
}

export default memo(EvidenceLedger);
