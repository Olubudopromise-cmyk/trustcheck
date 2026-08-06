'use client';

interface SearchFormProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
  loading: boolean;
}

export default function SearchForm({ value, onChange, onSubmit, loading }: SearchFormProps) {
  const canSubmit = value.trim().length > 0 && !loading;

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        if (canSubmit) onSubmit();
      }}
      className="mt-10 flex flex-col gap-3"
    >
      <label htmlFor="search-input" className="sr-only">
        Search input
      </label>
      <input
        id="search-input"
        type="text"
        autoComplete="off"
        autoFocus
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={loading}
        placeholder="Enter a website, email, IP address, or company..."
        className="w-full rounded-xl border border-slate-300 bg-white px-5 py-4 text-base text-slate-900 placeholder-slate-400 outline-none ring-cyan-500 focus:border-cyan-500 focus:ring-2 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:placeholder-slate-500 dark:focus:border-cyan-400 dark:focus:ring-cyan-400"
        aria-disabled={loading}
      />
      <button
        type="submit"
        disabled={!canSubmit}
        className="relative w-full rounded-xl bg-cyan-500 py-4 text-base font-medium text-white shadow-lg shadow-cyan-500/20 transition hover:bg-cyan-600 disabled:cursor-not-allowed disabled:opacity-60"
      >
        {loading ? (
          <span className="flex items-center justify-center gap-2">
            <svg
              className="h-5 w-5 animate-spin"
              viewBox="0 0 24 24"
              fill="none"
              aria-hidden="true"
            >
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
            <span>Verifying…</span>
          </span>
        ) : (
          'Verify'
        )}
      </button>
    </form>
  );
}
