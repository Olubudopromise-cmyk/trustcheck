'use client';

import { BATCH_MAX_INPUTS } from '../hooks/useBatchVerification';

const countNonEmptyLines = (value: string): number =>
  value.split(/\r?\n/).filter((line) => line.trim().length > 0).length;

// truncateToMaxLines keeps the first BATCH_MAX_INPUTS non-empty lines so a
// paste of 200 lines can never exceed the cap.
const truncateToMaxLines = (value: string): string => {
  const lines: string[] = [];
  for (const raw of value.split(/\r?\n/)) {
    const trimmed = raw.trim();
    if (!trimmed) {
      continue;
    }
    lines.push(raw);
    if (lines.length >= BATCH_MAX_INPUTS) {
      break;
    }
  }
  return lines.join('\n');
};

interface BatchInputProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
  loading: boolean;
}

export default function BatchInput({ value, onChange, onSubmit, loading }: BatchInputProps) {
  const lineCount = countNonEmptyLines(value);
  const atLimit = lineCount >= BATCH_MAX_INPUTS;
  const canSubmit = lineCount > 0 && !loading;

  const handleChange = (next: string) => {
    if (countNonEmptyLines(next) > BATCH_MAX_INPUTS) {
      onChange(truncateToMaxLines(next));
    } else {
      onChange(next);
    }
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        if (canSubmit) {
          onSubmit();
        }
      }}
      className="flex flex-col gap-3"
    >
      <label htmlFor="batch-input" className="sr-only">
        Batch inputs
      </label>
      <textarea
        id="batch-input"
        value={value}
        onChange={(e) => handleChange(e.target.value)}
        disabled={loading}
        rows={5}
        placeholder="Paste multiple inputs to verify, one per line..."
        aria-describedby="batch-line-count"
        className="w-full resize-y rounded-xl border border-slate-300 bg-white px-5 py-4 font-mono text-base text-slate-900 placeholder-slate-400 outline-none ring-cyan-500 focus:border-cyan-500 focus:ring-2 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:placeholder-slate-500 dark:focus:border-cyan-400 dark:focus:ring-cyan-400"
      />
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p
          id="batch-line-count"
          className={`text-xs ${
            atLimit
              ? 'font-semibold text-amber-600 dark:text-amber-400'
              : 'text-slate-500 dark:text-slate-400'
          }`}
        >
          {lineCount} of {BATCH_MAX_INPUTS} lines
        </p>
        <div className="flex flex-col gap-2 sm:flex-row">
          <button
            type="button"
            onClick={() => onChange('')}
            disabled={loading || lineCount === 0}
            className="w-full rounded-xl border border-slate-300 px-5 py-2.5 text-sm font-medium text-slate-700 transition hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 disabled:cursor-not-allowed disabled:opacity-60 sm:w-auto dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-800"
          >
            Clear
          </button>
          <button
            type="submit"
            disabled={!canSubmit}
            className="w-full rounded-xl bg-cyan-500 px-5 py-2.5 text-sm font-medium text-white shadow-lg shadow-cyan-500/20 transition hover:bg-cyan-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 focus-visible:ring-offset-2 ring-offset-slate-50 disabled:cursor-not-allowed disabled:opacity-60 sm:w-auto dark:ring-offset-slate-950"
          >
            {loading ? 'Verifying…' : `Verify ${lineCount} input${lineCount === 1 ? '' : 's'}`}
          </button>
        </div>
      </div>
    </form>
  );
}
