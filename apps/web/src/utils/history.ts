import type {
  AnalysisMode,
  ClaimStatus,
  Entity,
  EvidenceResult,
  LedgerEntry,
  SourceRelation,
  SourceType,
  VerificationHistoryItem,
  VerifyResponse,
  WarningSeverity,
} from '../types';

const HISTORY_KEY = 'trustcheck:verification-history';
const MAX_ENTRIES = 20;

// makeId prefers crypto.randomUUID (spec); falls back for non-secure contexts.
function makeId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

// --- small safe readers -----------------------------------------------------
// These readers never throw on malformed persisted data. They turn any shape
// of stored JSON into typed primitives with safe defaults, so a record saved
// before a field existed — or written by an older app version — can never
// crash the UI with `.map()` on undefined or destructuring of a non-object.

function asRecord(value: unknown): Record<string, unknown> | null {
  return typeof value === 'object' && value !== null ? (value as Record<string, unknown>) : null;
}

function asString(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined;
}

function asNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined;
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((v): v is string => typeof v === 'string');
}

// asObjectArray returns only array elements that are objects, dropping null
// and scalar entries that would crash rendering with property access.
function asObjectArray(value: unknown): Record<string, unknown>[] {
  if (!Array.isArray(value)) {
    return [];
  }
  const out: Record<string, unknown>[] = [];
  for (const v of value) {
    const rec = asRecord(v);
    if (rec !== null) {
      out.push(rec);
    }
  }
  return out;
}

// --- nested normalizers ----------------------------------------------------

function normalizeTimeline(value: unknown): VerifyResponse['timeline'] {
  const steps = asObjectArray(value);
  if (steps.length === 0) {
    return undefined;
  }
  return steps.map((step) => ({
    title: asString(step.title) ?? '',
    summary: asString(step.summary) ?? '',
    details: asStringArray(step.details),
  }));
}

function normalizeSourceEvidence(value: unknown): VerifyResponse['supportingEvidence'] {
  const groups = asObjectArray(value);
  if (groups.length === 0) {
    return undefined;
  }
  return groups.map((group) => ({
    category: asString(group.category) ?? 'Other',
    items: asObjectArray(group.items).map((item) => ({
      title: asString(item.title) ?? '',
      source: asString(item.source) ?? '',
      credibility: asString(item.credibility) ?? 'unknown',
      publicationDate: asString(item.publicationDate),
      summary: asString(item.summary) ?? '',
    })),
  }));
}

function normalizeContradictions(value: unknown): VerifyResponse['contradictingEvidence'] {
  return asObjectArray(value).map((c) => ({
    sourceA: asString(c.sourceA) ?? '',
    claimA: asString(c.claimA) ?? '',
    sourceB: asString(c.sourceB) ?? '',
    claimB: asString(c.claimB) ?? '',
    whyTheyDisagree: asString(c.whyTheyDisagree) ?? '',
    confidenceInContradiction: asNumber(c.confidenceInContradiction) ?? 0,
  }));
}

function normalizeMissingInfo(value: unknown): VerifyResponse['missingInformation'] {
  return asObjectArray(value).map((m) => ({
    item: asString(m.item) ?? '',
    whyItMatters: asString(m.whyItMatters) ?? '',
  }));
}

function normalizeInterpretations(value: unknown): VerifyResponse['interpretations'] {
  return asObjectArray(value).map((i) => ({
    title: asString(i.title) ?? '',
    explanation: asString(i.explanation) ?? '',
    confidence: asNumber(i.confidence) ?? 0,
    reasoning: asString(i.reasoning) ?? '',
    supportingEvidenceCount: asNumber(i.supportingEvidenceCount) ?? 0,
  }));
}

function normalizeWarningSignals(value: unknown): VerifyResponse['warningSignals'] {
  return asObjectArray(value).map((s) => ({
    label: asString(s.label) ?? '',
    severity: (asString(s.severity) ?? 'low') as WarningSeverity,
    description: asString(s.description) ?? '',
  }));
}

function normalizeRecommendations(value: unknown): VerifyResponse['recommendations'] {
  return asObjectArray(value).map((r) => ({
    title: asString(r.title) ?? '',
    description: asString(r.description) ?? '',
  }));
}

