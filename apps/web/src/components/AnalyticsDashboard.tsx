'use client';

import { memo, useMemo } from 'react';
import type { ReactNode } from 'react';
import type { VerificationHistoryItem } from '../types';
import { computeAnalytics } from '../utils/analytics';
import { formatRelativeTime } from '../utils/relativeTime';
import StatusBadge from './StatusBadge';

const STATUS_BAR_COLORS: Record<string, string> = {
  verified: 'bg-green-500',
  warning: 'bg-yellow-500',
  invalid: 'bg-red-500',
  private: 'bg-blue-500',
  local: 'bg-purple-500',
  unreachable: 'bg-gray-400',
  suggestion: 'bg-indigo-500',
  unknown: 'bg-slate-400',
  not_implemented: 'bg-slate-500',
};

const BUCKET_BAR_COLORS = [
  'bg-red-500',
  'bg-orange-500',
  'bg-yellow-500',
  'bg-blue-500',
  'bg-green-500',
];

function Card({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5 shadow dark:border-slate-800 dark:bg-slate-900">
      <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">{title}</h3>
      <div className="mt-4">{children}</div>
    </div>
  );
}

function Bar({ color, percent }: { color: string; percent: number }) {
  return (
    <div
      className="mt-1.5 h-2 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700"
      aria-hidden="true"
    >
      <div
        className={`h-full rounded-full ${color}`}
        style={{ width: `${Math.min(100, Math.max(0, percent))}%` }}
      />
    </div>
  );
}

