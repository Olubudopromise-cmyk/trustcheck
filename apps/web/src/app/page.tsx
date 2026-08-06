export default function HomePage() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center bg-slate-950 px-6 text-center">
      <div className="max-w-2xl rounded-2xl border border-slate-800 bg-slate-900/70 p-10 shadow-2xl">
        <p className="mb-4 text-sm uppercase tracking-[0.3em] text-cyan-400">TrustCheck</p>
        <h1 className="text-4xl font-semibold sm:text-6xl">
          AI-powered verification for the modern web
        </h1>
        <p className="mt-6 text-lg text-slate-300">
          Verify domains, emails, phones, IPs, and businesses from one secure platform.
        </p>
      </div>
    </main>
  );
}
