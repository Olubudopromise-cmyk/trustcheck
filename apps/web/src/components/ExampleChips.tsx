'use client';

const EXAMPLES = [
  'google.com',
  'test@gmail.com',
  '8.8.8.8',
  '2606:4700:4700::1111',
  '192.168.1.1',
  'OpenAI',
];

export default function ExampleChips({ onSelect }: { onSelect: (value: string) => void }) {
  return (
    <div
      className="mt-8 flex flex-wrap justify-center gap-2"
      role="group"
      aria-label="Example searches"
    >
      {EXAMPLES.map((value) => (
        <button
          key={value}
          type="button"
          onClick={() => onSelect(value)}
          className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-800 shadow-sm transition hover:border-cyan-500 hover:bg-cyan-50 hover:text-cyan-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-cyan-400 dark:hover:bg-slate-900/60 dark:hover:text-cyan-300"
        >
          {value}
        </button>
      ))}
    </div>
  );
}
