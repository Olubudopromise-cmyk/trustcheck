'use client';

import { useState } from 'react';
import EmptyState from '../components/EmptyState';
import ErrorCard from '../components/ErrorCard';
import ExampleChips from '../components/ExampleChips';
import LoadingSpinner from '../components/LoadingSpinner';
import ResultCard from '../components/ResultCard';
import SearchForm from '../components/SearchForm';
import ThemeToggle from '../components/ThemeToggle';
import type { VerifyResponse } from '../types';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

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
      const res = await fetch(`${API_URL}/verify`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ input: trimmed }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.error || `Verification failed (HTTP ${res.status}).`);
      }
      setResult((await res.json()) as VerifyResponse);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Something went wrong. Is the backend running?');
    } finally {
      setLoading(false);
    }
  };

  const showResult = !loading && !error && result;
  const showEmpty = !loading && !error && !result;

  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-8 bg-slate-50 px-6 py-16 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
      <div className="w-full max-w-2xl">
        <header className="relative text-center">
          <p className="text-xs font-semibold uppercase tracking-[0.35em] text-cyan-500 dark:text-cyan-400">
            TrustCheck
          </p>
          <h1 className="mt-3 text-4xl font-semibold sm:text-5xl">
            <span className="bg-gradient-to-r from-cyan-500 to-indigo-500 bg-clip-text text-transparent dark:from-cyan-400 dark:to-indigo-300">
              Verify Anything. Trust Everything.
            </span>
          </h1>
          <p className="mt-4 text-lg text-slate-500 dark:text-slate-400">
            One place to sanity-check domains, emails, IPs, and businesses before you trust them.
          </p>
          <div className="absolute top-0 right-0">
            <ThemeToggle />
          </div>
        </header>

        <SearchForm value={input} onChange={setInput} onSubmit={handleSubmit} loading={loading} />
        <ExampleChips onSelect={setInput} />

        <div className="mt-6 space-y-4">
          {error && <ErrorCard message={error} />}
          {loading && <LoadingSpinner />}
          {showResult && <ResultCard result={result!} />}
          {showEmpty && <EmptyState />}
        </div>
      </div>
    </main>
  );
}