function normalizeSuggestedReading(value: unknown): VerifyResponse['suggestedReading'] {
  return asObjectArray(value).map((r) => ({
    title: asString(r.title) ?? '',
    publisher: asString(r.publisher) ?? '',
    whyItHelps: asString(r.whyItHelps) ?? '',
  }));
}

function normalizeWhatChanged(value: unknown): VerifyResponse['whatChanged'] {
  return asObjectArray(value).map((e) => ({
    date: asString(e.date) ?? '',
    event: asString(e.event) ?? '',
  }));
}

function normalizeSourceIntelligence(value: unknown): VerifyResponse['sourceIntelligence'] {
  return asObjectArray(value).map((s) => ({
    title: asString(s.title) ?? '',
    domain: asString(s.domain),
    publicationDate: asString(s.publicationDate),
    sourceType: (asString(s.sourceType) ?? 'unknown') as SourceType,
    relation: (asString(s.relation) ?? 'tertiary') as SourceRelation,
    isOfficial: s.isOfficial === true,
    author: asString(s.author),
    citation: asString(s.citation),
    relevance: asNumber(s.relevance) ?? 0,
    supportsClaim: s.supportsClaim === true,
    contradictsClaim: s.contradictsClaim === true,
    isIndependent: s.isIndependent === true,
    confidence: asNumber(s.confidence) ?? 0,
  }));
}

function normalizeConfidenceBreakdown(value: unknown): VerifyResponse['confidenceBreakdown'] {
  const obj = asRecord(value);
  if (obj === null) {
    return undefined;
  }
  return {
    overall: asNumber(obj.overall) ?? 0,
    metrics: asObjectArray(obj.metrics).map((m) => ({
      name: asString(m.name) ?? '',
      score: asNumber(m.score) ?? 0,
      note: asString(m.note) ?? '',
    })),
  };
}

function normalizeEvidenceLedger(value: unknown): VerifyResponse['evidenceLedger'] {
  const obj = asRecord(value);
  if (obj === null) {
    return undefined;
  }
  const normalizeEntry = (entry: unknown): LedgerEntry | null => {
    const rec = asRecord(entry);
    if (rec === null) {
      return null;
    }
    const source = asRecord(rec.source);
    return {
      source: {
        title: asString(source?.title) ?? '',
        domain: asString(source?.domain),
        publicationDate: asString(source?.publicationDate),
        sourceType: (asString(source?.sourceType) ?? 'unknown') as SourceType,
        relation: (asString(source?.relation) ?? 'tertiary') as SourceRelation,
        isOfficial: source?.isOfficial === true,
        author: asString(source?.author),
        citation: asString(source?.citation),
        relevance: asNumber(source?.relevance) ?? 0,
        supportsClaim: source?.supportsClaim === true,
        contradictsClaim: source?.contradictsClaim === true,
        isIndependent: source?.isIndependent === true,
        confidence: asNumber(source?.confidence) ?? 0,
      },
      summary: asString(rec.summary) ?? '',
      strength: asNumber(rec.strength) ?? 0,
      notes: asString(rec.notes),
    };
  };

  const supporting = asObjectArray(obj.supporting)
    .map(normalizeEntry)
    .filter((e): e is LedgerEntry => e !== null);
  const contradicting = asObjectArray(obj.contradicting)
    .map(normalizeEntry)
    .filter((e): e is LedgerEntry => e !== null);

  return {
    claim: asString(obj.claim) ?? '',
    supporting,
    contradicting,
    unknown: asStringArray(obj.unknown),
    totalSources: asNumber(obj.totalSources) ?? 0,
    independentCount: asNumber(obj.independentCount) ?? 0,
    duplicateCount: asNumber(obj.duplicateCount) ?? 0,
  };
}

function normalizeScoreExplanation(value: unknown): VerifyResponse['scoreExplanation'] {
  const obj = asRecord(value);
  if (obj === null) {
    return undefined;
  }
  return {
    evidenceStrength: asNumber(obj.evidenceStrength) ?? 0,
    evidenceStrengthNote: asString(obj.evidenceStrengthNote) ?? '',
    sourceQuality: asNumber(obj.sourceQuality) ?? 0,
    sourceQualityNote: asString(obj.sourceQualityNote) ?? '',
    independentConfirmation: asNumber(obj.independentConfirmation) ?? 0,
    independentNote: asString(obj.independentNote) ?? '',
    contradictionRisk: asNumber(obj.contradictionRisk) ?? 0,
    contradictionNote: asString(obj.contradictionNote) ?? '',
    missingEvidence: asNumber(obj.missingEvidence) ?? 0,
    missingNote: asString(obj.missingNote) ?? '',
  };
}

