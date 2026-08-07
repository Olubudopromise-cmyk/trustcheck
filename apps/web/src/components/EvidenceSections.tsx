'use client';

import { memo } from 'react';
import type { EvidenceItem, EvidenceResult } from '../types';

const resultMeta: Record<EvidenceResult, { symbol: string; label: string; color: string }> = {
  pass: { symbol: '\u2713', label: 'passed', color: 'text-green-500' },
  warning: { symbol: '\u26a0', label: 'warning', color: 'text-yellow-500 dark:text-yellow-400' },
  fail: { symbol: '\u2717', label: 'failed', color: 'text-red-500' },
  info: { symbol: '\u24d8', label: 'info', color: 'text-blue-500' },
};

const formatPoints = (points: number): string => (points > 0 ? `+${points}` : String(points));

function EvidenceRow({ item }: { item: EvidenceItem }) {
  const meta = resultMeta[item.result] ?? resultMeta.info;
  return (
    <li className="flex min-w-0 items-baseline justify-between gap-3">
      <span className="flex min-w-0 items-baseline gap-2">
        <span aria-hidden="true" className={`shrink-0 ${meta.color}`}>
          {meta.symbol}
        </span>
        <span className="min-w-0 break-words text-sm text-slate-700 dark:text-slate-300">
          {item.label}
          {item.note ? (
            <span className="ml-1 text-xs text-slate-500 dark:text-slate-400">— {item.note}</span>
          ) : null}
        </span>
      </span>
      <span className="sr-only">{meta.label}</span>
      <span className={`shrink-0 text-sm font-semibold tabular-nums ${meta.color}`}>
        {formatPoints(item.points)}
      </span>
    </li>
  );
}

// EvidenceSection renders a single evidence group (supporting or contradicting)
// as a collapsible-free, scannable list. The whole result page is assembled
// from these groups so sections stay reusable.
function EvidenceSection({
  title,
  items,
  emptyMessage,
  tone,
}: {
  title: string;
  items: EvidenceItem[];
  emptyMessage: string;
  tone: 'support' | 'contradict';
}) {
  const accent =
    tone === 'support' ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400';

  return (
    <section aria-label={title}>
      <h3 className={`text-sm font-semibold ${accent}`}>{title}</h3>
      {items.length === 0 ? (
        <p className="mt-2 text-sm text-slate-500 dark:text-slate-400">{emptyMessage}</p>
      ) : (
        <ul role="list" className="mt-2 space-y-2">
          {items.map((item, index) => (
            <EvidenceRow key={`${item.label}-${index}`} item={item} />
          ))}
        </ul>
      )}
    </section>
  );
}

// EvidenceSections presents the supporting and contradicting evidence plus the
// explicit statements about missing and unknown information.
function EvidenceSections({
  evidenceFor,
  evidenceAgainst,
  missingEvidence,
  unknownInformation,
}: {
  evidenceFor?: EvidenceItem[];
  evidenceAgainst?: EvidenceItem[];
  missingEvidence?: string[];
  unknownInformation?: string[];
}) {
  const hasEvidence = evidenceFor?.length || evidenceAgainst?.length;
  const hasGaps = missingEvidence?.length || unknownInformation?.length;

  if (!hasEvidence && !hasGaps) {
    return null;
  }

  return (
    <div className="space-y-4">
      <EvidenceSection
        title="Supporting Evidence"
        tone="support"
        items={evidenceFor ?? []}
        emptyMessage="No supporting evidence was found."
      />
      <EvidenceSection
        title="Contradicting Evidence"
        tone="contradict"
        items={evidenceAgainst ?? []}
        emptyMessage="No contradicting evidence was found."
      />
      {hasGaps && (
        <section aria-label="Missing and unknown information" className="space-y-4">
          <div>
            <h3 className="text-sm font-semibold text-slate-500 dark:text-slate-400">
              Missing Evidence
            </h3>
            <ul role="list" className="mt-2 space-y-1.5">
              {(missingEvidence ?? []).map((item) => (
                <li key={item} className="text-sm text-slate-600 dark:text-slate-400">
                  {item}
                </li>
              ))}
            </ul>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-slate-500 dark:text-slate-400">
              Unknown Information
            </h3>
            <ul role="list" className="mt-2 space-y-1.5">
              {(unknownInformation ?? []).map((item) => (
                <li key={item} className="text-sm text-slate-600 dark:text-slate-400">
                  {item}
                </li>
              ))}
            </ul>
          </div>
        </section>
      )}
    </div>
  );
}

export default memo(EvidenceSections);
