'use client';

import { memo } from 'react';
import type { ScoreExplanation as ScoreExplanationType } from '../types';

interface ScoreExplanationProps {
  explanation: ScoreExplanationType;
}

function ScoreBar({
  label,
  value,
  note,
  color,
}: {
  label: string;
  value: number;
  note: string;
  color: string;
}) {
  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-slate-700 dark:text-slate-300">{label}</span>
        <span className="text-xs tabular-nums text-slate-500 dark:text-slate-400">{value}%</span>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700">
        <div
          className={`h-full rounded-full transition-all duration-500 ${color}`}
          style={{ width: `${value}%` }}
        />
      </div>
      <p className="text-xs text-slate-500 dark:text-slate-400">{note}</p>
    </div>
  );
}

function ScoreExplanation({ explanation }: ScoreExplanationProps) {
  // Safe defaults for persisted explanations that predate the field (old
  // saved results render with zeros instead of undefined percentages).
  const evidenceStrength = explanation.evidenceStrength ?? 0;
  const sourceQuality = explanation.sourceQuality ?? 0;
  const independentConfirmation = explanation.independentConfirmation ?? 0;
  const contradictionRisk = explanation.contradictionRisk ?? 0;
  const missingEvidence = explanation.missingEvidence ?? 0;

  return (
    <section aria-label="Score Explanation" className="space-y-4">
      <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Why This Result?</h3>

      <div className="grid gap-4 sm:grid-cols-2">
        <ScoreBar
          label="Evidence Strength"
          value={evidenceStrength}
          note={explanation.evidenceStrengthNote ?? ''}
          color={
            evidenceStrength >= 70
              ? 'bg-green-500'
              : evidenceStrength >= 40
                ? 'bg-yellow-500'
                : 'bg-red-500'
          }
        />

        <ScoreBar
          label="Source Quality"
          value={sourceQuality}
          note={explanation.sourceQualityNote ?? ''}
          color={
            sourceQuality >= 70
              ? 'bg-green-500'
              : sourceQuality >= 40
                ? 'bg-yellow-500'
                : 'bg-red-500'
          }
        />

        <ScoreBar
          label="Independent Confirmation"
          value={independentConfirmation}
          note={explanation.independentNote ?? ''}
          color={
            independentConfirmation >= 70
              ? 'bg-green-500'
              : independentConfirmation >= 40
                ? 'bg-yellow-500'
                : 'bg-red-500'
          }
        />

        <ScoreBar
          label="Contradiction Risk"
          value={contradictionRisk}
          note={explanation.contradictionNote ?? ''}
          color={
            contradictionRisk <= 30
              ? 'bg-green-500'
              : contradictionRisk <= 60
                ? 'bg-yellow-500'
                : 'bg-red-500'
          }
        />

        <ScoreBar
          label="Missing Evidence"
          value={missingEvidence}
          note={explanation.missingNote ?? ''}
          color={
            missingEvidence <= 30
              ? 'bg-green-500'
              : missingEvidence <= 60
                ? 'bg-yellow-500'
                : 'bg-red-500'
          }
        />
      </div>
    </section>
  );
}

export default memo(ScoreExplanation);
