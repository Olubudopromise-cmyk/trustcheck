import { describe, expect, it } from 'vitest';
import type { VerifyResponse } from '../types';
import { normalizeAnalysisMode, normalizeVerifyResponse } from './history';

// A brand-new VerifyResponse as produced by the current /verify endpoint: all
// explainable-AI fields present.
const modernResult: VerifyResponse = {
  input: 'Python is a programming language',
  type: 'unknown',
  status: 'verified',
  trustScore: 60,
  summary: 'A summary.',
  evidence: [{ label: 'Check', result: 'pass', points: 10 }],
  verdict: 'Medium',
  keyClaim: 'The claim is that Python is a programming language.',
  entities: [{ name: 'Python', kind: 'organization' }],
  keywords: ['python', 'programming'],
  evidenceFor: [{ label: 'Web search: example.com', result: 'info', points: 5, note: 'A note' }],
  evidenceAgainst: [],
  missingEvidence: ['Not verified: original source.'],
  unknownInformation: ['The author is unknown.'],
  interpretations: [
    {
      title: 'Genuine claim',
      explanation: 'Explanation',
      confidence: 80,
      reasoning: 'R',
      supportingEvidenceCount: 2,
    },
  ],
  warningSignals: [],
  confidence: 72,
  reasoning: ['+ Evidence gathered'],
  timeline: [{ title: 'Claim Detected', summary: 'Summary', details: ['Detail'] }],
  recommendations: [{ title: 'Check sources', description: 'Description' }],
  supportingEvidence: [
    {
      category: 'Academic',
      items: [{ title: 'T', source: 'example.com', credibility: 'high', summary: 'S' }],
    },
  ],
  contradictingEvidence: [
    {
      sourceA: 'A',
      claimA: '1',
      sourceB: 'B',
      claimB: '2',
      whyTheyDisagree: 'W',
      confidenceInContradiction: 50,
    },
  ],
  missingInformation: [{ item: 'Original source', whyItMatters: 'Matters' }],
  confidenceBreakdown: {
    overall: 72,
    metrics: [{ name: 'Evidence', score: 70, note: 'Note' }],
  },
  aiSummary: 'A generated summary.',
  suggestedReading: [{ title: 'Read', publisher: 'Pub', whyItHelps: 'Help' }],
  whatChanged: [{ date: '2026-01-01', event: 'Event' }],
  claims: [
    {
      id: 'c1',
      text: 'Python is a programming language',
      status: 'verified',
      confidence: 80,
      evidence: [{ label: 'E', result: 'pass', points: 5 }],
      timeline: [{ title: 'Claim Detected', summary: 'S', details: ['D'] }],
    },
  ],
  claimCount: 1,
  verifiedClaims: 1,
  analysisMode: 'deep_research',
  evidenceLedger: {
    claim: 'Python is a programming language',
    supporting: [
      {
        source: {
          title: 'T',
          domain: 'example.com',
          sourceType: 'academic',
          relation: 'secondary',
          isOfficial: false,
          relevance: 80,
          supportsClaim: true,
          contradictsClaim: false,
          isIndependent: true,
          confidence: 70,
        },
        summary: 'S',
        strength: 60,
      },
    ],
    contradicting: [],
    unknown: [],
    totalSources: 1,
    independentCount: 1,
    duplicateCount: 0,
  },
  scoreExplanation: {
    evidenceStrength: 70,
    evidenceStrengthNote: 'N',
    sourceQuality: 60,
    sourceQualityNote: 'N',
    independentConfirmation: 50,
    independentNote: 'N',
    contradictionRisk: 20,
    contradictionNote: 'N',
    missingEvidence: 40,
    missingNote: 'N',
  },
  sourceIntelligence: [
    {
      title: 'T',
      domain: 'example.com',
      sourceType: 'academic',
      relation: 'secondary',
      isOfficial: false,
      relevance: 80,
      supportsClaim: true,
      contradictsClaim: false,
      isIndependent: true,
      confidence: 70,
    },
  ],
};

// A legacy result saved before the explainable-AI fields existed.
const legacyResult = {
  input: 'google.com',
  type: 'domain',
  status: 'verified',
  trustScore: 60,
  summary: 'Domain resolves.',
  evidence: [{ label: 'DNS Resolves', result: 'pass', points: 0 }],
};

