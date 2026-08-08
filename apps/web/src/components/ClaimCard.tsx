'use client';

import { memo, useState } from 'react';
import type {
  Claim,
  ClaimStatus,
  EvidenceItem,
  Contradiction,
  MissingInfo,
  ReasoningStep,
  Recommendation,
} from '../types';

const STATUS_META: Record<
  ClaimStatus,
  { symbol: string; label: string; color: string; bgColor: string }
> = {
  verified: {
    symbol: '\u2713',
    label: 'Verified',
    color: 'text-green-700 dark:text-green-300',
    bgColor: 'bg-green-50 border-green-200 dark:bg-green-950/20 dark:border-green-800',
  },
  partially_verified: {
    symbol: '\u26a0',
    label: 'Partially Verified',
    color: 'text-yellow-700 dark:text-yellow-300',
    bgColor: 'bg-yellow-50 border-yellow-200 dark:bg-yellow-950/20 dark:border-yellow-800',
  },
  unverified: {
    symbol: '\u2717',
    label: 'Unverified',
    color: 'text-red-700 dark:text-red-300',
    bgColor: 'bg-red-50 border-red-200 dark:bg-red-950/20 dark:border-red-800',
  },
  no_reliable_evidence: {
    symbol: '?',
    label: 'No Reliable Evidence',
    color: 'text-slate-500 dark:text-slate-400',
    bgColor: 'bg-slate-50 border-slate-200 dark:bg-slate-800 dark:border-slate-700',
  },
};

