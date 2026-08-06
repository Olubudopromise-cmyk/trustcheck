'use client';

import { memo } from 'react';
import type { VerifyResponse } from '../types';
import EvidenceList from './EvidenceList';
import ExportMenu from './ExportMenu';
import StatusBadge from './StatusBadge';
import TrustScore from './TrustScore';
import TypeIcon, { typeLabel } from './TypeIcon';

function ResultCard({ result }: { result: VerifyResponse }) {
  return (
    <div
      className="animate-fadeIn rounded-xl border border-slate-200 bg-white p-6 shadow dark:border-slate-800 dark:bg-slate-900"
      role="region"
      aria-label="Verification result"
    >
      <div className="mb-4 flex items-center justify-between gap-3">
        <h2 className="min-w-0 truncate text-base font-semibold text-slate-900 dark:text-slate-100">
          Verification Result
        </h2>
        <div className="shrink-0">
          <ExportMenu result={result} />
        </div>
      </div>

      <div className="mb-4 flex items-center justify-between">
        <StatusBadge status={result.status} />
        <div className="flex items-center gap-2">
          <TypeIcon type={result.type} />
          <span className="text-sm font-medium text-slate-900 dark:text-slate-100">
            {typeLabel(result.type)}
          </span>
        </div>
      </div>

      <dl className="grid grid-cols-1 gap-x-4 gap-y-3 text-sm sm:grid-cols-2">
        <div className="sm:col-span-2">
          <dt className="text-slate-500 dark:text-slate-400">Input</dt>
          <dd className="font-mono break-all text-slate-900 dark:text-slate-200">{result.input}</dd>
        </div>
        <div>
          <dt className="text-slate-500 dark:text-slate-400">Type</dt>
          <dd className="text-slate-900 dark:text-slate-200">{typeLabel(result.type)}</dd>
        </div>
        <div>
          <dt className="text-slate-500 dark:text-slate-400">Status</dt>
          <dd className="text-slate-900 dark:text-slate-200">{result.status}</dd>
        </div>
        <div className="flex items-center gap-4 sm:col-span-2">
          <dt className="text-slate-500 dark:text-slate-400">Trust Score</dt>
          <TrustScore score={result.trustScore} />
        </div>
        <div className="sm:col-span-2">
          <dt className="text-slate-500 dark:text-slate-400">Summary</dt>
          <dd className="text-slate-900 dark:text-slate-200">{result.summary}</dd>
        </div>
      </dl>

      <div className="mt-5">
        <EvidenceList evidence={result.evidence ?? []} />
      </div>
    </div>
  );
}

export default memo(ResultCard);
