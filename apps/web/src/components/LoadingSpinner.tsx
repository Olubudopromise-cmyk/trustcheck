'use client';

export default function LoadingSpinner({ label = 'Verifying…' }: { label?: string }) {
  return (
    <div
      className="flex items-center justify-center gap-3 rounded-xl border border-slate-200 bg-slate-50/60 py-10 text-slate-600 dark:border-slate-800 dark:bg-slate-800/60 dark:text-slate-300"
      role="status"
    >
      <svg className="h-6 w-6 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <circle
          className="opacity-25"
          cx={12}
          cy={12}
          r={10}
          stroke="currentColor"
          strokeWidth={4}
        />
        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
      </svg>
      <span className="font-medium">{label}</span>
    </div>
  );
}