function normalizeClaims(value: unknown): VerifyResponse['claims'] {
  return asObjectArray(value).map((c) => ({
    id: asString(c.id) ?? '',
    text: asString(c.text) ?? '',
    entities: asObjectArray(c.entities).map((e) => ({
      name: asString(e.name) ?? '',
      kind: (asString(e.kind) ?? 'organization') as Entity['kind'],
    })),
    keywords: asStringArray(c.keywords),
    verdict: asString(c.verdict) as VerifyResponse['verdict'],
    confidence: asNumber(c.confidence),
    evidence: asObjectArray(c.evidence).map((e) => ({
      label: asString(e.label) ?? '',
      result: (asString(e.result) ?? 'info') as EvidenceResult,
      points: asNumber(e.points) ?? 0,
      note: asString(e.note),
    })),
    conflicts: normalizeContradictions(c.conflicts),
    summary: asString(c.summary),
    timeline: normalizeTimeline(c.timeline) ?? [],
    recommendations: normalizeRecommendations(c.recommendations),
    missingInformation: normalizeMissingInfo(c.missingInformation),
    status: asString(c.status) as ClaimStatus | undefined,
  }));
}

// --- public API -------------------------------------------------------------

// normalizeAnalysisMode validates a stored mode value against the modes the
// application actually supports. Anything unknown falls back to undefined so
// callers can decide (e.g. default to quick) instead of passing an invalid
// mode into the API.
export function normalizeAnalysisMode(mode: unknown): AnalysisMode | undefined {
  if (mode === 'quick' || mode === 'deep_research' || mode === 'government_official') {
    return mode;
  }
  return undefined;
}

// normalizeVerifyResponse converts any persisted or partial result object into
// a fully populated, type-safe VerifyResponse with safe array and object
// defaults. Old saved results (created before fields such as the reasoning
// timeline, perspectives, ledger, or score explanation existed) are mapped
// here rather than deleted, so saved research remains openable and
// continuable forever.
export function normalizeVerifyResponse(raw: unknown): VerifyResponse {
  if (typeof raw !== 'object' || raw === null) {
    return {
      input: '',
      type: 'unknown',
      status: 'invalid',
      trustScore: 0,
      summary: 'No result data.',
      evidence: [],
    };
  }
  const r = raw as Record<string, unknown>;

  const legacyBase = {
    input: asString(r.input) ?? '',
    type: asString(r.type) ?? 'unknown',
    status: asString(r.status) ?? 'invalid',
    trustScore: asNumber(r.trustScore) ?? 0,
    summary: asString(r.summary) ?? '',
    evidence: asObjectArray(r.evidence).map((e) => ({
      label: asString(e.label) ?? '',
      result: (asString(e.result) ?? 'info') as EvidenceResult,
      points: asNumber(e.points) ?? 0,
    })),
  };

  return {
    ...legacyBase,
    verdict: asString(r.verdict) as VerifyResponse['verdict'],
    keyClaim: asString(r.keyClaim),
    entities: asObjectArray(r.entities).map((e) => ({
      name: asString(e.name) ?? '',
      kind: (asString(e.kind) ?? 'organization') as Entity['kind'],
    })),
    keywords: asStringArray(r.keywords),
    evidenceFor: asObjectArray(r.evidenceFor).map((e) => ({
      label: asString(e.label) ?? '',
      result: (asString(e.result) ?? 'info') as EvidenceResult,
      points: asNumber(e.points) ?? 0,
      note: asString(e.note),
    })),
    evidenceAgainst: asObjectArray(r.evidenceAgainst).map((e) => ({
      label: asString(e.label) ?? '',
      result: (asString(e.result) ?? 'info') as EvidenceResult,
      points: asNumber(e.points) ?? 0,
      note: asString(e.note),
    })),
    missingEvidence: asStringArray(r.missingEvidence),
    unknownInformation: asStringArray(r.unknownInformation),
    interpretations: normalizeInterpretations(r.interpretations),
    warningSignals: normalizeWarningSignals(r.warningSignals),
    confidence: asNumber(r.confidence),
    reasoning: asStringArray(r.reasoning),
    timeline: normalizeTimeline(r.timeline),
    recommendations: normalizeRecommendations(r.recommendations),
    supportingEvidence: normalizeSourceEvidence(r.supportingEvidence),
    contradictingEvidence: normalizeContradictions(r.contradictingEvidence),
    missingInformation: normalizeMissingInfo(r.missingInformation),
    confidenceBreakdown: normalizeConfidenceBreakdown(r.confidenceBreakdown),
    aiSummary: asString(r.aiSummary),
    suggestedReading: normalizeSuggestedReading(r.suggestedReading),
    suggestedReadingNote: asString(r.suggestedReadingNote),
    whatChanged: normalizeWhatChanged(r.whatChanged),
    whatChangedNote: asString(r.whatChangedNote),
    claims: normalizeClaims(r.claims),
    claimCount: asNumber(r.claimCount),
    verifiedClaims: asNumber(r.verifiedClaims),
    partialClaims: asNumber(r.partialClaims),
    unverifiedClaims: asNumber(r.unverifiedClaims),
    analysisMode: normalizeAnalysisMode(r.analysisMode),
    evidenceLedger: normalizeEvidenceLedger(r.evidenceLedger),
    scoreExplanation: normalizeScoreExplanation(r.scoreExplanation),
    sourceIntelligence: normalizeSourceIntelligence(r.sourceIntelligence) ?? [],
    securityReport: (asRecord(r.securityReport) ??
      undefined) as unknown as VerifyResponse['securityReport'],
  };
}

