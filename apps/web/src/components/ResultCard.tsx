'use client';

import { memo } from 'react';
import type { VerifyResponse } from '../types';
import AISummary from './AISummary';
import ClaimsList from './ClaimsList';
import CollapsibleSection from './CollapsibleSection';
import EvidenceLedger from './EvidenceLedger';
import ScoreExplanation from './ScoreExplanation';
import ConfidenceBreakdown from './ConfidenceBreakdown';
import ContradictingEvidence from './ContradictingEvidence';
import EvidenceList from './EvidenceList';
import EvidenceSections from './EvidenceSections';
import ExportMenu from './ExportMenu';
import ImageEvidence from './ImageEvidence';
import InterpretationsList from './InterpretationsList';
import MainClaimSection from './MainClaimSection';
import MissingInformation from './MissingInformation';
import ReasoningList from './ReasoningList';
import RecommendationsList from './RecommendationsList';
import ReasoningTimeline from './ReasoningTimeline';
import StatusBadge from './StatusBadge';
import SuggestedReading from './SuggestedReading';
import SecurityFindings from './SecurityFindings';
import SupportingEvidence from './SupportingEvidence';
import TrustScore from './TrustScore';
import TypeIcon, { typeLabel } from './TypeIcon';
import WarningSignals from './WarningSignals';
import WhatChanged from './WhatChanged';

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
        <StatusBadge status={result.status} verdict={result.verdict} />
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
          {/* Entity Identity - shown prominently at the top */}
          {result.entityIdentity && (
            <section
              aria-label="Entity identification"
              className="mt-3 rounded-xl border border-slate-200 bg-gradient-to-br from-slate-50 to-white p-4 dark:border-slate-800 dark:from-slate-900 dark:to-slate-900"
            >
              <div className="flex items-start gap-3">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-cyan-100 text-lg dark:bg-cyan-900/30">
                  {result.entityIdentity.entityType === 'government'
                    ? '🏛️'
                    : result.entityIdentity.entityType === 'place'
                      ? '📍'
                      : result.entityIdentity.entityType === 'company'
                        ? '🏢'
                        : result.entityIdentity.entityType === 'person'
                          ? '👤'
                          : '🔍'}
                </div>
                <div className="min-w-0 flex-1">
                  <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100">
                    {result.entityIdentity.canonicalName}
                  </h3>
                  <div className="mt-1 flex flex-wrap items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
                    {result.entityIdentity.country && (
                      <span className="inline-flex items-center gap-1">
                        📍 {result.entityIdentity.country}
                      </span>
                    )}
                    <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-300">
                      {result.entityIdentity.entityType}
                    </span>
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                        result.entityIdentity.identityConfidence === 'high'
                          ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
                          : result.entityIdentity.identityConfidence === 'medium'
                            ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300'
                            : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300'
                      }`}
                    >
                      {' '}
                      Identity: {result.entityIdentity.identityConfidence}
                    </span>
                  </div>
                </div>
              </div>
              {/* What is this? */}
              {result.entityIdentity.description && (
                <div className="mt-3 border-t border-slate-200 pt-3 dark:border-slate-700">
                  <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                    What is this?
                  </h4>
                  <p className="mt-1 text-sm leading-relaxed text-slate-700 dark:text-slate-300">
                    {result.entityIdentity.description}
                  </p>
                </div>
              )}
              {/* Possible alternatives */}
              {result.entityIdentity.possibleAlternatives &&
                result.entityIdentity.possibleAlternatives.length > 0 && (
                  <div className="mt-3 border-t border-slate-200 pt-3 dark:border-slate-700">
                    <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                      Possible alternatives
                    </h4>
                    <ul className="mt-1 space-y-1">
                      {result.entityIdentity.possibleAlternatives.map((alt, i) => (
                        <li
                          key={i}
                          className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400"
                        >
                          <span className="text-slate-400 dark:text-slate-500">•</span>
                          {alt}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
            </section>
          )}
          {/* Reasoning timeline — first section, so the path from claim to
              verdict is visible before the detailed breakdown. */}
          <div className="mt-3">
            <ReasoningTimeline steps={result.timeline} />
          </div>
          {/* Overall assessment */}
          <section
            aria-label="Overall assessment"
            className="mt-3 rounded-xl border border-slate-200 p-4 dark:border-slate-800"
          >
            <div className="flex flex-wrap items-center gap-4">
              <TrustScore score={result.trustScore} />
              <div className="min-w-0 flex-1">
                <p className="mt-2 text-sm leading-relaxed text-slate-700 dark:text-slate-300">
                  {result.summary}
                </p>{' '}
                {result.confidence !== undefined && (
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Analysis confidence: {result.confidence}%
                  </p>
                )}
                {result.claimCount !== undefined && result.claimCount > 0 && (
                  <div className="mt-2 flex flex-wrap gap-2 text-xs">
                    <span className="rounded-full bg-green-100 px-2 py-0.5 text-green-700 dark:bg-green-900/30 dark:text-green-300">
                      {result.verifiedClaims ?? 0} verified
                    </span>
                    <span className="rounded-full bg-yellow-100 px-2 py-0.5 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300">
                      {result.partialClaims ?? 0} partial
                    </span>
                    <span className="rounded-full bg-red-100 px-2 py-0.5 text-red-700 dark:bg-red-900/30 dark:text-red-300">
                      {result.unverifiedClaims ?? 0} unverified
                    </span>
                  </div>
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
          {/* Multi-perspective fact analysis, in the spec order. */}
          <div className="mt-3 space-y-3">
            <CollapsibleSection title="AI Summary" defaultOpen>
              <AISummary summary={result.aiSummary} />
            </CollapsibleSection>

            <CollapsibleSection title="Confidence Breakdown" defaultOpen>
              <ConfidenceBreakdown breakdown={result.confidenceBreakdown} />
            </CollapsibleSection>

            <CollapsibleSection
              title="Multiple Interpretations"
              badge={String(result.interpretations?.length ?? 0)}
              defaultOpen
            >
              <InterpretationsList interpretations={result.interpretations} />
            </CollapsibleSection>

            <CollapsibleSection
              title="Supporting Evidence"
              badge={
                result.supportingEvidence?.length
                  ? String(result.supportingEvidence.length)
                  : undefined
              }
              defaultOpen
            >
              <SupportingEvidence groups={result.supportingEvidence} />
            </CollapsibleSection>

            <CollapsibleSection
              title="Contradicting Evidence"
              badge={
                result.contradictingEvidence?.length
                  ? String(result.contradictingEvidence.length)
                  : undefined
              }
            >
              <ContradictingEvidence contradictions={result.contradictingEvidence} />
            </CollapsibleSection>

            <CollapsibleSection title="Missing Information">
              <MissingInformation items={result.missingInformation} />
            </CollapsibleSection>

            <CollapsibleSection title="What Changed?">
              <WhatChanged events={result.whatChanged} note={result.whatChangedNote} />
            </CollapsibleSection>

            <CollapsibleSection title="Suggested Reading">
              <SuggestedReading
                items={result.suggestedReading}
                note={result.suggestedReadingNote}
              />
            </CollapsibleSection>
          </div>{' '}
          {/* Security Findings - Security Intelligence Engine */}
          {result.securityReport && (
            <div className="mt-3">
              <CollapsibleSection title="Security Analysis" defaultOpen>
                <SecurityFindings report={result.securityReport} />
              </CollapsibleSection>
            </div>
          )}
          {/* Score Explanation - Evidence Depth */}
          {result.scoreExplanation && (
            <div className="mt-3">
              <CollapsibleSection title="Score Explanation" defaultOpen>
                <ScoreExplanation explanation={result.scoreExplanation} />
              </CollapsibleSection>
            </div>
          )}
          {/* Evidence Ledger - Evidence Depth */}
          {result.evidenceLedger && (
            <div className="mt-3">
              <CollapsibleSection title="Evidence Ledger" defaultOpen>
                <EvidenceLedger ledger={result.evidenceLedger} />
              </CollapsibleSection>
            </div>
          )}
          {/* Image Evidence - Visual evidence from uploaded images */}
          {result.evidenceFor?.some((e) => e.imageType) && (
            <div className="mt-3">
              <CollapsibleSection title="Image Evidence" defaultOpen>
                <div className="space-y-3">
                  {result.evidenceFor
                    ?.filter((e) => e.imageType)
                    .map((evidence, index) => (
                      <ImageEvidence key={index} evidence={evidence} />
                    ))}
                </div>
              </CollapsibleSection>
            </div>
          )}
          {/* Phase 13: extracted claims */}
          {result.claims && result.claims.length > 0 && (
            <div className="mt-3">
              <CollapsibleSection
                title="Verified Claims"
                badge={result.claimCount ? String(result.claimCount) : undefined}
                defaultOpen
              >
                <ClaimsList claims={result.claims} />
              </CollapsibleSection>
            </div>
          )}
          {/* Legacy evidence detail, kept for full transparency. */}
          <div className="mt-3">
            <CollapsibleSection
              title="Evidence Detail"
              badge={evidenceCount ? String(evidenceCount) : undefined}
            >
              <EvidenceSections
                evidenceFor={result.evidenceFor}
                evidenceAgainst={result.evidenceAgainst}
                missingEvidence={result.missingEvidence}
                unknownInformation={result.unknownInformation}
              />
            </CollapsibleSection>
          </div>
          <div className="mt-3 space-y-3">
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
