'use client';

import { memo } from 'react';
import type { VerifyResponse } from '../types';
import CollapsibleSection from './CollapsibleSection';
import EvidenceList from './EvidenceList';
import EvidenceSections from './EvidenceSections';
import ExportMenu from './ExportMenu';
import InterpretationsList from './InterpretationsList';
import MainClaimSection from './MainClaimSection';
import ReasoningList from './ReasoningList';
import RecommendationsList from './RecommendationsList';
import ReasoningTimeline from './ReasoningTimeline';
import StatusBadge from './StatusBadge';
import TrustScore from './TrustScore';
import TypeIcon, { typeLabel } from './TypeIcon';
import WarningSignals from './WarningSignals';

// verdictMeta styles the High/Medium/Low verdict badge.
const verdictMeta: Record<string, { label: string; color: string }> = {
  High: {
    label: 'HIGH TRUST',
    color: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
  },
  Medium: {
    label: 'MEDIUM TRUST',
    color: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
  },
  Low: {
    label: 'LOW TRUST',
    color: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
  },
};

// LegacyResult renders the original single-card layout for results that were
// saved to local history before the explainable analysis shipped.
function LegacyResult({ result }: { result: VerifyResponse }) {
  return (
    <>
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
    </>
  );
}

function ResultCard({ result }: { result: VerifyResponse }) {
  const hasAnalysis = result.verdict !== undefined;

  const verdict = result.verdict ? verdictMeta[result.verdict] : undefined;
  const evidenceCount = (result.evidenceFor?.length ?? 0) + (result.evidenceAgainst?.length ?? 0);

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

      {!hasAnalysis ? (
        <LegacyResult result={result} />
      ) : (
        <>
          {/* Reasoning timeline — first section, so the path from claim to
              verdict is visible before the detailed breakdown. */}
          <div className="mt-3">
            <ReasoningTimeline steps={result.timeline} />
          </div>

          {/* Overall assessment */}
          <section
            aria-label="Overall assessment"
            className="rounded-xl border border-slate-200 p-4 dark:border-slate-800"
          >
            <div className="flex flex-wrap items-center gap-4">
              <TrustScore score={result.trustScore} />
              <div className="min-w-0 flex-1">
                {verdict && (
                  <span
                    className={`inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold ${verdict.color}`}
                  >
                    {verdict.label}
                  </span>
                )}
                <p className="mt-2 text-sm leading-relaxed text-slate-700 dark:text-slate-300">
                  {result.summary}
                </p>
                {result.confidence !== undefined && (
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Analysis confidence: {result.confidence}%
                  </p>
                )}
              </div>
            </div>
          </section>

          <div className="mt-3">
            <MainClaimSection
              claim={result.keyClaim}
              entities={result.entities}
              keywords={result.keywords}
            />
          </div>

          <div className="mt-3 space-y-3">
            <CollapsibleSection
              title="Evidence"
              badge={evidenceCount ? String(evidenceCount) : undefined}
              defaultOpen
            >
              <EvidenceSections
                evidenceFor={result.evidenceFor}
                evidenceAgainst={result.evidenceAgainst}
                missingEvidence={result.missingEvidence}
                unknownInformation={result.unknownInformation}
              />
            </CollapsibleSection>

            <CollapsibleSection
              title="Possible Interpretations"
              badge={String(result.interpretations?.length ?? 0)}
              defaultOpen
            >
              <InterpretationsList interpretations={result.interpretations} />
            </CollapsibleSection>

            <CollapsibleSection title="Warning Signals">
              <WarningSignals signals={result.warningSignals} />
            </CollapsibleSection>

            <CollapsibleSection title="Recommendations" defaultOpen>
              <RecommendationsList recommendations={result.recommendations} />
            </CollapsibleSection>

            <CollapsibleSection title="Raw AI Reasoning">
              <ReasoningList reasoning={result.reasoning} />
            </CollapsibleSection>
          </div>
        </>
      )}
    </div>
  );
}

export default memo(ResultCard);