describe('normalizeVerifyResponse', () => {
  it('passes a brand-new VerifyResponse through with all fields intact', () => {
    const normalized = normalizeVerifyResponse(modernResult);
    expect(normalized.input).toBe('Python is a programming language');
    expect(normalized.analysisMode).toBe('deep_research');
    expect(normalized.timeline?.[0]?.title).toBe('Claim Detected');
    expect(normalized.supportingEvidence?.[0]?.category).toBe('Academic');
    expect(normalized.confidenceBreakdown?.overall).toBe(72);
    expect(normalized.evidenceLedger?.totalSources).toBe(1);
    expect(normalized.claims?.[0]?.status).toBe('verified');
  });

  it('keeps a legacy result (no verdict) renderable with safe defaults', () => {
    const normalized = normalizeVerifyResponse(legacyResult);
    expect(normalized.verdict).toBeUndefined();
    expect(normalized.timeline).toBeUndefined();
    expect(normalized.supportingEvidence).toBeUndefined();
    expect(normalized.evidenceLedger).toBeUndefined();
    expect(normalized.scoreExplanation).toBeUndefined();
    expect(Array.isArray(normalized.evidence)).toBe(true);
    expect(normalized.evidence).toHaveLength(1);
  });

  it('normalizes an old result missing the timeline into an empty-safe shape', () => {
    const partial = { ...modernResult };
    delete (partial as Record<string, unknown>).timeline;
    const normalized = normalizeVerifyResponse(partial);
    expect(normalized.timeline).toBeUndefined();
    // The rest of the analysis still renders.
    expect(normalized.verdict).toBe('Medium');
  });

  it('normalizes an old result missing perspectives into an empty-safe shape', () => {
    const partial = { ...modernResult } as Record<string, unknown>;
    delete partial.supportingEvidence;
    delete partial.contradictingEvidence;
    delete partial.missingInformation;
    delete partial.confidenceBreakdown;
    const normalized = normalizeVerifyResponse(partial);
    expect(normalized.supportingEvidence).toBeUndefined();
    expect(normalized.contradictingEvidence).toEqual([]);
    expect(normalized.missingInformation).toEqual([]);
    expect(normalized.confidenceBreakdown).toBeUndefined();
  });

  it('normalizes an old result missing the evidence ledger', () => {
    const partial = { ...modernResult } as Record<string, unknown>;
    delete partial.evidenceLedger;
    const normalized = normalizeVerifyResponse(partial);
    expect(normalized.evidenceLedger).toBeUndefined();
  });

  it('repairs a partial ledger that lacks counters (older shape)', () => {
    const partial = {
      ...modernResult,
      evidenceLedger: {
        claim: 'Python is a programming language',
        supporting: [
          {
            source: { title: 'T', sourceType: 'academic' },
            summary: 'S',
            strength: 60,
          },
        ],
        contradicting: [],
        unknown: [],
        // no totalSources / independentCount / duplicateCount
      },
    };
    const normalized = normalizeVerifyResponse(partial);
    expect(normalized.evidenceLedger?.totalSources).toBe(0);
    expect(normalized.evidenceLedger?.independentCount).toBe(0);
    expect(normalized.evidenceLedger?.supporting?.[0]?.source?.title).toBe('T');
    expect(normalized.evidenceLedger?.supporting?.[0]?.source?.sourceType).toBe('academic');
  });

  it('repairs a partial confidence breakdown that lacks overall (older shape)', () => {
    const partial = {
      ...modernResult,
      confidenceBreakdown: {
        metrics: [{ name: 'Evidence', score: 70, note: 'Note' }],
      },
    };
    const normalized = normalizeVerifyResponse(partial);
    expect(normalized.confidenceBreakdown?.overall).toBe(0);
    expect(normalized.confidenceBreakdown?.metrics).toHaveLength(1);
  });

  it('drops invalid analysis modes instead of propagating them', () => {
    const partial = { ...modernResult, analysisMode: 'bogus_mode' };
    const normalized = normalizeVerifyResponse(partial);
    expect(normalized.analysisMode).toBeUndefined();
    expect(normalizeAnalysisMode('government_official')).toBe('government_official');
    expect(normalizeAnalysisMode('bogus')).toBeUndefined();
  });

  it('tolerates malformed/partial history records without throwing', () => {
    const malformed = {
      id: 'x',
      input: 'weird',
      result: {
        input: 'weird',
        type: 'unknown',
        status: 'unknown',
        trustScore: 'not-a-number',
        summary: null,
        timeline: null,
        evidence: 'not-an-array',
        claims: [null, { text: 'valid-ish' }],
        supportingEvidence: [{ items: 'not-an-array' }],
      },
    };
    expect(() => normalizeVerifyResponse(malformed.result)).not.toThrow();
    const normalized = normalizeVerifyResponse(malformed.result);
    expect(normalized.trustScore).toBe(0);
    expect(normalized.summary).toBe('');
    expect(normalized.timeline).toBeUndefined();
    expect(normalized.evidence).toEqual([]);
    expect(normalized.claims?.[0]?.text).toBe('valid-ish');
    expect(normalized.supportingEvidence?.[0]?.items).toEqual([]);
  });

  it('returns a safe empty result for non-object input', () => {
    for (const bad of [null, undefined, 'string', 42]) {
      const normalized = normalizeVerifyResponse(bad);
      expect(normalized.input).toBe('');
      expect(normalized.evidence).toEqual([]);
      expect(Array.isArray(normalized.timeline ?? [])).toBe(true);
    }
  });

  it('handles null values in all optional fields without crashing', () => {
    const withNulls = {
      input: 'test',
      type: 'unknown',
      status: 'invalid',
      trustScore: 0,
      summary: '',
      evidence: [],
      verdict: null,
      keyClaim: null,
      entities: null,
      keywords: null,
      evidenceFor: null,
      evidenceAgainst: null,
      missingEvidence: null,
      unknownInformation: null,
      interpretations: null,
      warningSignals: null,
      confidence: null,
      reasoning: null,
      timeline: null,
      recommendations: null,
      supportingEvidence: null,
      contradictingEvidence: null,
      missingInformation: null,
      confidenceBreakdown: null,
      aiSummary: null,
      suggestedReading: null,
      whatChanged: null,
      claims: null,
      claimCount: null,
      verifiedClaims: null,
      partialClaims: null,
      unverifiedClaims: null,
      analysisMode: null,
      evidenceLedger: null,
      scoreExplanation: null,
      sourceIntelligence: null,
    };
    expect(() => normalizeVerifyResponse(withNulls)).not.toThrow();
    const normalized = normalizeVerifyResponse(withNulls);
    expect(normalized.input).toBe('test');
    expect(normalized.evidence).toEqual([]);
    expect(normalized.entities).toEqual([]);
    expect(normalized.keywords).toEqual([]);
    expect(normalized.evidenceFor).toEqual([]);
    expect(normalized.evidenceAgainst).toEqual([]);
    expect(normalized.interpretations).toEqual([]);
    expect(normalized.warningSignals).toEqual([]);
    expect(normalized.timeline).toBeUndefined();
    expect(normalized.recommendations).toEqual([]);
    expect(normalized.contradictingEvidence).toEqual([]);
    expect(normalized.missingInformation).toEqual([]);
    expect(normalized.claims).toEqual([]);
    expect(normalized.sourceIntelligence).toEqual([]);
  });

  it('handles deeply nested malformed data without crashing', () => {
    const deeplyMalformed = {
      input: 'test',
      type: 'unknown',
      status: 'invalid',
      trustScore: 0,
      summary: '',
      evidence: [],
      claims: [
        {
          id: 'c1',
          text: 'claim',
          entities: [{ notName: 123 }],
          keywords: [123, null, 'valid'],
          evidence: [{ notLabel: true }],
          conflicts: [{ notSourceA: true }],
          timeline: [{ notTitle: true }],
          recommendations: [{ notTitle: true }],
          missingInformation: [{ notItem: true }],
        },
      ],
      supportingEvidence: [
        {
          category: 123,
          items: [{ notTitle: true }],
        },
      ],
      contradictingEvidence: [{ notSourceA: true }],
      missingInformation: [{ notItem: true }],
      interpretations: [{ notTitle: true }],
      warningSignals: [{ notLabel: true }],
      recommendations: [{ notTitle: true }],
      suggestedReading: [{ notTitle: true }],
      whatChanged: [{ notDate: true }],
    };
    expect(() => normalizeVerifyResponse(deeplyMalformed)).not.toThrow();
    const normalized = normalizeVerifyResponse(deeplyMalformed);
    expect(normalized.claims).toHaveLength(1);
    expect(normalized.claims?.[0]?.text).toBe('claim');
    expect(normalized.claims?.[0]?.keywords).toEqual(['valid']);
    expect(normalized.supportingEvidence).toHaveLength(1);
  });

  it('preserves valid data when mixed with invalid data in arrays', () => {
    const mixedArray = {
      input: 'test',
      type: 'unknown',
      status: 'invalid',
      trustScore: 0,
      summary: '',
      evidence: [],
      claims: [null, { text: 'valid' }, 123, { text: 'also valid' }],
      interpretations: [
        null,
        {
          title: 'valid',
          explanation: 'e',
          confidence: 50,
          reasoning: 'r',
          supportingEvidenceCount: 1,
        },
      ],
      warningSignals: ['invalid', { label: 'valid', severity: 'low', description: 'd' }],
    };
    const normalized = normalizeVerifyResponse(mixedArray);
    // Only valid objects should be included
    expect(normalized.claims).toHaveLength(2);
    expect(normalized.claims?.[0]?.text).toBe('valid');
    expect(normalized.claims?.[1]?.text).toBe('also valid');
    expect(normalized.interpretations).toHaveLength(1);
    expect(normalized.warningSignals).toHaveLength(1);
  });

  it('handles NaN and Infinity in numeric fields gracefully', () => {
    const withBadNumbers = {
      input: 'test',
      type: 'unknown',
      status: 'invalid',
      trustScore: NaN,
      summary: '',
      evidence: [],
      confidence: Infinity,
      claims: [
        {
          id: 'c1',
          text: 'claim',
          confidence: NaN,
          evidence: [{ label: 'e', result: 'pass', points: Infinity }],
        },
      ],
    };
    const normalized = normalizeVerifyResponse(withBadNumbers);
    // NaN and Infinity should be replaced with safe defaults
    expect(normalized.trustScore).toBe(0);
    expect(normalized.confidence).toBeUndefined();
    expect(normalized.claims?.[0]?.confidence).toBeUndefined();
  });
});
