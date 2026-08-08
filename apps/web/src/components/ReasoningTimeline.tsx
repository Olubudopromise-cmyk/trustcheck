'use client';

import { memo, useState } from 'react';
import type { ReasoningStep } from '../types';

// stepIcon maps each reasoning step to an emoji so the timeline reads like a
// journey from claim to final verdict. Unknown titles fall back to a neutral
// marker so future steps still render sensibly.
const stepIcon: Record<string, string> = {
  'Claim Detected': '📝',
  'Evidence Gathered': '🔍',
  'Conflicts Identified': '⚖️',
  'Risk Signals Detected': '🚩',
  'AI Reasoning': '🧠',
  'Final Assessment': '✅',
};

// ReasoningTimeline renders the six-step reasoning process as an expandable
// vertical timeline. Each step shows a one-line summary and can be expanded
// to inspect its detail lines. It is keyboard accessible and announces its
// expanded state to screen readers.
function ReasoningTimeline({ steps }: { steps?: ReasoningStep[] }) {
  const [openStep, setOpenStep] = useState<number>(0);

  if (!steps?.length) {
    return null;
  }

  return (
    <section
      aria-label="Reasoning timeline"
      className="rounded-xl border border-slate-200 bg-white p-4 shadow dark:border-slate-800 dark:bg-slate-900"
    >
      <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
        How this result was reached
      </h3>
      <ol role="list" className="mt-3 space-y-0">
        {steps.map((step, index) => {
          const open = openStep === index;
          const isLast = index === steps.length - 1;
          return (
            <li key={`${step.title}-${index}`} className="relative flex gap-3 pb-5">
              {!isLast && (
                <span
                  aria-hidden="true"
                  className="absolute left-[21px] top-10 bottom-0 w-0.5 bg-slate-200 dark:bg-slate-700"
                />
              )}
              <span
                aria-hidden="true"
                className="z-10 flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-slate-50 text-lg dark:border-slate-700 dark:bg-slate-800"
              >
                {stepIcon[step.title] ?? '•'}
              </span>
              <div className="min-w-0 flex-1">
                <button
                  type="button"
                  aria-expanded={open}
                  aria-controls={`timeline-details-${index}`}
                  aria-label={`${step.title}: ${step.summary}`}
                  onClick={() => setOpenStep(open ? -1 : index)}
                  className="w-full rounded-lg px-2 py-1 text-left transition hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 dark:hover:bg-slate-800/60"
                >
                  <span className="flex items-center justify-between gap-3">
                    <span className="text-sm font-semibold text-slate-900 dark:text-slate-100">
                      {step.title}
                    </span>
                    <span
                      aria-hidden="true"
                      className={`shrink-0 text-xs font-medium tabular-nums text-slate-400 transition-transform dark:text-slate-500 ${open ? 'rotate-180' : ''}`}
                    >
                      ▾
                    </span>
                  </span>
                  <span className="mt-0.5 block text-sm text-slate-500 dark:text-slate-400">
                    {step.summary}
                  </span>
                </button>
                {open && (
                  <ul
                    id={`timeline-details-${index}`}
                    role="list"
                    className="mt-2 space-y-1.5 border-l border-slate-200 pl-5 dark:border-slate-700"
                  >
                    {(step.details ?? []).map((detail, detailIndex) => (
                      <li
                        key={detailIndex}
                        className="text-sm leading-relaxed text-slate-700 dark:text-slate-300"
                      >
                        {detail}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </li>
          );
        })}
      </ol>
    </section>
  );
}

export default memo(ReasoningTimeline);
