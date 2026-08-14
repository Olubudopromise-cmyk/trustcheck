'use client';

import { memo, useCallback, useRef, useEffect, useState } from 'react';
import type { AnalysisMode } from '../types';
import type { ImageProcessingResult } from '../utils/imageProcessing';
import AnalysisModeSelector from './AnalysisModeSelector';
import ImageUpload from './ImageUpload';

interface WorkspaceComposerProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
  loading: boolean;
  disabled?: boolean;
  mode: AnalysisMode;
  onModeChange: (mode: AnalysisMode) => void;
  onImageAttached?: (result: ImageProcessingResult) => void;
  imageResult?: ImageProcessingResult | null;
}

function WorkspaceComposer({
  value,
  onChange,
  onSubmit,
  loading,
  disabled,
  mode,
  onModeChange,
  onImageAttached,
  imageResult,
}: WorkspaceComposerProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [imageError, setImageError] = useState<string | null>(null);
  const canSubmit = value.trim().length > 0 && !loading && !disabled;

  // Auto-resize textarea
  useEffect(() => {
    const textarea = textareaRef.current;
    if (textarea) {
      textarea.style.height = 'auto';
      textarea.style.height = `${Math.min(textarea.scrollHeight, 150)}px`;
    }
  }, [value]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        if (canSubmit) onSubmit();
      }
    },
    [canSubmit, onSubmit],
  );

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (canSubmit) onSubmit();
    },
    [canSubmit, onSubmit],
  );

  const handleImageProcessed = useCallback(
    (result: ImageProcessingResult) => {
      setImageError(null);
      if (onImageAttached) {
        onImageAttached(result);
      }
    },
    [onImageAttached],
  );

  const handleImageError = useCallback((error: string) => {
    setImageError(error);
  }, []);

  return (
    <div className="sticky bottom-0 z-30 border-t border-slate-200 bg-white/80 backdrop-blur-lg dark:border-slate-800 dark:bg-slate-950/80">
      <form onSubmit={handleSubmit} className="mx-auto max-w-4xl px-4 py-3">
        <div className="flex items-end gap-3 rounded-2xl border border-slate-200 bg-white p-2 shadow-lg shadow-slate-200/50 dark:border-slate-700 dark:bg-slate-900 dark:shadow-none">
          {/* Image upload button */}
          <ImageUpload
            onImageProcessed={handleImageProcessed}
            onError={handleImageError}
            disabled={loading || disabled}
          />
          <textarea
            ref={textareaRef}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Enter a URL, claim, email, or IP address..."
            rows={1}
            disabled={loading || disabled}
            className="min-h-[44px] flex-1 resize-none bg-transparent px-3 py-2 text-sm text-slate-900 placeholder-slate-400 outline-none dark:text-slate-100 dark:placeholder-slate-500"
            aria-label="Research input"
          />
          <button
            type="submit"
            disabled={!canSubmit}
            className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-cyan-500 text-white shadow-lg shadow-cyan-500/20 transition hover:bg-cyan-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:ring-offset-slate-900"
            aria-label="Verify"
          >
            {loading ? (
              <svg className="h-5 w-5 animate-spin" viewBox="0 0 24 24" fill="none">
                <circle
                  className="opacity-25"
                  cx={12}
                  cy={12}
                  r={10}
                  stroke="currentColor"
                  strokeWidth={4}
                />
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"
                />
              </svg>
            ) : (
              <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"
                />
              </svg>
            )}
          </button>
        </div>{' '}
        <AnalysisModeSelector value={mode} onChange={onModeChange} disabled={loading} />
        <p className="mt-2 text-center text-xs text-slate-400 dark:text-slate-500">
          Press Enter to verify &middot; Shift+Enter for new line
        </p>
        {imageError && (
          <p className="mt-2 text-center text-xs text-red-500 dark:text-red-400">{imageError}</p>
        )}
        {imageResult && !imageError && (
          <div className="mt-2 rounded-lg border border-cyan-200 bg-cyan-50 p-2 text-xs text-cyan-700 dark:border-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-300">
            <span className="font-medium">Image evidence attached:</span> {imageResult.imageType}{' '}
            image processed
            {imageResult.extractedText && ' with extracted text'}
          </div>
        )}
      </form>
    </div>
  );
}

export default memo(WorkspaceComposer);
