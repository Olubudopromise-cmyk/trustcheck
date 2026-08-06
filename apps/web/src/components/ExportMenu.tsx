'use client';

import { useEffect, useRef, useState } from 'react';
import type { VerifyResponse } from '../types';
import { downloadJSON, downloadPDF } from '../utils/report';

const OPTIONS = [
  { key: 'json', label: 'Download JSON', run: downloadJSON },
  { key: 'pdf', label: 'Download PDF / Print Report', run: downloadPDF },
];

export default function ExportMenu({ result }: { result: VerifyResponse }) {
  const [open, setOpen] = useState(false);
  const [focusedIndex, setFocusedIndex] = useState(0);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const itemRefs = useRef<Array<HTMLButtonElement | null>>([]);

  const close = (refocus = true) => {
    setOpen(false);
    if (refocus) {
      triggerRef.current?.focus();
    }
  };

  // Close when clicking outside the trigger or the open menu.
  useEffect(() => {
    if (!open) {
      return;
    }
    const onPointerDown = (e: MouseEvent) => {
      const target = e.target as Node;
      const inside =
        triggerRef.current?.contains(target) || itemRefs.current.some((el) => el?.contains(target));
      if (!inside) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', onPointerDown);
    return () => document.removeEventListener('mousedown', onPointerDown);
  }, [open]);

  // Keep keyboard focus on the highlighted menu item.
  useEffect(() => {
    if (open) {
      itemRefs.current[focusedIndex]?.focus();
    }
  }, [open, focusedIndex]);

  const move = (direction: 1 | -1) => {
    setFocusedIndex((index) => (index + direction + OPTIONS.length) % OPTIONS.length);
  };

  const select = (index: number) => {
    OPTIONS[index].run(result);
    close();
  };

  return (
    <div className="relative inline-block">
      <button
        ref={triggerRef}
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls="export-menu"
        onClick={() => setOpen((value) => !value)}
        onKeyDown={(e) => {
          if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
            e.preventDefault();
            setFocusedIndex(0);
            setOpen(true);
          }
        }}
        className="inline-flex items-center gap-1 rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-800"
      >
        Export
        <span
          aria-hidden="true"
          className={`inline-block transition-transform ${open ? 'rotate-180' : ''}`}
        >
          ▾
        </span>
      </button>

      {open && (
        <div
          id="export-menu"
          role="menu"
          aria-label="Export options"
          onKeyDown={(e) => {
            switch (e.key) {
              case 'Escape':
                e.preventDefault();
                close();
                break;
              case 'ArrowDown':
                e.preventDefault();
                move(1);
                break;
              case 'ArrowUp':
                e.preventDefault();
                move(-1);
                break;
            }
          }}
          className="absolute right-0 z-10 mt-1 w-56 rounded-xl border border-slate-200 bg-white p-1 shadow-lg dark:border-slate-700 dark:bg-slate-900"
        >
          {OPTIONS.map((option, index) => (
            <button
              key={option.key}
              ref={(el) => {
                itemRefs.current[index] = el;
              }}
              type="button"
              role="menuitem"
              tabIndex={index === focusedIndex ? 0 : -1}
              onClick={() => select(index)}
              className={`block w-full rounded-lg px-3 py-2 text-left text-sm transition ${
                index === focusedIndex
                  ? 'bg-cyan-50 text-cyan-700 dark:bg-cyan-950/40 dark:text-cyan-300'
                  : 'text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-800'
              }`}
            >
              {option.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
