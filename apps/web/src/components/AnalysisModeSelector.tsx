'use client';

import { memo, useState } from 'react';
import type { AnalysisMode } from '../types';

interface AnalysisModeSelectorProps {
  value: AnalysisMode;
  onChange: (mode: AnalysisMode) => void;
  disabled?: boolean;
}

const MODES: {
  value: AnalysisMode;
  label: string;
  description: string;
  icon: string;
  color: string;
}[] = [
  {
    value: 'quick',
    label: 'Quick',
    description: 'Fast verification with standard evidence search.',
    icon: '\u26a1',
    color:
      'border-cyan-200 bg-cyan-50 text-cyan-700 dark:border-cyan-800 dark:bg-cyan-950/30 dark:text-cyan-300',
  },
  {
    value: 'deep_research',
    label: 'Deep Research',
    description: 'Extensive evidence search with contradiction analysis.',
    icon: '\ud83d\udd0d',
    color:
      'border-indigo-200 bg-indigo-50 text-indigo-700 dark:border-indigo-800 dark:bg-indigo-950/30 dark:text-indigo-300',
  },
  {
    value: 'government_official',
    label: 'Government & Official',
    description: 'Prioritizes official sources with independent verification.',
    icon: '\ud83c\udfdb\ufe0f',
    color:
      'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-300',
  },
];

function AnalysisModeSelector({ value, onChange, disabled }: AnalysisModeSelectorProps) {
  const [showDetails, setShowDetails] = useState(false);

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Analysis Mode</h3>
        <button
          type="button"
          onClick={() => setShowDetails(!showDetails)}
          className="text-xs text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300"
        >
          {showDetails ? 'Hide details' : 'Show details'}
        </button>
      </div>

      <div className="mt-3 grid grid-cols-3 gap-2" role="radiogroup" aria-label="Analysis mode">
        {MODES.map((mode) => {
          const isSelected = value === mode.value;
          return (
            <button
              key={mode.value}
              type="button"
              role="radio"
              aria-checked={isSelected}
              disabled={disabled}
              onClick={() => onChange(mode.value)}
              className={`
                flex flex-col items-center gap-1 rounded-xl border-2 p-3 text-center
                transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500
                ${isSelected ? mode.color : 'border-slate-200 bg-slate-50 text-slate-600 hover:border-slate-300 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-400'}
                ${disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'}
              `}
            >
              <span className="text-xl" aria-hidden="true">
                {mode.icon}
              </span>
              <span className="text-xs font-semibold">{mode.label}</span>
            </button>
          );
        })}
      </div>

      {/* Selected mode description */}
      <p className="mt-3 text-center text-xs text-slate-500 dark:text-slate-400">
        {MODES.find((m) => m.value === value)?.description}
      </p>

      {/* Detailed explanation */}
      {showDetails && (
        <div className="mt-4 space-y-3 border-t border-slate-200 pt-4 dark:border-slate-800">
          <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-800/50">
            <h4 className="text-xs font-semibold text-slate-700 dark:text-slate-300">Quick Mode</h4>
            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
              Optimized for speed. Uses the standard verification pipeline with normal evidence
              search. Best for quick checks where you need a fast answer.
            </p>
          </div>
          <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-800/50">
            <h4 className="text-xs font-semibold text-slate-700 dark:text-slate-300">
              Deep Research Mode
            </h4>
            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
              Searches more sources and actively looks for evidence that disagrees with the claim.
              Identifies primary sources, detects duplicate articles, and compares publication
              dates. Takes longer but provides more comprehensive evidence.
            </p>
          </div>
          <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-800/50">
            <h4 className="text-xs font-semibold text-slate-700 dark:text-slate-300">
              Government & Official Mode
            </h4>
            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
              Prioritizes official government domains, agencies, and institutional sources. Still
              searches for independent confirmation and contradiction. Official sources are labeled
              as primary/secondary but not automatically trusted.
            </p>
          </div>
        </div>
      )}
    </div>
  );
}

export default memo(AnalysisModeSelector);
