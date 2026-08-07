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
  recommendations?: Recommendation[];
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
