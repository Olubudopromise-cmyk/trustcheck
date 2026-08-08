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

// Phase 13: multi-claim types
export type ClaimStatus = 'verified' | 'partially_verified' | 'unverified' | 'no_reliable_evidence';

export interface Claim {
  id: string;
  text: string;
  entities?: Entity[];
  keywords?: string[];
  verdict?: Verdict;
  confidence?: number;
  evidence?: EvidenceItem[];
  conflicts?: Contradiction[];
  summary?: string;
  timeline?: ReasoningStep[];
  recommendations?: Recommendation[];
  missingInformation?: MissingInfo[];
  status?: ClaimStatus;
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
  // Phase 13: intelligent claim extraction. Optional so results saved in
  // local history before this extension shipped still render.
  claims?: Claim[];
  claimCount?: number;
  verifiedClaims?: number;
  partialClaims?: number;
  unverifiedClaims?: number; // Evidence Depth & Analysis Modes. Optional for backward compatibility.
  analysisMode?: AnalysisMode;
  evidenceLedger?: EvidenceLedger;
  scoreExplanation?: ScoreExplanation;
  sourceIntelligence?: SourceIntelligence[];
  // Security Intelligence Engine. Optional for security_review mode.
  securityReport?: SecurityReport;
};

// Evidence Depth & Analysis Modes types
export type AnalysisMode = 'quick' | 'deep_research' | 'government_official';

export interface AnalysisSettings {
  mode: AnalysisMode;
  searchDepth: number;
  maxSources: number;
  requireIndependentSources: boolean;
  searchContradictions: boolean;
  prioritizeGovernmentSources: boolean;
  prioritizeAcademicSources: boolean;
  prioritizePrimarySources: boolean;
  minimumEvidenceThreshold: number;
}

export type SourceType =
  'official' | 'institutional' | 'journalism' | 'community' | 'academic' | 'commercial' | 'unknown';
export type SourceRelation = 'primary' | 'secondary' | 'tertiary';

export interface SourceIntelligence {
  title: string;
  domain?: string;
  publicationDate?: string;
  sourceType: SourceType;
  relation: SourceRelation;
  isOfficial: boolean;
  author?: string;
  citation?: string;
  relevance: number;
  supportsClaim: boolean;
  contradictsClaim: boolean;
  isIndependent: boolean;
  confidence: number;
}

export interface LedgerEntry {
  source: SourceIntelligence;
  summary: string;
  strength: number;
  notes?: string;
}

export interface EvidenceLedger {
  claim: string;
  supporting: LedgerEntry[];
  contradicting: LedgerEntry[];
  unknown: string[];
  totalSources: number;
  independentCount: number;
  duplicateCount: number;
}

export interface ScoreExplanation {
  evidenceStrength: number;
  evidenceStrengthNote: string;
  sourceQuality: number;
  sourceQualityNote: string;
  independentConfirmation: number;
  independentNote: string;
  contradictionRisk: number;
  contradictionNote: string;
  missingEvidence: number;
  missingNote: string;
}

export interface VerificationHistoryItem {
  id: string;
  timestamp: number;
  input: string;
  result: VerifyResponse;
}

export type ApiError = {
  error: string;
};

// Security Intelligence Engine types
export type Severity = 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW' | 'INFO';

export type FindingStatus =
  | 'CONFIRMED'
  | 'REQUIRES_REVIEW'
  | 'FALSE_POSITIVE'
  | 'NOT_FIXED'
  | 'PARTIALLY_FIXED'
  | 'FIXED'
  | 'UNVERIFIED';

export type SecurityCategory =
  | 'injection'
  | 'authentication_weakness'
  | 'authorization_flaw'
  | 'insecure_direct_object_reference'
  | 'server_side_request_forgery'
  | 'cross_site_scripting'
  | 'cross_site_request_forgery'
  | 'insecure_deserialization'
  | 'path_traversal'
  | 'command_execution'
  | 'secrets_exposure'
  | 'insecure_cryptography'
  | 'weak_password_handling'
  | 'unsafe_file_handling'
  | 'insecure_http_configuration'
  | 'dependency_vulnerability'
  | 'excessive_permissions'
  | 'unsafe_error_handling'
  | 'missing_security_controls';

export interface SecurityFinding {
  id: string;
  title: string;
  severity: Severity;
  confidence: number;
  category: SecurityCategory;
  file: string;
  line?: number;
  endLine?: number;
  description: string;
  securityImpact: string;
  evidence: string;
  remediation: string;
  patch?: string;
  references?: string[];
  status: FindingStatus;
  evidenceType: string;
}

export interface DependencyRisk {
  package: string;
  version: string;
  vulnerability: string;
  severity: Severity;
  affectedVersions: string;
  recommendedUpgrade: string;
  isAffected: boolean;
  advisorySource: string;
  retrievalDate: string;
}

export interface RecommendedFix {
  findingId: string;
  priority: number;
  explanation: string;
  patch?: string;
}

export interface VerificationResult {
  findingId: string;
  status: string;
  details: string;
}

export interface SecurityReport {
  executiveSummary: string;
  securityScore: number;
  criticalCount: number;
  highCount: number;
  mediumCount: number;
  lowCount: number;
  infoCount: number;
  findings: SecurityFinding[];
  dependencyRisks: DependencyRisk[];
  recommendedFixes: RecommendedFix[];
  verificationResults?: VerificationResult[];
  remainingRisks: string[];
  evidenceType: string;
}

export interface SecurityResponse {
  report: SecurityReport;
}

export type SecurityMode = 'security_review';
