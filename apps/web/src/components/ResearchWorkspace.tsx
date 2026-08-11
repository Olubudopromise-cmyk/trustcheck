'use client';

import { memo } from 'react';
import type { VerifyResponse } from '../types';
import EmptyState from './EmptyState';
import ErrorBoundary from './ErrorBoundary';
import ErrorCard from './ErrorCard';
import LoadingSpinner from './LoadingSpinner';
import ResultCard from './ResultCard';

interface ResearchWorkspaceProps {
  result: VerifyResponse | null;
  loading: boolean;
  error: string | null;
  isNewResearch: boolean;
}

function ResearchWorkspace({ result, loading, error, isNewResearch }: ResearchWorkspaceProps) {
  const showResult = !loading && !error && result;
  const showEmpty = !loading && !error && !result;

  return (
    <main className="flex-1 overflow-y-auto">
      <div className="mx-auto max-w-4xl px-4 py-6">
        {error && <ErrorCard message={error} />}
        {loading && <LoadingSpinner />}
        {showResult && (
          <div
            className={`transition-all duration-300 ease-out ${
              isNewResearch ? 'animate-slideUp' : ''
            }`}
          >
            <ErrorBoundary>
              <ResultCard result={result!} />
            </ErrorBoundary>
          </div>
        )}
        {showEmpty && <EmptyState />}
      </div>
    </main>
  );
}

export default memo(ResearchWorkspace);
