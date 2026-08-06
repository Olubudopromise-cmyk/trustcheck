import type { VerificationHistoryItem } from '../types';

// analytics.ts computes all dashboard statistics locally from verification
// history. Everything is pure (no storage, no API): the history array is the
// single source of truth. History is stored newest-first, so the most recent
// verification is history[0].

export const STATUS_LABELS: Record<string, string> = {
  verified: 'Verified',
  warning: 'Warning',
  invalid: 'Invalid',
  private: 'Private',
  local: 'Local',
  unreachable: 'Unreachable',
  suggestion: 'Suggestion',
  unknown: 'Unknown',
  not_implemented: 'Not Implemented',
};

export const STATUS_ORDER = [
  'verified',
  'warning',
  'invalid',
  'private',
  'local',
  'unreachable',
  'suggestion',
  'unknown',
  'not_implemented',
] as const;

export const TYPE_LABELS: Record<string, string> = {
  domain: 'Domain',
  url: 'URL',
  email: 'Email',
  phone: 'Phone',
  ipv4: 'IPv4',
  ipv6: 'IPv6',
  company: 'Company',
  unknown: 'Unknown',
};

export const TYPE_ORDER = [
  'domain',
  'url',
  'email',
  'phone',
  'ipv4',
  'ipv6',
  'company',
  'unknown',
] as const;

const SCORE_BUCKETS = [
  { label: '0–19', min: 0, max: 19 },
  { label: '20–39', min: 20, max: 39 },
  { label: '40–59', min: 40, max: 59 },
  { label: '60–79', min: 60, max: 79 },
  { label: '80–100', min: 80, max: 100 },
] as const;

export interface StatusStat {
  key: string;
  label: string;
  count: number;
  percent: number;
}

export interface TypeStat {
  key: string;
  label: string;
  count: number;
}

export interface AverageByTypeStat extends TypeStat {
  average: number;
}

export interface ScoreBucketStat {
  label: string;
  count: number;
  percent: number;
}

export interface Analytics {
  total: number;
  averageScore: number | null;
  highestScore: number | null;
  lowestScore: number | null;
  statusStats: StatusStat[];
  typeStats: TypeStat[];
  averageScoreByType: AverageByTypeStat[];
  scoreBuckets: ScoreBucketStat[];
  lastVerification: VerificationHistoryItem | null;
  streak: number;
  topVerifiedTypes: TypeStat[];
}

const roundPercent = (count: number, total: number): number =>
  total === 0 ? 0 : Math.round((count / total) * 100);

export function computeAnalytics(history: VerificationHistoryItem[]): Analytics {
  const total = history.length;

  const scores = history.map((item) => item.result.trustScore);
  const averageScore =
    total === 0 ? null : Math.round(scores.reduce((sum, s) => sum + s, 0) / total);
  const highestScore = total === 0 ? null : Math.max(...scores);
  const lowestScore = total === 0 ? null : Math.min(...scores);

  const statusCounts = new Map<string, number>();
  const typeCounts = new Map<string, number>();
  const typeScoreSum = new Map<string, { sum: number; count: number }>();
  const verifiedTypeCounts = new Map<string, number>();

  for (const item of history) {
    const statusKey = item.result.status || 'unknown';
    statusCounts.set(statusKey, (statusCounts.get(statusKey) ?? 0) + 1);

    const typeKey = item.result.type || 'unknown';
    typeCounts.set(typeKey, (typeCounts.get(typeKey) ?? 0) + 1);
    const agg = typeScoreSum.get(typeKey) ?? { sum: 0, count: 0 };
    agg.sum += item.result.trustScore;
    agg.count += 1;
    typeScoreSum.set(typeKey, agg);

    if (statusKey === 'verified') {
      verifiedTypeCounts.set(typeKey, (verifiedTypeCounts.get(typeKey) ?? 0) + 1);
    }
  }

  const statusStats: StatusStat[] = STATUS_ORDER.map((key) => ({
    key,
    label: STATUS_LABELS[key] ?? key,
    count: statusCounts.get(key) ?? 0,
    percent: roundPercent(statusCounts.get(key) ?? 0, total),
  }));

  const typeStats: TypeStat[] = TYPE_ORDER.map((key) => ({
    key,
    label: TYPE_LABELS[key] ?? key,
    count: typeCounts.get(key) ?? 0,
  }));

  const averageScoreByType: AverageByTypeStat[] = TYPE_ORDER.flatMap((key) => {
    const agg = typeScoreSum.get(key);
    if (!agg || agg.count === 0) {
      return [];
    }
    return [
      {
        key,
        label: TYPE_LABELS[key] ?? key,
        count: agg.count,
        average: Math.round(agg.sum / agg.count),
      },
    ];
  });

  const scoreBuckets: ScoreBucketStat[] = SCORE_BUCKETS.map((bucket) => {
    const count = scores.filter((score) => score >= bucket.min && score <= bucket.max).length;
    return { label: bucket.label, count, percent: roundPercent(count, total) };
  });

  const lastVerification = total === 0 ? null : history[0];

  let streak = 0;
  for (const item of history) {
    if (item.result.status === 'verified') {
      streak += 1;
    } else {
      break;
    }
  }

  const topVerifiedTypes: TypeStat[] = [...verifiedTypeCounts.entries()]
    .map(([key, count]) => ({ key, label: TYPE_LABELS[key] ?? key, count }))
    .sort((a, b) => b.count - a.count || a.label.localeCompare(b.label));

  return {
    total,
    averageScore,
    highestScore,
    lowestScore,
    statusStats,
    typeStats,
    averageScoreByType,
    scoreBuckets,
    lastVerification,
    streak,
    topVerifiedTypes,
  };
}