function AnalyticsDashboard({ history }: { history: VerificationHistoryItem[] }) {
  const analytics = useMemo(() => computeAnalytics(history), [history]);

  if (analytics.total === 0) {
    return (
      <section aria-label="Verification analytics">
        <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Analytics</h2>
        <div className="mt-3 rounded-xl border border-dashed border-slate-300 bg-white p-8 text-center shadow dark:border-slate-700 dark:bg-slate-900">
          <p className="text-sm font-medium text-slate-700 dark:text-slate-300">
            No analytics yet.
          </p>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            Verify something to begin collecting statistics.
          </p>
        </div>
      </section>
    );
  }

  const kpis = [
    { label: 'Total Verifications', value: analytics.total },
    { label: 'Average Trust Score', value: analytics.averageScore ?? '—' },
    { label: 'Highest Trust Score', value: analytics.highestScore ?? '—' },
    { label: 'Lowest Trust Score', value: analytics.lowestScore ?? '—' },
  ];

  const visibleStatusStats = analytics.statusStats.filter((stat) => stat.count > 0);
  const visibleTypeStats = analytics.typeStats.filter((stat) => stat.count > 0);
  const last = analytics.lastVerification;

  return (
    <section aria-label="Verification analytics">
      <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Analytics</h2>
      <div className="mt-3 space-y-4">
        <dl className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {kpis.map((kpi) => (
            <div
              key={kpi.label}
              className="rounded-xl border border-slate-200 bg-white p-4 shadow dark:border-slate-800 dark:bg-slate-900"
            >
              <dt className="text-xs font-medium text-slate-500 dark:text-slate-400">
                {kpi.label}
              </dt>
              <dd className="mt-1 text-2xl font-semibold tabular-nums text-slate-900 dark:text-slate-100">
                {kpi.value}
              </dd>
            </div>
          ))}
        </dl>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card title="Status Distribution">
            <ul role="list" className="space-y-3">
              {visibleStatusStats.map((stat) => (
                <li key={stat.key}>
                  <div className="flex items-baseline justify-between gap-3 text-sm">
                    <span className="font-medium text-slate-700 dark:text-slate-300">
                      {stat.label}
                    </span>
                    <span className="shrink-0 tabular-nums text-slate-500 dark:text-slate-400">
                      {stat.count}
                      <span aria-hidden="true"> · </span>
                      {stat.percent}%<span className="sr-only"> percent</span>
                    </span>
                  </div>
                  <Bar
                    color={STATUS_BAR_COLORS[stat.key] ?? 'bg-slate-400'}
                    percent={stat.percent}
                  />
                </li>
              ))}
            </ul>
          </Card>

          <Card title="Type Counts">
            <ul role="list" className="space-y-2">
              {visibleTypeStats.map((stat) => (
                <li key={stat.key} className="flex items-baseline justify-between gap-3 text-sm">
                  <span className="text-slate-700 dark:text-slate-300">{stat.label}</span>
                  <span className="shrink-0 font-medium tabular-nums text-slate-900 dark:text-slate-100">
                    {stat.count}
                  </span>
                </li>
              ))}
            </ul>
          </Card>

          <Card title="Trust Score Distribution">
            <ul role="list" className="space-y-3">
              {analytics.scoreBuckets.map((bucket, index) => (
                <li key={bucket.label}>
                  <div className="flex items-baseline justify-between gap-3 text-sm">
                    <span className="font-medium text-slate-700 dark:text-slate-300">
                      {bucket.label}
                    </span>
                    <span className="shrink-0 tabular-nums text-slate-500 dark:text-slate-400">
                      {bucket.count}
                      <span aria-hidden="true"> · </span>
                      {bucket.percent}%<span className="sr-only"> percent</span>
                    </span>
                  </div>
                  <Bar
                    color={BUCKET_BAR_COLORS[index] ?? 'bg-slate-400'}
                    percent={bucket.percent}
                  />
                </li>
              ))}
            </ul>
          </Card>

          <Card title="Average Score Per Type">
            <ul role="list" className="space-y-2">
              {analytics.averageScoreByType.map((stat) => (
                <li key={stat.key} className="flex items-baseline justify-between gap-3 text-sm">
                  <span className="text-slate-700 dark:text-slate-300">{stat.label}</span>
                  <span className="shrink-0 tabular-nums text-slate-900 dark:text-slate-100">
                    {stat.average}
                    <span className="sr-only">
                      , average score across {stat.count} verifications
                    </span>
                  </span>
                </li>
              ))}
            </ul>
          </Card>

          <Card title="Recent Streak">
            <div className="flex items-end gap-2">
              <span className="text-4xl font-semibold tabular-nums text-slate-900 dark:text-slate-100">
                {analytics.streak}
              </span>
              <span className="pb-1 text-sm text-slate-500 dark:text-slate-400">
                verified in a row
              </span>
            </div>
          </Card>

          <Card title="Top Verified Types">
            {analytics.topVerifiedTypes.length === 0 ? (
              <p className="text-sm text-slate-500 dark:text-slate-400">No verified results yet.</p>
            ) : (
              <ol role="list" className="space-y-2">
                {analytics.topVerifiedTypes.map((stat, index) => (
                  <li key={stat.key} className="flex items-baseline justify-between gap-3 text-sm">
                    <span className="flex items-baseline gap-2">
                      <span className="w-5 shrink-0 text-right tabular-nums text-slate-400 dark:text-slate-500">
                        {index + 1}.
                      </span>
                      <span className="text-slate-700 dark:text-slate-300">{stat.label}</span>
                    </span>
                    <span className="shrink-0 font-medium tabular-nums text-slate-900 dark:text-slate-100">
                      {stat.count}
                    </span>
                  </li>
                ))}
              </ol>
            )}
          </Card>

          {last && (
            <div className="lg:col-span-2">
              <Card title="Last Verification">
                <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
                  <StatusBadge status={last.result.status} verdict={last.result.verdict} />
                  <span className="min-w-0 font-mono break-all text-sm text-slate-900 dark:text-slate-100">
                    {last.input}
                  </span>
                  <time
                    dateTime={new Date(last.timestamp).toISOString()}
                    className="shrink-0 text-sm text-slate-500 dark:text-slate-400"
                  >
                    {formatRelativeTime(last.timestamp)}
                  </time>
                </div>
              </Card>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}

export default memo(AnalyticsDashboard);