// isHistoryItem guards against corrupt or foreign data written to the same key.
function isHistoryItem(value: unknown): value is VerificationHistoryItem {
  if (typeof value !== 'object' || value === null) {
    return false;
  }
  const item = value as Record<string, unknown>;
  return (
    typeof item.id === 'string' &&
    typeof item.timestamp === 'number' &&
    typeof item.input === 'string' &&
    typeof item.result === 'object' &&
    item.result !== null
  );
}

export function loadHistory(): VerificationHistoryItem[] {
  if (typeof window === 'undefined') {
    return [];
  }
  try {
    const raw = window.localStorage.getItem(HISTORY_KEY);
    if (!raw) {
      return [];
    }
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed
      .filter(isHistoryItem)
      .map((item) => ({
        ...item,
        result: normalizeVerifyResponse(item.result),
      }))
      .slice(0, MAX_ENTRIES);
  } catch {
    return [];
  }
}

export function saveHistory(items: VerificationHistoryItem[]): void {
  if (typeof window === 'undefined') {
    return;
  }
  try {
    window.localStorage.setItem(HISTORY_KEY, JSON.stringify(items.slice(0, MAX_ENTRIES)));
  } catch {
    // Quota or serialization errors must never break a verification.
  }
}

// addHistoryItem prepends a new entry, drops any duplicate for the same input
// (case-insensitive, trimmed) and keeps the newest MAX_ENTRIES. It persists the
// result to localStorage and returns the updated list.
export function addHistoryItem(input: string, result: VerifyResponse): VerificationHistoryItem[] {
  const normalized = input.trim().toLowerCase();
  const item: VerificationHistoryItem = {
    id: makeId(),
    timestamp: Date.now(),
    input: input.trim(),
    result,
  };
  const next = [
    item,
    ...loadHistory().filter((entry) => entry.input.trim().toLowerCase() !== normalized),
  ].slice(0, MAX_ENTRIES);
  saveHistory(next);
  return next;
}

export function removeHistoryItem(id: string): VerificationHistoryItem[] {
  const next = loadHistory().filter((entry) => entry.id !== id);
  saveHistory(next);
  return next;
}

export function clearHistory(): void {
  if (typeof window === 'undefined') {
    return;
  }
  try {
    window.localStorage.removeItem(HISTORY_KEY);
  } catch {
    // Ignore; the in-memory list is cleared regardless by the caller.
  }
}
