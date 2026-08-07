'use client';

import { memo } from 'react';

// AISummary presents the AI-generated summary paragraph. It is explicitly
// labeled as AI-generated so verified facts are never confused with a
// synthesized summary, per the honesty rules.
function AISummary({ summary }: { summary?: string }) {
  if (!summary) {
    return null;
  }

  return (
    <section aria-label="AI summary">
      <p className="text-sm leading-relaxed text-slate-700 dark:text-slate-300">{summary}</p>
      <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
        AI-generated summary of the sections above; it does not add facts beyond what those sections
        report.
      </p>
    </section>
  );
}

export default memo(AISummary);
