'use client';

import { memo, useState } from 'react';
import type { SecurityReport, SecurityFinding, Severity } from '../types';

interface SecurityFindingsProps {
  report: SecurityReport;
}

const SEVERITY_META: Record<
  Severity,
  { symbol: string; label: string; color: string; bgColor: string }
> = {
  CRITICAL: {
    symbol: '\u26a0',
    label: 'CRITICAL',
    color: 'text-red-700 dark:text-red-300',
    bgColor: 'bg-red-50 border-red-200 dark:bg-red-950/20 dark:border-red-800',
  },
  HIGH: {
    symbol: '\u26a0',
    label: 'HIGH',
    color: 'text-orange-700 dark:text-orange-300',
    bgColor: 'bg-orange-50 border-orange-200 dark:bg-orange-950/20 dark:border-orange-800',
  },
  MEDIUM: {
    symbol: '\u25cf',
    label: 'MEDIUM',
    color: 'text-yellow-700 dark:text-yellow-300',
    bgColor: 'bg-yellow-50 border-yellow-200 dark:bg-yellow-950/20 dark:border-yellow-800',
  },
  LOW: {
    symbol: '\u25cb',
    label: 'LOW',
    color: 'text-blue-700 dark:text-blue-300',
    bgColor: 'bg-blue-50 border-blue-200 dark:bg-blue-950/20 dark:border-blue-800',
  },
  INFO: {
    symbol: '\u24d8',
    label: 'INFO',
    color: 'text-slate-600 dark:text-slate-400',
    bgColor: 'bg-slate-50 border-slate-200 dark:bg-slate-800 dark:border-slate-700',
  },
};

