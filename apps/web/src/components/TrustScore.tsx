'use client';

import { useEffect, useRef, useState } from 'react';

const ANIMATION_DURATION_MS = 800;

const scoreColor = (score: number): string =>
  score >= 90
    ? 'text-green-500'
    : score >= 70
      ? 'text-blue-500'
      : score >= 50
        ? 'text-yellow-400'
        : 'text-red-500';

export default function TrustScore({
  score,
  compact = false,
}: {
  score: number;
  compact?: boolean;
}) {
  const target = Math.max(0, Math.min(100, Math.round(score)));
  const [display, setDisplay] = useState(0);
  const rafRef = useRef<number>(0);

  useEffect(() => {
    if (compact) {
      return;
    }
    const start = performance.now();
    const tick = (now: number) => {
      const progress = Math.min(1, (now - start) / ANIMATION_DURATION_MS);
      const eased = 1 - Math.pow(1 - progress, 3);
      setDisplay(Math.round(eased * target));
      if (progress < 1) {
        rafRef.current = requestAnimationFrame(tick);
      }
    };
    rafRef.current = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(rafRef.current);
  }, [compact, target]);

  const r = 38;
  const C = 2 * Math.PI * r;
  const offset = C * (1 - display / 100);
  const color = scoreColor(target);

  if (compact) {
    return (
      <span
        className={`font-semibold tabular-nums ${color}`}
        aria-label={`Trust score ${target} of 100`}
      >
        {target}
        <span className="sr-only"> / 100</span>
      </span>
    );
  }

  return (
    <div
      className="relative flex h-24 w-24 items-center justify-center"
      aria-live="polite"
      aria-atomic="true"
    >
      <span className="sr-only">Trust score {target} of 100</span>
      <svg className="h-full w-full" viewBox="0 0 100 100" aria-hidden="true">
        <circle
          cx={50}
          cy={50}
          r={r}
          strokeWidth={12}
          fill="none"
          stroke="currentColor"
          className="text-slate-200 dark:text-slate-700"
          strokeDasharray={C}
          transform="rotate(-90 50 50)"
        />
        <circle
          cx={50}
          cy={50}
          r={r}
          strokeWidth={12}
          fill="none"
          stroke="currentColor"
          className={color}
          strokeDasharray={C}
          strokeDashoffset={offset}
          strokeLinecap="round"
          transform="rotate(-90 50 50)"
        />
        <text x={50} y={56} textAnchor="middle" className={`text-3xl font-bold ${color}`}>
          {display}
        </text>
      </svg>
    </div>
  );
}
