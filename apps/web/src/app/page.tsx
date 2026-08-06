'use client';

import { useEffect, useState } from 'react';

type VerifyResponse = {
  input: string;
  type: string;
  status: string;
  trustScore: number;
  summary: string;
};

const EXAMPLE_QUERIES = ['openai.com', 'support@stripe.com', '8.8.8.8', 'Acme Corp LLC'];

function SunIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-4 w-4"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <circle cx={12} cy={12} r={5} />
      <line x1={12} y1={1} x2={12} y2={3} />
      <line x1={12} y1={21} x2={12} y2={23} />
      <line x1={4.22} y1={4.22} x2={5.64} y2={5.64} />
      <line x1={18.36} y1={18.36} x2={19.78} y2={19.78} />
      <line x1={1} y1={12} x2={3} y2={12} />
      <line x1={21} y1={12} x2={23} y2={12} />
      <line x1={4.22} y1={19.78} x2={5.64} y2={18.36} />
      <line x1={18.36} y1={5.64} x2={19.78} y2={4.22} />
    </svg>
  );
}

function MoonIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-4 w-4"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </svg>
  );
}

function ThemeToggle() {
  const [dark, setDark] = useState(false);

  useEffect(() => {
    const saved = localStorage.getItem('theme');
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    const initial = saved === 'dark' || (!saved && prefersDark);
    // eslint-disable-next-line react-hooks/set-state-in-effect -- canonical SSR-safe localStorage init
    setDark(initial);
  }, []);

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark);
  }, [dark]);

  const toggle = () => {
    const next = !dark;
    setDark(next);
    localStorage.setItem('theme', next ? 'dark' : 'light');
  };

  return (
    <button
      type="button"
      onClick={toggle}
      aria-label="Toggle theme"
      className="rounded-lg border border-slate-300 bg-white/70 p-2 text-slate-700 shadow-sm backdrop-blur hover:bg-white dark:border-slate-700 dark:bg-slate-800/50 dark:text-slate-200"
    >
      {dark ? <MoonIcon /> : <SunIcon />}
    </button>
  );
}

export default function HomePage() {
  const [input, setInput] = useState('');
  const [result, setResult] = useState<VerifyResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    setResult(null);
    setError(null);

    const trimmed = input.trim();
    if (!trimmed) {
      setError('Please enter something to verify.');
      return;
    }

    setLoading(true);
    try {
      const res = await fetch('http://localhost:8080/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ input: trimmed }),
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.error || `Request failed with status ${res.status}`);
      }

      const data: VerifyResponse = await res.json();
      setResult(data);
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Something went wrong. Is the backend running?';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') handleSubmit();
  };

  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-8 bg-slate-50 px-6 py-16 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
      <div className="w-full max-w-2xl">
        <header className="relative text-center">
          <p className="text-xs font-semibold uppercase tracking-[0.35em] text-cyan-500 dark:text-cyan-400">
            TrustCheck
          </p>
          <h1 className="mt-3 text-4xl font-semibold sm:text-6xl">
            <span className="bg-gradient-to-r from-cyan-500 to-indigo-500 bg-clip-text text-transparent dark:from-cyan-400 dark:to-indigo-300">
              Verify Anything. Trust Everything.
            </span>
          </h1>
          <p className="mt-4 text-lg text-slate-500 dark:text-slate-400">
            A single place to sanity-check domains, emails, IPs, and businesses before you trust
            them.
          </p>
          <div className="absolute top-0 right-0">
            <ThemeToggle />
          </div>
        </header>

        <form
          onSubmit={(e) => {
            e.preventDefault();
            handleSubmit();
          }}
          className="mt-10 flex flex-col gap-3"
        >
          <input
            type="text"
            placeholder="Enter a URL, email, IP, phone, or company..."
            value={input}
            onChange={(e) => {
              setInput(e.target.value);
              setResult(null);
              setError(null);
            }}
            onKeyDown={onKeyDown}
            disabled={loading}
            className="w-full rounded-xl border border-slate-300 bg-white px-5 py-4 text-base text-slate-900 placeholder-slate-400 outline-none ring-cyan-500 focus:border-cyan-500 focus:ring-2 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:placeholder-slate-500 dark:focus:border-cyan-400 dark:focus:ring-cyan-400"
          />
          <button
            type="submit"
            disabled={loading || !input.trim()}
            className="w-full rounded-xl bg-cyan-500 py-4 text-base font-medium text-white shadow-lg shadow-cyan-500/20 transition-colors hover:bg-cyan-600 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {loading ? 'Verifying…' : 'Verify'}
          </button>
        </form>

        <p className="mt-4 text-center text-xs text-slate-500 dark:text-slate-500">
          Example searches:
          {EXAMPLE_QUERIES.map((q) => (
            <button
              key={q}
              onClick={() => setInput(q)}
              className="ml-1 inline-block cursor-pointer underline transition-opacity hover:opacity-70"
            >
              {q}
            </button>
          ))}
        </p>

        <div className="mt-8">
          {error && (
            <div className="rounded-xl border border-red-200 bg-red-50 p-5 text-red-800 dark:border-red-900/40 dark:bg-red-950/40 dark:text-red-300">
              <p className="font-medium">Error</p>
              <p className="mt-1">{error}</p>
            </div>
          )}

          {loading && (
            <div
              role="status"
              className="rounded-xl border border-slate-200 bg-slate-100/60 p-6 text-center dark:border-slate-800 dark:bg-slate-800"
            >
              <p className="text-slate-600 dark:text-slate-400">Running verification…</p>
            </div>
          )}

          {result && (
            <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900">
              <div className="mb-4 flex items-center justify-between">
                <h2 className="text-lg font-semibold">Verification Result</h2>
                <span className="rounded-full bg-cyan-100 px-3 py-1 text-xs font-medium text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-300">
                  {result.status}
                </span>
              </div>

              <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                <div>
                  <dt className="text-slate-500 dark:text-slate-500">Input</dt>
                  <dd className="mt-1 font-mono text-slate-900 dark:text-slate-200 break-all">
                    {result.input}
                  </dd>
                </div>
                <div>
                  <dt className="text-slate-500 dark:text-slate-500">Type</dt>
                  <dd className="mt-1 text-slate-900 dark:text-slate-200">{result.type}</dd>
                </div>
                <div>
                  <dt className="text-slate-500 dark:text-slate-500">Trust Score</dt>
                  <dd className="mt-1 text-slate-900 dark:text-slate-200">{result.trustScore}</dd>
                </div>
                <div className="sm:col-span-2">
                  <dt className="text-slate-500 dark:text-slate-500">Summary</dt>
                  <dd className="mt-1 text-slate-900 dark:text-slate-200">{result.summary}</dd>
                </div>
              </dl>
            </div>
          )}
        </div>
      </div>
    </main>
  );
}
