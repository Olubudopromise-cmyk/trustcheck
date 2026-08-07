'use client';

import { memo } from 'react';
import type { Contradiction } from '../types';

// ContradictingEvidence presents disagreements found between the submitted
// claim and the observed evidence. When none was found it says so explicitly
// rather than inventing a conflict.
function ContradictingEvidence({ contradictions }: { contradictions?: Contradiction[] }) {
  if (!contradictions?.length) {
    return (
      <p className="text-sm text-slate-500 dark:text-slate-400">
        No contradicting evidence was found in the checks performed.
      </p>
    );
  }

  return (
    <section aria-label="Contradicting evidence" className="space-y-3">
      {contradictions.map((contradiction, index) => (
        <article
          key={index}
          className="rounded-lg border border-red-200 p-3 dark:border-red-900/50"
        >
          <div className="space-y-2 text-sm">
            <div>
              <span className="font-semibold text-slate-900 dark:text-slate-100">
                {contradiction.sourceA}
              </span>
              <p className="text-slate-700 dark:text-slate-300">{contradiction.claimA}</p>
            </div>
            <div>
              <span className="font-semibold text-slate-900 dark:text-slate-100">
                {contradiction.sourceB}
              </span>
              <p className="text-slate-700 dark:text-slate-300">{contradiction.claimB}</p>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {contradiction.whyTheyDisagree}
            </p>
            <p className="text-xs font-medium text-slate-500 dark:text-slate-400">
              Confidence in contradiction: {contradiction.confidenceInContradiction}%
            </p>
          </div>
        </article>
      ))}
    </section>
  );
}

export default memo(ContradictingEvidence);
