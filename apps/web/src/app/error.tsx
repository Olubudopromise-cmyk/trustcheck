'use client';

import Link from 'next/link';

interface GlobalErrorProps {
  reset: () => void;
}

export default function GlobalError({ reset }: GlobalErrorProps) {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-4 bg-slate-50 px-6 py-16 text-center text-slate-900 dark:bg-slate-950 dark:text-slate-100">
      <p className="text-xs font-semibold uppercase tracking-[0.35em] text-cyan-500 dark:text-cyan-400">
        TrustCheck
      </p>
      <h1 className="text-3xl font-semibold text-slate-900 dark:text-slate-100">
        Something went wrong
      </h1>
      <p className="max-w-md text-slate-500 dark:text-slate-400">
        An unexpected error occurred. Try again, or head back to the homepage.
      </p>
      <div className="mt-4 flex flex-wrap justify-center gap-3">
        <button
          type="button"
          onClick={reset}
          className="rounded-xl bg-cyan-500 px-6 py-3 text-sm font-medium text-white shadow-lg shadow-cyan-500/20 transition hover:bg-cyan-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 focus-visible:ring-offset-2 ring-offset-slate-50 dark:ring-offset-slate-950"
        >
          Try again
        </button>
        <Link
          href="/"
          className="rounded-xl border border-slate-300 px-6 py-3 text-sm font-medium text-slate-700 transition hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-800"
        >
          Go home
        </Link>
      </div>
    </main>
  );
}
