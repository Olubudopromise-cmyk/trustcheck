'use client';

import { memo, useMemo, useState } from 'react';
import type { Claim, ClaimStatus } from '../types';
import ClaimCard from './ClaimCard';

type Filter = 'all' | ClaimStatus;

const FILTERS: { key: Filter; label: string; color: string }[] = [
  {
    key: 'all',
    label: 'All',
    color: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300',
  },
  {
    key: 'verified',
    label: 'Verified',
    color: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300',
  },
  {
    key: 'partially_verified',
    label: 'Partial',
    color: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300',
  },
  {
    key: 'unverified',
    label: 'Unverified',
    color: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300',
  },
  {
    key: 'no_reliable_evidence',
    label: 'No Evidence',
    color: 'bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400',
  },
];

function ClaimsList({ claims }: { claims: Claim[] }) {
  const [filter, setFilter] = useState<Filter>('all');

  const counts = useMemo(() => {
    const c = {
      all: claims.length,
      verified: 0,
      partially_verified: 0,
      unverified: 0,
      no_reliable_evidence: 0,
    };
    for (const claim of claims) {
      const s = claim.status ?? 'no_reliable_evidence';
      if (s in c) c[s as keyof typeof c]++;
    }
    return c;
  }, [claims]);

  const filtered = useMemo(() => {
    if (filter === 'all') return claims;
    return claims.filter((c) => (c.status ?? 'no_reliable_evidence') === filter);
  }, [claims, filter]);

  if (claims.length === 0) return null;

  return (
    <section aria-label="Extracted claims">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
          Extracted Claims
        </h3>
        <span className="text-xs text-slate-500 dark:text-slate-400">
          {claims.length} claim{claims.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Filter bar */}
      <div className="mb-3 flex flex-wrap gap-1.5" role="group" aria-label="Filter claims">
        {FILTERS.map((f) => (
          <button
            key={f.key}
            type="button"
            onClick={() => setFilter(f.key)}
            className={`inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium transition ${
              filter === f.key
                ? 'ring-2 ring-cyan-500 ring-offset-1 dark:ring-offset-slate-900 ' + f.color
                : 'bg-slate-50 text-slate-500 hover:bg-slate-100 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700'
            }`}
            aria-pressed={filter === f.key}
          >
            {f.label}
            <span className="tabular-nums opacity-70">{counts[f.key]}</span>
          </button>
        ))}
      </div>

      {/* Claim cards */}
      <div className="space-y-2">
        {filtered.map((claim) => (
          <ClaimCard key={claim.id} claim={claim} />
        ))}
      </div>

      {filtered.length === 0 && (
        <p className="py-4 text-center text-sm text-slate-500 dark:text-slate-400">
          No claims match the selected filter.
        </p>
      )}
    </section>
  );
}

export default memo(ClaimsList);
