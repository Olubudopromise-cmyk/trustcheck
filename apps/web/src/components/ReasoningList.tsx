'use client';

import { memo } from 'react';

// ReasoningList presents the raw bullet explanation of why the trust score is
// what it is. "+" bullets support the score, "-" bullets contradict it.
function ReasoningList({ reasoning }: { reasoning?: string[] }) {
  if (!reasoning?.length) {
    return null;
  }

  return (
    <section aria-label="Score reasoning">
      <ul role="list" className="space-y-1.5">
        {reasoning.map((bullet, index) => {
          const isNegative = bullet.startsWith('-');
          const isPositive = bullet.startsWith('+');
          const color = isNegative
            ? 'text-red-600 dark:text-red-400'
            : isPositive
              ? 'text-green-600 dark:text-green-400'
              : 'text-slate-600 dark:text-slate-400';
          return (
            <li
              key={`${bullet}-${index}`}
              className={`text-sm ${isNegative || isPositive ? 'font-medium' : ''} ${color}`}
            >
              {bullet}
            </li>
          );
        })}
      </ul>
    </section>
  );
}

export default memo(ReasoningList);
