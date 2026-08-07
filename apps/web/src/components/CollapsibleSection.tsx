'use client';

import { memo, useState } from 'react';
import type { ReactNode } from 'react';

// CollapsibleSection wraps a result section in a toggleable card so the
// analysis page stays scannable instead of becoming a wall of text. It is
// keyboard accessible and announces its expanded state to screen readers.
function CollapsibleSection({
  title,
  badge,
  defaultOpen = false,
  children,
}: {
  title: string;
  badge?: string;
  defaultOpen?: boolean;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);

  return (
    <section className="rounded-xl border border-slate-200 bg-white shadow dark:border-slate-800 dark:bg-slate-900">
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
        className="flex w-full items-center justify-between gap-3 rounded-xl px-4 py-3 text-left transition hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 dark:hover:bg-slate-800/60"
      >
        <span className="flex min-w-0 items-center gap-2">
          <span className="truncate text-sm font-semibold text-slate-900 dark:text-slate-100">
            {title}
          </span>
          {badge && (
            <span className="shrink-0 rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium tabular-nums text-slate-500 dark:bg-slate-800 dark:text-slate-400">
              {badge}
            </span>
          )}
        </span>
        <span
          aria-hidden="true"
          className={`shrink-0 text-slate-400 transition-transform dark:text-slate-500 ${open ? 'rotate-180' : ''}`}
        >
          ▾
        </span>
      </button>
      {open && (
        <div className="border-t border-slate-200 px-4 py-4 dark:border-slate-800">{children}</div>
      )}
    </section>
  );
}

export default memo(CollapsibleSection);
