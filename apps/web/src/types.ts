export type EvidenceResult = 'pass' | 'warning' | 'fail' | 'info';

export interface Evidence {
  label: string;
  result: EvidenceResult;
  points: number;
}

export type Verdict = 'High' | 'Medium' | 'Low';

export type EntityKind = 'organization' | 'location' | 'person' | 'date';

export interface Entity {
  name: string;
  kind: EntityKind;
}

export interface EvidenceItem {
  label: string;
  result: EvidenceResult;
  points: number;
  note?: string;
}

export interface Interpretation {
  title: string;
  explanation: string;
  confidence: number;
  reasoning: string;
  supportingEvidenceCount: number;
}

export type WarningSeverity = 'high' | 'medium' | 'low';

export interface WarningSignal {
  label: string;
  severity: WarningSeverity;
  description: string;
}

export interface Recommendation {
  title: string;
  description: string;
}

export interface ReasoningStep {
  title: string;
  summary: string;
  details: string[];
}

export interface SourceEvidence {
  title: string;
  source: string;
  credibility: string;
  publicationDate?: string;
  summary: string;
}

export interface SourceGroup {
  category: string;
  items: SourceEvidence[];
}

export interface Contradiction {
  sourceA: string;
  claimA: string;
  sourceB: string;
  claimB: string;
  whyTheyDisagree: string;
  confidenceInContradiction: number;
}

export interface MissingInfo {
  item: string;
  whyItMatters: string;
}

export interface ConfidenceMetric {
  name: string;
  score: number;
  note: string;
}

export interface ConfidenceBreakdown {
  overall: number;
  metrics: ConfidenceMetric[];
}

export interface SuggestedReading {
  title: string;
  publisher: string;
  whyItHelps: string;
}

export interface ChangeEvent {
  date: string;
  event: string;
}

export type VerifyResponse = {
  input: string;
  type: string;
  status: string;
  trustScore: number;
  summary: string;
  evidence: Evidence[];
  // Explainable-AI extension. Optional so results saved in local history
  // before the extension was shipped still render with the legacy card.
  verdict?: Verdict;
  keyClaim?: string;
  entities?: Entity[];
  keywords?: string[];
  evidenceFor?: EvidenceItem[];
  evidenceAgainst?: EvidenceItem[];
  missingEvidence?: string[];
  unknownInformation?: string[];
  interpretations?: Interpretation[];
  warningSignals?: WarningSignal[];
  confidence?: number;
  reasoning?: string[];
  timeline?: ReasoningStep[];
  recommendations?: Recommendation[];
  // Phase 12: multi-perspective fact analysis. Optional so results saved in
  // local history before this extension shipped still render.
  supportingEvidence?: SourceGroup[];
  contradictingEvidence?: Contradiction[];
  missingInformation?: MissingInfo[];
  confidenceBreakdown?: ConfidenceBreakdown;
  aiSummary?: string;
  suggestedReading?: SuggestedReading[];
  suggestedReadingNote?: string;
  whatChanged?: ChangeEvent[];
  whatChangedNote?: string;
};

export interface VerificationHistoryItem {
  id: string;
  timestamp: number;
  input: string;
  result: VerifyResponse;
}

export type ApiError = {
  error: string;
};
