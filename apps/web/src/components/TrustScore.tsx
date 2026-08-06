'use client';

const scoreColor = (score: number): string =>
  score >= 90
    ? 'text-green-500'
    : score >= 70
      ? 'text-blue-500'
      : score >= 50
        ? 'text-yellow-400'
        : 'text-red-500';

export default function TrustScore({ score }: { score: number }) {
  const pct = Math.max(0, Math.min(100, Math.round(score)));
  const r = 38;
  const C = 2 * Math.PI * r;
  const offset = C * (1 - pct / 100);
  const color = scoreColor(score);

  return (
    <div className="relative flex h-24 w-24 items-center justify-center">
      <svg
        className="h-full w-full"
        viewBox="0 0 100 100"
        role="img"
        aria-label={`trust score ${pct} of 100`}
      >
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
          {pct}
        </text>
      </svg>
    </div>
  );
}
