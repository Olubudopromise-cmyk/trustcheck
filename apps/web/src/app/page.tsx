'use client';

import { useCallback, useState } from 'react';
import AnalyticsDashboard from '../components/AnalyticsDashboard';
import BatchInput from '../components/BatchInput';
import BatchResults from '../components/BatchResults';
import EmptyState from '../components/EmptyState';
import ErrorCard from '../components/ErrorCard';
import ExampleChips from '../components/ExampleChips';
import HistoryList from '../components/HistoryList';
import LoadingSpinner from '../components/LoadingSpinner';
import ResultCard from '../components/ResultCard';
import SearchForm from '../components/SearchForm';
import ThemeToggle from '../components/ThemeToggle';
import { useBatchVerification, parseBatchInput } from '../hooks/useBatchVerification';
import { useVerificationHistory } from '../hooks/useVerificationHistory';
import type { VerificationHistoryItem, VerifyResponse } from '../types';
import { verify } from '../utils/api';

export default function HomePage() {
  const [input, setInput] = useState('');
  const [result, setResult] = useState<VerifyResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [batchText, setBatchText] = useState('');
  const [batchError, setBatchError] = useState<string | null>(null);
  const { history, remember, clear } = useVerificationHistory();
  const batch = useBatchVerification();

  const handleSubmit = useCallback(async () => {
    setResult(null);
    setError(null);
    const trimmed = input.trim();
    if (!trimmed) {
      setError('Please enter something to verify.');
      return;
    }
    setLoading(true);
    try {
      const data = await verify(trimmed);
      setResult(data);
      remember(trimmed, data);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Something went wrong. Is the backend running?');
    } finally {
      setLoading(false);
    }
  }, [input, remember]);

  const handleReopen = useCallback((item: VerificationHistoryItem) => {
    setInput(item.input);
    setResult(item.result);
    setError(null);
  }, []);

  const handleBatchSubmit = () => {
    const inputs = parseBatchInput(batchText);
    if (inputs.length === 0) {
      setBatchError('Please enter at least one input to verify.');
      return;
    }
    setBatchError(null);
    batch.verifyBatch(inputs, (input, data) => remember(input, data));
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

        <section aria-label="Batch Verification" className="mt-8">
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow dark:border-slate-800 dark:bg-slate-900">
            <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
              Batch Verification
            </h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
              Verify up to 100 inputs at once. One input per line.
            </p>
            <div className="mt-4">
              <BatchInput
                value={batchText}
                onChange={setBatchText}
                onSubmit={handleBatchSubmit}
                loading={batch.status === 'verifying'}
              />
            </div>
            {batchError && (
              <p role="alert" className="mt-3 text-sm text-red-600 dark:text-red-400">
                {batchError}
              </p>
            )}
          </div>

          {(batch.status === 'verifying' || batch.results.length > 0) && (
            <div className="mt-4">
              <BatchResults
                status={batch.status}
                progress={batch.progress}
                results={batch.results}
              />
            </div>
          )}
        </section>

        <div className="mt-8">
          <HistoryList history={history} onReopen={handleReopen} onClear={clear} />
        </div>

        <div className="mt-8">
          <AnalyticsDashboard history={history} />
        </div>
      </div>
    </main>
  );
}
