import Link from 'next/link';

export default function NotFound() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-4 bg-slate-50 px-6 py-16 text-center text-slate-900 dark:bg-slate-950 dark:text-slate-100">
      <p className="text-xs font-semibold uppercase tracking-[0.35em] text-cyan-500 dark:text-cyan-400">
        TrustCheck
      </p>
      <h1 className="text-5xl font-semibold text-slate-900 dark:text-slate-100">404</h1>
      <p className="max-w-md text-lg text-slate-500 dark:text-slate-400">
        This page doesn&apos;t exist. The link may be broken, or the page may have moved.
      </p>
      <Link
        href="/"
        className="mt-4 rounded-xl bg-cyan-500 px-6 py-3 text-sm font-medium text-white shadow-lg shadow-cyan-500/20 transition hover:bg-cyan-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 focus-visible:ring-offset-2 ring-offset-slate-50 dark:ring-offset-slate-950"
      >
        Back to TrustCheck
      </Link>
    </main>
  );
}