function ClaimStatusBadge({ status }: { status: ClaimStatus }) {
  const meta = STATUS_META[status] ?? STATUS_META.no_reliable_evidence;
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-semibold ${meta.color}`}
    >
      <span aria-hidden="true">{meta.symbol}</span>
      {meta.label}
    </span>
  );
}

function EvidenceSection({
  title,
  items,
  type,
}: {
  title: string;
  items: EvidenceItem[];
  type: 'pass' | 'fail' | 'warning' | 'info';
}) {
  if (items.length === 0) return null;

  const colorMap = {
    pass: 'border-l-green-400 bg-green-50/50 dark:bg-green-950/10',
    fail: 'border-l-red-400 bg-red-50/50 dark:bg-red-950/10',
    warning: 'border-l-yellow-400 bg-yellow-50/50 dark:bg-yellow-950/10',
    info: 'border-l-blue-400 bg-blue-50/50 dark:bg-blue-950/10',
  };

  return (
    <div className="mt-3">
      <h4 className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">
        {title} ({items.length})
      </h4>
      <ul className="mt-1.5 space-y-1.5">
        {items.map((item, i) => (
          <li
            key={`${type}-${i}`}
            className={`border-l-2 ${colorMap[type]} rounded-r-md px-3 py-1.5`}
          >
            <p className="text-sm font-medium text-slate-700 dark:text-slate-300">
              {item.label}
              {item.points !== 0 && (
                <span className="ml-1.5 text-xs text-slate-400">
                  ({item.points > 0 ? '+' : ''}
                  {item.points})
                </span>
              )}
            </p>
            {item.note && (
              <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">{item.note}</p>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}

function ConflictSection({ conflicts }: { conflicts: Contradiction[] }) {
  if (conflicts.length === 0) return null;

  return (
    <div className="mt-3">
      <h4 className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">
        Conflicts ({conflicts.length})
      </h4>
      <ul className="mt-1.5 space-y-2">
        {conflicts.map((c, i) => (
          <li
            key={i}
            className="rounded-md border border-red-200 bg-red-50/50 p-2.5 dark:border-red-800 dark:bg-red-950/10"
          >
            <div className="flex items-start gap-2">
              <span className="text-red-500" aria-hidden="true">
                !
              </span>
              <div className="min-w-0 flex-1">
                <p className="text-xs text-slate-500 dark:text-slate-400">{c.sourceA}</p>
                <p className="text-sm text-slate-700 dark:text-slate-300">{c.claimA}</p>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">vs. {c.sourceB}</p>
                <p className="text-sm text-slate-700 dark:text-slate-300">{c.claimB}</p>
                <p className="mt-1 text-xs italic text-slate-500 dark:text-slate-400">
                  {c.whyTheyDisagree}
                </p>
              </div>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}

function TimelineSection({ steps }: { steps: ReasoningStep[] }) {
  if (!steps || steps.length === 0) return null;

  return (
    <div className="mt-3">
      <h4 className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">
        Timeline
      </h4>
      <ol className="mt-1.5 space-y-2 border-l-2 border-slate-200 pl-3 dark:border-slate-700">
        {(steps ?? []).map((step, i) => (
          <li key={i} className="relative">
            <span className="absolute -left-[1.35rem] top-1 h-2 w-2 rounded-full bg-cyan-500" />
            <p className="text-sm font-medium text-slate-700 dark:text-slate-300">{step.title}</p>
            <p className="text-xs text-slate-500 dark:text-slate-400">{step.summary}</p>
            {(step.details ?? []).length > 0 && (
              <ul className="mt-1 space-y-0.5">
                {(step.details ?? []).map((d, j) => (
                  <li key={j} className="text-xs text-slate-500 dark:text-slate-400">
                    {d}
                  </li>
                ))}
              </ul>
            )}
          </li>
        ))}
      </ol>
    </div>
  );
}

function MissingSection({ items }: { items: MissingInfo[] }) {
  if (items.length === 0) return null;

  return (
    <div className="mt-3">
      <h4 className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">
        Missing Information
      </h4>
      <ul className="mt-1.5 space-y-1.5">
        {items.map((item, i) => (
          <li key={i} className="text-sm text-slate-600 dark:text-slate-400">
            <span className="font-medium">{item.item}:</span> {item.whyItMatters}
          </li>
        ))}
      </ul>
    </div>
  );
}

function RecommendationsSection({ items }: { items: Recommendation[] }) {
  if (items.length === 0) return null;

  return (
    <div className="mt-3">
      <h4 className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">
        Recommendations
      </h4>
      <ul className="mt-1.5 space-y-1.5">
        {items.map((item, i) => (
          <li key={i} className="text-sm text-slate-600 dark:text-slate-400">
            <span className="font-medium">{item.title}:</span> {item.description}
          </li>
        ))}
      </ul>
    </div>
  );
}

function ClaimCard({ claim }: { claim: Claim }) {
  const [open, setOpen] = useState(false);
  const status = claim.status ?? 'no_reliable_evidence';
  const meta = STATUS_META[status] ?? STATUS_META.no_reliable_evidence;

  const supporting = (claim.evidence ?? []).filter((e) => e.result === 'pass');
  const contradicting = (claim.evidence ?? []).filter(
    (e) => e.result === 'fail' || e.result === 'warning',
  );

  return (
    <div
      className={`rounded-xl border ${meta.bgColor} overflow-hidden transition-all duration-200`}
    >
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-3 px-4 py-3 text-left transition hover:bg-white/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 dark:hover:bg-white/5"
      >
        <ClaimStatusBadge status={status} />
        <span className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-slate-800 dark:text-slate-200">
            &ldquo;{claim.text}&rdquo;
          </p>
        </span>
        {claim.confidence !== undefined && (
          <span className="shrink-0 text-xs tabular-nums text-slate-500 dark:text-slate-400">
            {claim.confidence}%
          </span>
        )}
        <span
          aria-hidden="true"
          className={`shrink-0 text-slate-400 transition-transform dark:text-slate-500 ${
            open ? 'rotate-180' : ''
          }`}
        >
          ▾
        </span>
      </button>

      {open && (
        <div className="border-t border-slate-200/50 px-4 py-4 dark:border-slate-700/50">
          {/* Summary */}
          {claim.summary && (
            <p className="text-sm leading-relaxed text-slate-600 dark:text-slate-400">
              {claim.summary}
            </p>
          )}

          {/* Confidence bar */}
          {claim.confidence !== undefined && (
            <div className="mt-3">
              <div className="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400">
                <span>Confidence</span>
                <span className="tabular-nums">{claim.confidence}%</span>
              </div>
              <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700">
                <div
                  className="h-full rounded-full bg-cyan-500 transition-all duration-500"
                  style={{ width: `${claim.confidence}%` }}
                />
              </div>
            </div>
          )}

          {/* Evidence */}
          <EvidenceSection title="Supporting Evidence" items={supporting} type="pass" />
          <EvidenceSection title="Contradicting Evidence" items={contradicting} type="fail" />

          {/* Conflicts */}
          <ConflictSection conflicts={claim.conflicts ?? []} />

          {/* Timeline */}
          <TimelineSection steps={claim.timeline ?? []} />

          {/* Missing */}
          <MissingSection items={claim.missingInformation ?? []} />

          {/* Recommendations */}
          <RecommendationsSection items={claim.recommendations ?? []} />

          {/* Keywords */}
          {claim.keywords && claim.keywords.length > 0 && (
            <div className="mt-3">
              <h4 className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">
                Keywords
              </h4>
              <div className="mt-1 flex flex-wrap gap-1">
                {claim.keywords.map((k) => (
                  <span
                    key={k}
                    className="rounded-md bg-slate-100 px-1.5 py-0.5 text-xs text-slate-600 dark:bg-slate-800 dark:text-slate-400"
                  >
                    {k}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default memo(ClaimCard);
