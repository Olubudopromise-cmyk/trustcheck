'use client';

import { memo } from 'react';
import type { Recommendation } from '../types';

// RecommendationsList presents the actionable next steps for the user.
function RecommendationsList({ recommendations }: { recommendations?: Recommendation[] }) {
  if (!recommendations?.length) {
    return null;
  }

  return (
    <section aria-label="Recommendations">
      <ol role="list" className="space-y-2.5">
        {recommendations.map((recommendation) => (
          <li key={recommendation.title} className="flex items-start gap-3">
            <span
              aria-hidden="true"
              className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-cyan-500 text-xs font-bold text-white"
            >
              ✓
            </span>
            <span className="min-w-0">
              <span className="block text-sm font-medium text-slate-900 dark:text-slate-100">
                {recommendation.title}
              </span>
              <span className="block text-sm text-slate-500 dark:text-slate-400">
                {recommendation.description}
              </span>
            </span>
          </li>
        ))}
      </ol>
    </section>
  );
}

export default memo(RecommendationsList);
