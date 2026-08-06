'use client';

export default function EmptyState() {
  return (
    <div
      className="rounded-xl border border-slate-200 bg-slate-50/60 py-12 text-center dark:border-slate-800 dark:bg-slate-800/60"
      aria-live="polite"
    >
      <p className="text-2xl font-medium text-slate-900 dark:text-slate-100">
        Nothing verified yet.
      </p>
      <p className="mt-2 text-slate-500 dark:text-slate-400">
        Enter a website, email, IP address or company.
      </p>
    </div>
  );
}
