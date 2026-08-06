export type EvidenceResult = 'pass' | 'warning' | 'fail' | 'info';

export interface Evidence {
  label: string;
  result: EvidenceResult;
  points: number;
}

export type VerifyResponse = {
  input: string;
  type: string;
  status: string;
  trustScore: number;
  summary: string;
  evidence: Evidence[];
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
