'use client';

import { useCallback, useState } from 'react';
import ExampleChips from '../components/ExampleChips';
import ResearchSidebar from '../components/ResearchSidebar';
import ResearchWorkspace from '../components/ResearchWorkspace';
import ThemeToggle from '../components/ThemeToggle';
import WorkspaceComposer from '../components/WorkspaceComposer';
import { useVerificationHistory } from '../hooks/useVerificationHistory';
import type { AnalysisMode, VerificationHistoryItem, VerifyResponse } from '../types';
import { verify } from '../utils/api';
import { normalizeAnalysisMode, normalizeVerifyResponse } from '../utils/history';

export default function HomePage() {
  const [input, setInput] = useState('');
  const [result, setResult] = useState<VerifyResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [isNewResearch, setIsNewResearch] = useState(true);
  const [mode, setMode] = useState<AnalysisMode>('quick');
  const { history, remember, remove } = useVerificationHistory();

  const handleSubmit = useCallback(async () => {
    setResult(null);
    setError(null);
    setIsNewResearch(true);
    const trimmed = input.trim();
    if (!trimmed) {
      setError('Please enter something to verify.');
      return;
    }
    setLoading(true);
    try {
      const data = await verify(trimmed, mode);
      // Normalize the fresh API response (Go emits `null` for empty slices)
      // so every render path — fresh results and persisted history — sees the
      // same safe shape.
      const normalized = normalizeVerifyResponse(data);
      setResult(normalized);
      const item = remember(trimmed, normalized);
      setActiveId(item?.id ?? null);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Something went wrong. Is the backend running?');
    } finally {
      setLoading(false);
    }
  }, [input, remember, mode]);

  const handleReopen = useCallback((item: VerificationHistoryItem) => {
    setInput(item.input);
    // Normalize defensively: history entries saved before newer result fields
    // existed must render and continue without crashing, and the stored mode
    // (if any) is restored so "Continue Research" re-runs the same kind of
    // analysis.
    setResult(normalizeVerifyResponse(item.result));
    const storedMode = normalizeAnalysisMode(item.result.analysisMode);
    if (storedMode) {
      setMode(storedMode);
    }
    setActiveId(item.id);
    setError(null);
    setIsNewResearch(false);
    setSidebarOpen(false);
  }, []);

  const handleNewResearch = useCallback(() => {
    setInput('');
    setResult(null);
    setActiveId(null);
    setError(null);
    setIsNewResearch(true);
    setSidebarOpen(false);
    // Focus the composer after a brief delay
    setTimeout(() => {
      const textarea = document.querySelector(
        '[aria-label="Research input"]',
      ) as HTMLTextAreaElement;
      textarea?.focus();
    }, 100);
  }, []);

  const handleDelete = useCallback(
    (id: string) => {
      remove(id);
      if (activeId === id) {
        setResult(null);
        setActiveId(null);
        setIsNewResearch(true);
      }
    },
    [activeId, remove],
  );

  return (
    <div className="flex h-screen overflow-hidden bg-slate-50 dark:bg-slate-950">
      {/* Sidebar */}
      <ResearchSidebar
        history={history}
        activeId={activeId}
        onSelect={handleReopen}
        onNewResearch={handleNewResearch}
        onDelete={handleDelete}
        isOpen={sidebarOpen}
        onClose={() => setSidebarOpen(false)}
      />

      {/* Main content area */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Top bar */}
        <header className="flex items-center justify-between border-b border-slate-200 bg-white px-4 py-3 dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center gap-3">
            {/* Mobile menu button */}
            <button
              type="button"
              onClick={() => setSidebarOpen(true)}
              className="rounded-lg p-2 text-slate-400 hover:bg-slate-100 hover:text-slate-600 lg:hidden dark:hover:bg-slate-800 dark:hover:text-slate-300"
              aria-label="Open sidebar"
            >
              <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 6h16M4 12h16M4 18h16"
                />
              </svg>
            </button>
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.25em] text-cyan-500 dark:text-cyan-400">
                TrustCheck
              </p>
              <h1 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
                {activeId
                  ? history.find((h) => h.id === activeId)?.input.slice(0, 40) + '...'
                  : 'Verify Anything'}
              </h1>
            </div>
          </div>
          <ThemeToggle />
        </header>
        {/* Workspace */}
        <ResearchWorkspace
          result={result}
          loading={loading}
          error={error}
          isNewResearch={isNewResearch}
        />
        {/* Example chips - only show when no result */}
        {!result && !loading && !error && (
          <div
            className="fixed bottom-24 left-1/2 -translate-x-1/2 px-4 lg:left-auto lg:translate-x-0 lg:pr-4"
            style={{ left: 'min(50%, calc(50% + 144px))' }}
          >
            <ExampleChips onSelect={setInput} />
          </div>
        )}
        {/* Composer */}{' '}
        <WorkspaceComposer
          value={input}
          onChange={setInput}
          onSubmit={handleSubmit}
          loading={loading}
          mode={mode}
          onModeChange={setMode}
        />
        {/* Batch verification section - hidden in workspace mode, accessible via sidebar */}
        {/* Keeping batch functionality accessible but not prominent in workspace layout */}
      </div>
    </div>
  );
}