function SeverityBadge({ severity }: { severity: Severity }) {
  const meta = SEVERITY_META[severity] ?? SEVERITY_META.INFO;
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-semibold ${meta.color}`}
    >
      <span aria-hidden="true">{meta.symbol}</span>
      {meta.label}
    </span>
  );
}

function FindingCard({ finding }: { finding: SecurityFinding }) {
  const [expanded, setExpanded] = useState(false);
  const meta = SEVERITY_META[finding.severity] ?? SEVERITY_META.INFO;

  return (
    <div
      className={`rounded-xl border ${meta.bgColor} overflow-hidden transition-all duration-200`}
    >
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-3 px-4 py-3 text-left transition hover:bg-white/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 dark:hover:bg-white/5"
        aria-expanded={expanded}
      >
        <SeverityBadge severity={finding.severity} />
        <span className="min-w-0 flex-1">
          <p className="text-sm font-medium text-slate-800 dark:text-slate-200">{finding.title}</p>
          <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
            {finding.file}
            {finding.line ? `:${finding.line}` : ''}
          </p>
        </span>
        <span className="shrink-0 text-xs tabular-nums text-slate-500 dark:text-slate-400">
          {finding.confidence}%
        </span>
        <span
          aria-hidden="true"
          className={`shrink-0 text-slate-400 transition-transform dark:text-slate-500 ${expanded ? 'rotate-180' : ''}`}
        >
          \u25b8
        </span>
      </button>

      {expanded && (
        <div className="border-t border-slate-200/50 px-4 py-4 dark:border-slate-700/50">
          <div className="space-y-3">
            <div>
              <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                Description
              </h4>
              <p className="mt-1 text-sm text-slate-700 dark:text-slate-300">
                {finding.description}
              </p>
            </div>

            <div>
              <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                Security Impact
              </h4>
              <p className="mt-1 text-sm text-slate-700 dark:text-slate-300">
                {finding.securityImpact}
              </p>
            </div>

            <div>
              <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                Evidence
              </h4>
              <pre className="mt-1 overflow-x-auto rounded-lg bg-slate-100 p-3 text-xs text-slate-700 dark:bg-slate-800 dark:text-slate-300">
                {finding.evidence}
              </pre>
            </div>

            <div>
              <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                Remediation
              </h4>
              <p className="mt-1 text-sm text-slate-700 dark:text-slate-300">
                {finding.remediation}
              </p>
            </div>

            {finding.patch && (
              <div>
                <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                  Suggested Patch
                </h4>
                <pre className="mt-1 overflow-x-auto rounded-lg bg-green-50 p-3 text-xs text-green-800 dark:bg-green-950/30 dark:text-green-300">
                  {finding.patch}
                </pre>
              </div>
            )}

            <div className="flex flex-wrap gap-2">
              <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-800 dark:text-slate-400">
                {finding.category.replace(/_/g, ' ')}
              </span>
              <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-800 dark:text-slate-400">
                {finding.status.replace(/_/g, ' ')}
              </span>
              <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-800 dark:text-slate-400">
                {finding.evidenceType.replace(/_/g, ' ')}
              </span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function SecurityScore({ score }: { score: number }) {
  const color =
    score >= 80
      ? 'text-green-600 dark:text-green-400'
      : score >= 60
        ? 'text-yellow-600 dark:text-yellow-400'
        : 'text-red-600 dark:text-red-400';

  return (
    <div className="flex items-center gap-4">
      <div className="relative h-20 w-20">
        <svg className="h-20 w-20 -rotate-90" viewBox="0 0 36 36">
          <path
            className="text-slate-200 dark:text-slate-700"
            stroke="currentColor"
            strokeWidth="3"
            fill="none"
            d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
          />
          <path
            className={color}
            stroke="currentColor"
            strokeWidth="3"
            fill="none"
            strokeDasharray={`${score}, 100`}
            d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
          />
        </svg>
        <div className="absolute inset-0 flex items-center justify-center">
          <span className={`text-lg font-bold ${color}`}>{score}</span>
        </div>
      </div>
      <div>
        <p className="text-sm font-medium text-slate-900 dark:text-slate-100">Security Score</p>
        <p className="text-xs text-slate-500 dark:text-slate-400">
          {score >= 80 ? 'Good' : score >= 60 ? 'Fair' : 'Needs improvement'}
        </p>
      </div>
    </div>
  );
}

function SecurityFindings({ report }: SecurityFindingsProps) {
  const [filter, setFilter] = useState<Severity | 'all'>('all');

  const filteredFindings =
    filter === 'all' ? report.findings : report.findings.filter((f) => f.severity === filter);

  return (
    <section aria-label="Security Analysis" className="space-y-4">
      {/* Executive Summary */}
      <div className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
          Executive Summary
        </h3>
        <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">{report.executiveSummary}</p>
      </div>

      {/* Security Score */}
      <div className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
        <SecurityScore score={report.securityScore} />
      </div>

      {/* Findings by Severity */}
      <div className="grid grid-cols-5 gap-2">
        {(['CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFO'] as Severity[]).map((severity) => {
          const count =
            severity === 'CRITICAL'
              ? report.criticalCount
              : severity === 'HIGH'
                ? report.highCount
                : severity === 'MEDIUM'
                  ? report.mediumCount
                  : severity === 'LOW'
                    ? report.lowCount
                    : report.infoCount;
          return (
            <button
              key={severity}
              type="button"
              onClick={() => setFilter(filter === severity ? 'all' : severity)}
              className={`rounded-lg p-2 text-center transition ${
                filter === severity
                  ? 'ring-2 ring-cyan-500 ' + SEVERITY_META[severity].bgColor
                  : 'bg-slate-50 hover:bg-slate-100 dark:bg-slate-800 dark:hover:bg-slate-700'
              }`}
            >
              <span className={`text-lg font-bold ${SEVERITY_META[severity].color}`}>{count}</span>
              <p className="text-xs text-slate-500 dark:text-slate-400">{severity}</p>
            </button>
          );
        })}
      </div>

      {/* Filtered Findings */}
      <div className="space-y-2">
        {filteredFindings.map((finding) => (
          <FindingCard key={finding.id} finding={finding} />
        ))}
      </div>

      {filteredFindings.length === 0 && (
        <p className="py-4 text-center text-sm text-slate-500 dark:text-slate-400">
          No findings match the selected filter.
        </p>
      )}

      {/* Recommended Fixes */}
      {report.recommendedFixes.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
            Recommended Fixes
          </h3>
          <ol className="mt-2 space-y-2">
            {report.recommendedFixes.map((fix) => (
              <li key={fix.findingId} className="flex items-start gap-2">
                <span className="shrink-0 rounded-full bg-cyan-100 px-2 py-0.5 text-xs font-semibold text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-300">
                  #{fix.priority}
                </span>
                <p className="text-sm text-slate-600 dark:text-slate-400">{fix.explanation}</p>
              </li>
            ))}
          </ol>
        </div>
      )}
    </section>
  );
}

export default memo(SecurityFindings);
