'use client';

import { Component, type ReactNode } from 'react';

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

// ErrorBoundary catches rendering errors in its children and displays a
// controlled fallback UI instead of letting the error propagate to the
// Next.js error page. This prevents a single malformed saved session or
// a rendering bug in one component from taking down the entire application.
export default class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // Log the error for debugging but don't expose internals to the user.
    console.error('[ErrorBoundary] Rendering error:', error.message, errorInfo.componentStack);
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div
          className="rounded-xl border border-amber-200 bg-amber-50 p-5 dark:border-amber-900/40 dark:bg-amber-950/40"
          role="alert"
        >
          <p className="font-medium text-amber-800 dark:text-amber-300">
            This section could not be displayed
          </p>
          <p className="mt-1 text-sm text-amber-700 dark:text-amber-400">
            A rendering error occurred. Your saved research has not been deleted. You can try
            refreshing the page or opening a different session.
          </p>
          <button
            type="button"
            onClick={() => this.setState({ hasError: false, error: null })}
            className="mt-3 rounded-lg bg-amber-100 px-3 py-1.5 text-xs font-medium text-amber-800 transition hover:bg-amber-200 dark:bg-amber-900/50 dark:text-amber-300 dark:hover:bg-amber-900/70"
          >
            Try again
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}
