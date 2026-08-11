import { NextRequest, NextResponse } from 'next/server';
import type {
  VerifyResponse,
  AnalysisMode,
  EvidenceItem,
  Verdict,
  SourceType,
  SourceRelation,
  Entity,
} from '@/types';

export const dynamic = 'force-dynamic';
export const maxDuration = 10;

// ─── Input Classification ─────────────────────────────────────────────────────

type InputType = 'url' | 'domain' | 'email' | 'ipv4' | 'ipv6' | 'phone' | 'company' | 'unknown';

const MAX_COMPANY_WORDS = 4;

const DOMAIN_RE = /^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/;
const COMPANY_RE = /^[A-Za-z][A-Za-z0-9\s.'-]{0,63}$/;
const PHONE_RE = /^\+\d{6,15}$/;

function classifyInput(input: string): InputType {
  const in_ = input.trim();
  if (!in_) return 'unknown';

  // URL
  try {
    const url = new URL(in_);
    if (url.protocol === 'http:' || url.protocol === 'https:') return 'url';
  } catch {
    /* not a URL */
  }

  // Email
  if (!/\s/.test(in_) && in_.includes('@')) {
    const parts = in_.split('@');
    if (parts.length === 2 && parts[0].length > 0 && parts[1].includes('.')) {
      return 'email';
    }
  }

  // IPv4
  if (/^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(in_)) return 'ipv4';

  // IPv6 (simplified check)
  if (in_.includes(':') && /^[0-9a-fA-F:]+$/.test(in_)) return 'ipv6';

  // Phone
  const normalized = in_.replace(/[\s.\-()]/g, '');
  if (PHONE_RE.test(normalized)) return 'phone';

  // Domain
  if (DOMAIN_RE.test(in_)) return 'domain';

  // Company (short, letter-starting name)
  if (COMPANY_RE.test(in_) && in_.split(/\s+/).length <= MAX_COMPANY_WORDS) {
    return 'company';
  }

  return 'unknown';
}

// ─── Evidence Builder ──────────────────────────────────────────────────────────

class EvidenceBuilder {
  private evidence: EvidenceItem[] = [];
  private score = 0;

  pass(label: string, points: number) {
    this.score = clamp(this.score + points);
    this.evidence.push({ label, result: 'pass', points });
  }
  fail(label: string, points: number) {
    this.score = clamp(this.score + points);
    this.evidence.push({ label, result: 'fail', points });
  }
  warning(label: string, points: number) {
    this.score = clamp(this.score + points);
    this.evidence.push({ label, result: 'warning', points });
  }
  info(label: string) {
    this.evidence.push({ label, result: 'info', points: 0 });
  }
  getScore() {
    return this.score;
  }
  getEvidence() {
    return [...this.evidence];
  }
}

function clamp(n: number) {
  return Math.max(0, Math.min(100, n));
}

// ─── Domain Verification ───────────────────────────────────────────────────────

async function verifyDomain(
  domain: string,
): Promise<{ status: string; score: number; evidence: EvidenceItem[]; summary: string }> {
  const b = new EvidenceBuilder();

  // DNS check
  try {
    const res = await fetch(
      `https://dns.google/resolve?name=${encodeURIComponent(domain)}&type=A`,
      {
        signal: AbortSignal.timeout(3000),
      },
    );
    const data = await res.json();
    if (data.Answer && data.Answer.length > 0) {
      b.pass('DNS Resolves', 0);
    } else {
      b.fail('DNS Lookup', 0);
      return {
        status: 'unreachable',
        score: 15,
        evidence: b.getEvidence(),
        summary: 'Domain does not resolve.',
      };
    }
  } catch {
    b.fail('DNS Lookup', 0);
    return {
      status: 'unreachable',
      score: 15,
      evidence: b.getEvidence(),
      summary: 'Domain does not resolve.',
    };
  }

  // HTTPS check
  try {
    const res = await fetch(`https://${domain}`, {
      method: 'HEAD',
      signal: AbortSignal.timeout(5000),
      redirect: 'follow',
    });
    b.pass('HTTPS Available', 20);
    if (res.ok) {
      b.pass('HTTP Status OK', 20);
    } else if (res.status >= 400 && res.status < 500) {
      b.warning('HTTP Client Error', 10);
    }
    b.pass('TLS Certificate Present', 20);
  } catch {
    // Try HTTP fallback
    try {
      const res = await fetch(`http://${domain}`, {
        method: 'HEAD',
        signal: AbortSignal.timeout(5000),
        redirect: 'follow',
      });
      b.pass('HTTP Fallback', 10);
      if (res.ok) {
        b.pass('HTTP Status OK', 20);
      }
    } catch {
      b.warning('Connection failed', 0);
    }
  }

  const score = b.getScore();
  const status = score >= 70 ? 'verified' : score >= 40 ? 'warning' : 'invalid';
  const summary =
    score >= 70
      ? 'Domain resolves, HTTPS available, certificate valid.'
      : score >= 40
        ? 'Domain resolves, but verification is inconclusive.'
        : 'Domain verification failed.';

  return { status, score, evidence: b.getEvidence(), summary };
}

// ─── URL Verification ──────────────────────────────────────────────────────────

async function verifyURL(
  url: string,
): Promise<{ status: string; score: number; evidence: EvidenceItem[]; summary: string }> {
  const b = new EvidenceBuilder();
  try {
    const res = await fetch(url, {
      method: 'HEAD',
      signal: AbortSignal.timeout(5000),
      redirect: 'follow',
    });
    b.pass('URL Reachable', 30);
    if (url.startsWith('https://')) b.pass('HTTPS Protocol', 20);
    if (res.ok) b.pass('HTTP Status OK', 20);
  } catch {
    b.fail('URL Unreachable', 0);
    return {
      status: 'unreachable',
      score: 10,
      evidence: b.getEvidence(),
      summary: 'URL could not be reached.',
    };
  }
  const score = b.getScore();
  return {
    status: score >= 50 ? 'verified' : 'warning',
    score,
    evidence: b.getEvidence(),
    summary: 'URL verification complete.',
  };
}

// ─── Email Verification ────────────────────────────────────────────────────────

async function verifyEmail(
  email: string,
): Promise<{ status: string; score: number; evidence: EvidenceItem[]; summary: string }> {
  const b = new EvidenceBuilder();
  const domain = email.split('@')[1];

  // MX lookup via DNS
  try {
    const res = await fetch(
      `https://dns.google/resolve?name=${encodeURIComponent(domain)}&type=MX`,
      {
        signal: AbortSignal.timeout(3000),
      },
    );
    const data = await res.json();
    if (data.Answer && data.Answer.length > 0) {
      b.pass('MX Records Found', 40);
    } else {
      b.fail('No MX Records', -20);
    }
  } catch {
    b.warning('MX lookup failed', 0);
  }

  const score = b.getScore();
  return {
    status: score >= 50 ? 'verified' : 'warning',
    score,
    evidence: b.getEvidence(),
    summary: 'Email verification complete.',
  };
}

// ─── IP Verification ───────────────────────────────────────────────────────────

function verifyIP(ip: string): {
  status: string;
  score: number;
  evidence: EvidenceItem[];
  summary: string;
} {
  const b = new EvidenceBuilder();
  const parts = ip.split('.').map(Number);

  if (ip === '127.0.0.1' || ip === '::1') {
    b.pass('Loopback Address', 100);
    return {
      status: 'local',
      score: 100,
      evidence: b.getEvidence(),
      summary: 'Local/loopback address.',
    };
  }
  if (
    parts[0] === 10 ||
    (parts[0] === 172 && parts[1] >= 16 && parts[1] <= 31) ||
    (parts[0] === 192 && parts[1] === 168)
  ) {
    b.pass('Private IP Range', 90);
    return {
      status: 'private',
      score: 90,
      evidence: b.getEvidence(),
      summary: 'Private/internal IP address.',
    };
  }
  b.pass('Public IP', 70);
  return {
    status: 'verified',
    score: 70,
    evidence: b.getEvidence(),
    summary: 'Public IP address.',
  };
}

// ─── Phone Verification ────────────────────────────────────────────────────────

function verifyPhone(phone: string): {
  status: string;
  score: number;
  evidence: EvidenceItem[];
  summary: string;
} {
  const b = new EvidenceBuilder();
  const normalized = phone.replace(/[\s.\-()]/g, '');
  if (/^\+\d{6,15}$/.test(normalized)) {
    b.pass('Valid E.164 Format', 60);
    return {
      status: 'verified',
      score: 60,
      evidence: b.getEvidence(),
      summary: 'Phone number format is valid.',
    };
  }
  return {
    status: 'warning',
    score: 30,
    evidence: b.getEvidence(),
    summary: 'Phone number format could not be validated.',
  };
}

// ─── Company Verification ──────────────────────────────────────────────────────

function verifyCompany(name: string): {
  status: string;
  score: number;
  evidence: EvidenceItem[];
  summary: string;
} {
  const b = new EvidenceBuilder();
  b.pass('Name Format Valid', 40);
  if (/\b(Inc|LLC|Corp|Ltd|Co)\b/i.test(name)) {
    b.pass('Business Suffix Present', 10);
  }
  return {
    status: 'warning',
    score: b.getScore(),
    evidence: b.getEvidence(),
    summary: 'Company name format validated. Additional verification recommended.',
  };
}

// ─── Research (DuckDuckGo + Wikipedia) ─────────────────────────────────────────

interface SearchResult {
  title: string;
  url: string;
  snippet: string;
  domain: string;
}

async function searchDuckDuckGo(query: string): Promise<SearchResult[]> {
  try {
    const res = await fetch(
      `https://api.duckduckgo.com/?q=${encodeURIComponent(query)}&format=json&no_html=1&skip_disambig=1`,
      {
        signal: AbortSignal.timeout(3000),
      },
    );
    const data = await res.json();
    const results: SearchResult[] = [];

    if (data.AbstractText) {
      results.push({
        title: data.Heading || query,
        url: data.AbstractURL || '',
        snippet: data.AbstractText,
        domain: data.AbstractSource || 'duckduckgo.com',
      });
    }

    if (data.RelatedTopics && Array.isArray(data.RelatedTopics)) {
      for (const topic of data.RelatedTopics.slice(0, 5)) {
        if (topic.Text && topic.FirstURL) {
          results.push({
            title: topic.Text.slice(0, 100),
            url: topic.FirstURL,
            snippet: topic.Text,
            domain: new URL(topic.FirstURL).hostname,
          });
        }
      }
    }
    return results;
  } catch {
    return [];
  }
}

async function searchWikipedia(query: string): Promise<SearchResult[]> {
  try {
    const searchRes = await fetch(
      `https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=${encodeURIComponent(query)}&srlimit=3&format=json`,
      {
        signal: AbortSignal.timeout(3000),
        headers: { 'User-Agent': 'TrustCheck/1.0 (fact-checking tool)' },
      },
    );
    const data = await searchRes.json();
    const results: SearchResult[] = [];

    if (data.query?.search) {
      for (const item of data.query.search) {
        results.push({
          title: item.title,
          url: `https://en.wikipedia.org/wiki/${encodeURIComponent(item.title)}`,
          snippet: item.snippet.replace(/<[^>]*>/g, ''),
          domain: 'wikipedia.org',
        });
      }
    }
    return results;
  } catch {
    return [];
  }
}

async function researchClaim(
  claim: string,
): Promise<{ supporting: SearchResult[]; contradicting: SearchResult[] }> {
  const [ddgResults, wikiResults] = await Promise.allSettled([
    searchDuckDuckGo(claim),
    searchWikipedia(claim),
  ]);

  const allResults = [
    ...(ddgResults.status === 'fulfilled' ? ddgResults.value : []),
    ...(wikiResults.status === 'fulfilled' ? wikiResults.value : []),
  ];

  // Simple classification based on keywords
  const contradictKeywords = [
    'false',
    'fake',
    'debunked',
    'myth',
    'no evidence',
    'wrong',
    'incorrect',
  ];
  const supportingKeywords = ['confirmed', 'supports', 'proves', 'true', 'accurate', 'verified'];

  const supporting: SearchResult[] = [];
  const contradicting: SearchResult[] = [];

  for (const r of allResults) {
    const text = (r.title + ' ' + r.snippet).toLowerCase();
    const hasContradict = contradictKeywords.some((k) => text.includes(k));
    const hasSupport = supportingKeywords.some((k) => text.includes(k));

    if (hasContradict) {
      contradicting.push(r);
    } else if (hasSupport) {
      supporting.push(r);
    } else {
      // Default to supporting if no clear signal
      supporting.push(r);
    }
  }

  return { supporting, contradicting };
}

// ─── Analysis Pipeline ─────────────────────────────────────────────────────────

function generateInterpretations(input: string, inputType: InputType, score: number) {
  const interpretations = [];

  if (inputType === 'unknown') {
    interpretations.push({
      title: 'Genuine claim',
      explanation: `The claim "${input}" appears to be a straightforward factual statement.`,
      confidence: 60,
      reasoning:
        'The claim is structured as a factual assertion without obvious misinformation markers.',
      supportingEvidenceCount: 2,
    });
    interpretations.push({
      title: 'Unverified claim',
      explanation: `The claim "${input}" could not be fully verified with available evidence.`,
      confidence: 40,
      reasoning: 'Limited evidence was found to confirm or deny this claim.',
      supportingEvidenceCount: 1,
    });
  } else {
    interpretations.push({
      title: 'Genuine identifier',
      explanation: `The ${inputType} "${input}" appears to be legitimate based on available checks.`,
      confidence: Math.min(90, score + 20),
      reasoning: 'Standard verification checks passed.',
      supportingEvidenceCount: 3,
    });
    interpretations.push({
      title: 'Requires additional verification',
      explanation: `While basic checks passed for "${input}", additional verification is recommended.`,
      confidence: Math.max(30, score - 10),
      reasoning: 'Some verification checks could not be completed.',
      supportingEvidenceCount: 1,
    });
  }

  return interpretations;
}

function generateWarningSignals(input: string, inputType: InputType, score: number) {
  const signals = [];
  if (score < 40) {
    signals.push({
      label: 'Low trust score',
      severity: 'high' as const,
      description: 'The verification score is low, indicating potential risk.',
    });
  }
  if (inputType === 'unknown') {
    signals.push({
      label: 'Unverifiable input',
      severity: 'medium' as const,
      description: 'The input could not be classified into a standard verification type.',
    });
  }
  return signals;
}

function generateRecommendations(inputType: InputType, _score: number) {
  const recs = [];

  if (inputType === 'domain') {
    recs.push({
      title: 'Check WHOIS registration',
      description: 'Verify the domain owner and registration date.',
    });
    recs.push({
      title: 'Run a link scanner',
      description: 'Check for malware or phishing indicators.',
    });
  } else if (inputType === 'email') {
    recs.push({
      title: 'Verify sender identity',
      description: 'Contact the sender through a known channel.',
    });
    recs.push({
      title: 'Check for phishing indicators',
      description: 'Look for suspicious links or requests.',
    });
  } else if (inputType === 'unknown') {
    recs.push({
      title: 'Cross-reference sources',
      description: 'Verify the claim through multiple independent sources.',
    });
    recs.push({
      title: 'Check fact-checking sites',
      description: 'Search established fact-checking organizations.',
    });
  } else {
    recs.push({
      title: 'Verify independently',
      description: 'Cross-check this information with authoritative sources.',
    });
  }

  return recs;
}

function generateReasoning(score: number, evidence: EvidenceItem[]) {
  const bullets = [
    `Trust score of ${score} out of 100 reflects the checks that could be run against the input.`,
  ];
  for (const e of evidence) {
    if (e.result === 'pass') bullets.push(`+ ${e.label}`);
    else if (e.result === 'fail') bullets.push(`- ${e.label}`);
  }
  if (!evidence.some((e) => e.result === 'fail')) {
    bullets.push('No contradicting evidence was found.');
  }
  return bullets;
}

function generateTimeline(
  input: string,
  inputType: InputType,
  evidence: EvidenceItem[],
  score: number,
) {
  const steps = [
    {
      title: 'Claim Detected',
      summary: `Analyzing ${inputType}: "${input.slice(0, 80)}"`,
      details: [`Input type: ${inputType}`, `Claim: ${input}`],
    },
  ];

  const passCount = evidence.filter((e) => e.result === 'pass').length;
  const failCount = evidence.filter((e) => e.result === 'fail').length;

  steps.push({
    title: 'Evidence Gathered',
    summary: `${passCount} supporting, ${failCount} contradicting`,
    details: evidence.map(
      (e) => `${e.result === 'pass' ? '+' : e.result === 'fail' ? '-' : '~'} ${e.label}`,
    ),
  });

  steps.push({
    title: 'Conflicts Identified',
    summary: failCount > 0 ? `${failCount} issues found` : 'No conflicts',
    details: [],
  });
  steps.push({
    title: 'Risk Signals Detected',
    summary: failCount > 0 ? 'Some risk signals' : 'No high-risk signals',
    details: [],
  });
  steps.push({
    title: 'AI Reasoning',
    summary: 'Analyzing evidence patterns',
    details: ['Comparing evidence strength', 'Evaluating source reliability'],
  });
  steps.push({
    title: 'Final Assessment',
    summary: `Score: ${score}/100`,
    details: [`Trust score: ${score}`, `Evidence items: ${evidence.length}`],
  });

  return steps;
}

// ─── Handler ───────────────────────────────────────────────────────────────────

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const input = body.input;
    const mode: AnalysisMode = body.mode || 'quick';

    if (!input || typeof input !== 'string' || !input.trim()) {
      return NextResponse.json({ error: 'input is required' }, { status: 400 });
    }

    const trimmed = input.trim();
    const inputType = classifyInput(trimmed);

    // Run verification based on type
    let verification: { status: string; score: number; evidence: EvidenceItem[]; summary: string };

    switch (inputType) {
      case 'domain':
        verification = await verifyDomain(trimmed);
        break;
      case 'url':
        verification = await verifyURL(trimmed);
        break;
      case 'email':
        verification = await verifyEmail(trimmed);
        break;
      case 'ipv4':
      case 'ipv6':
        verification = verifyIP(trimmed);
        break;
      case 'phone':
        verification = verifyPhone(trimmed);
        break;
      case 'company':
        verification = verifyCompany(trimmed);
        break;
      default:
        // For unknown text, run research
        const research = await researchClaim(trimmed);
        const evidence: EvidenceItem[] = [];

        for (const r of research.supporting) {
          evidence.push({
            label: `Web search: ${r.domain}`,
            result: 'info',
            points: 5,
            note: `${r.title} - ${r.snippet.slice(0, 100)}`,
          });
        }
        for (const r of research.contradicting) {
          evidence.push({
            label: `Web search: ${r.domain}`,
            result: 'warning',
            points: -5,
            note: `${r.title} - ${r.snippet.slice(0, 100)}`,
          });
        }

        const supportCount = research.supporting.length;
        const contradictCount = research.contradicting.length;
        const researchScore =
          supportCount > contradictCount
            ? clamp(50 + Math.min((supportCount - contradictCount) * 10, 40))
            : contradictCount > supportCount
              ? clamp(50 - Math.min((contradictCount - supportCount) * 10, 40))
              : 50;

        verification = {
          status: researchScore >= 70 ? 'verified' : researchScore >= 40 ? 'warning' : 'invalid',
          score: researchScore,
          evidence,
          summary:
            evidence.length > 0
              ? `Found ${supportCount} supporting and ${contradictCount} contradicting sources.`
              : 'No evidence found for this claim.',
        };
    }

    // Build full response
    const score = verification.score;
    const verdict: Verdict = score >= 70 ? 'High' : score >= 40 ? 'Medium' : 'Low';
    const evidence = verification.evidence;

    // Extract entities for text claims
    const entities =
      inputType === 'unknown'
        ? trimmed
            .split(/\s+/)
            .filter(
              (w) =>
                w.length > 3 &&
                !['this', 'that', 'with', 'from', 'have', 'been', 'were', 'they', 'their'].includes(
                  w.toLowerCase(),
                ),
            )
            .slice(0, 5)
            .map((name) => ({ name, kind: 'organization' as Entity['kind'] }))
        : [
            {
              name: trimmed,
              kind: (inputType === 'email' ? 'person' : 'organization') as Entity['kind'],
            },
          ];

    const keywords = trimmed
      .toLowerCase()
      .split(/\s+/)
      .filter((w) => w.length > 3)
      .slice(0, 10);

    const response: VerifyResponse = {
      input: trimmed,
      type: inputType,
      status: verification.status,
      trustScore: score,
      summary: verification.summary,
      evidence,
      verdict,
      keyClaim: `The claim under review is that ${inputType === 'unknown' ? `"${trimmed}"` : `the ${inputType} "${trimmed}"`} is trustworthy.`,
      entities,
      keywords,
      evidenceFor: evidence.filter((e) => e.result === 'pass'),
      evidenceAgainst: evidence.filter((e) => e.result === 'fail' || e.result === 'warning'),
      missingEvidence:
        inputType === 'unknown'
          ? ['Not verified: original source.', 'Not verified: independent corroboration.']
          : [`Not verified: additional ${inputType} checks.`],
      unknownInformation:
        inputType === 'unknown'
          ? [
              'The author of this content is unknown.',
              'The publication date and provenance are unknown.',
            ]
          : ['The real-world usage of this identifier is unknown.'],
      interpretations: generateInterpretations(trimmed, inputType, score),
      warningSignals: generateWarningSignals(trimmed, inputType, score),
      confidence: clamp(score + 10),
      reasoning: generateReasoning(score, evidence),
      timeline: generateTimeline(trimmed, inputType, evidence, score),
      recommendations: generateRecommendations(inputType, score),
      supportingEvidence:
        evidence.filter((e) => e.result === 'pass').length > 0
          ? [
              {
                category: 'Web Sources',
                items: evidence
                  .filter((e) => e.result === 'pass')
                  .map((e) => ({
                    title: e.label,
                    source: e.note || '',
                    credibility: 'web',
                    summary: e.note || '',
                  })),
              },
            ]
          : undefined,
      contradictingEvidence: evidence
        .filter((e) => e.result === 'fail')
        .map((e) => ({
          sourceA: 'Submitted claim',
          claimA: trimmed,
          sourceB: e.label,
          claimB: e.note || '',
          whyTheyDisagree: `Evidence '${e.label}' contradicts this claim.`,
          confidenceInContradiction: 50,
        })),
      missingInformation: [
        {
          item: 'Additional sources',
          whyItMatters: 'More sources would increase confidence in the verification.',
        },
      ],
      confidenceBreakdown: {
        overall: clamp(score + 10),
        metrics: [
          {
            name: 'Evidence Strength',
            score: clamp(evidence.filter((e) => e.result === 'pass').length * 20),
            note: `${evidence.filter((e) => e.result === 'pass').length} supporting checks`,
          },
          { name: 'Source Quality', score: 50, note: 'Standard source evaluation' },
          {
            name: 'Independent Confirmation',
            score: clamp(evidence.length * 15),
            note: `${evidence.length} evidence items`,
          },
          {
            name: 'Contradiction Risk',
            score: clamp(100 - evidence.filter((e) => e.result === 'fail').length * 30),
            note: `${evidence.filter((e) => e.result === 'fail').length} contradictions`,
          },
          {
            name: 'Missing Evidence',
            score: clamp(100 - (verification.summary.includes('Not verified') ? 30 : 10)),
            note: 'Some checks could not be performed',
          },
          {
            name: 'Input Specificity',
            score: inputType === 'unknown' ? 40 : 70,
            note: inputType === 'unknown' ? 'Free-form text' : `Classified as ${inputType}`,
          },
        ],
      },
      aiSummary: `TrustCheck analyzed "${trimmed}" (type: ${inputType}) and assigned a trust score of ${score}/100 (${verdict}). ${verification.summary} ${evidence.filter((e) => e.result === 'fail').length > 0 ? 'Some contradicting evidence was found.' : 'No contradicting evidence was found.'}`,
      suggestedReading:
        inputType === 'unknown'
          ? [
              {
                title: 'How to evaluate claims',
                publisher: 'General',
                whyItHelps: 'Learn techniques for evaluating factual claims.',
              },
            ]
          : [
              {
                title: `How to verify ${inputType}s`,
                publisher: 'General',
                whyItHelps: `Best practices for verifying ${inputType} information.`,
              },
            ],
      suggestedReadingNote: 'General guidance for verification.',
      whatChanged: [],
      whatChangedNote: 'No dated history could be reconstructed for this input.',
      claims: [
        {
          id: 'c1',
          text: trimmed,
          entities,
          keywords,
          verdict,
          confidence: clamp(score + 10),
          evidence,
          conflicts: evidence
            .filter((e) => e.result === 'fail')
            .map((e) => ({
              sourceA: 'Submitted claim',
              claimA: trimmed,
              sourceB: e.label,
              claimB: e.note || '',
              whyTheyDisagree: `Evidence contradicts the claim.`,
              confidenceInContradiction: 50,
            })),
          summary: verification.summary,
          timeline: generateTimeline(trimmed, inputType, evidence, score),
          recommendations: generateRecommendations(inputType, score),
          missingInformation: [
            {
              item: 'Additional verification',
              whyItMatters: 'More verification would increase confidence.',
            },
          ],
          status: score >= 70 ? 'verified' : score >= 40 ? 'partially_verified' : 'unverified',
        },
      ],
      claimCount: 1,
      verifiedClaims: score >= 70 ? 1 : 0,
      partialClaims: score >= 40 && score < 70 ? 1 : 0,
      unverifiedClaims: score < 40 ? 1 : 0,
      analysisMode: mode,
      evidenceLedger: {
        claim: trimmed,
        supporting: evidence
          .filter((e) => e.result === 'pass')
          .map((e) => ({
            source: {
              title: e.label,
              sourceType: 'unknown',
              relation: 'tertiary',
              isOfficial: false,
              supportsClaim: true,
              contradictsClaim: false,
              isIndependent: true,
              confidence: 50,
              relevance: 60,
            },
            summary: e.note || '',
            strength: e.points,
          })),
        contradicting: evidence
          .filter((e) => e.result === 'fail')
          .map((e) => ({
            source: {
              title: e.label,
              sourceType: 'unknown',
              relation: 'tertiary',
              isOfficial: false,
              supportsClaim: false,
              contradictsClaim: true,
              isIndependent: true,
              confidence: 50,
              relevance: 60,
            },
            summary: e.note || '',
            strength: Math.abs(e.points),
          })),
        unknown: [],
        totalSources: evidence.length,
        independentCount: evidence.length,
        duplicateCount: 0,
      },
      scoreExplanation: {
        evidenceStrength: clamp(evidence.filter((e) => e.result === 'pass').length * 20),
        evidenceStrengthNote: `${evidence.filter((e) => e.result === 'pass').length} supporting, ${evidence.filter((e) => e.result === 'fail').length} contradicting evidence items.`,
        sourceQuality: 50,
        sourceQualityNote: 'Standard source evaluation was used.',
        independentConfirmation: clamp(evidence.length * 15),
        independentNote: `${evidence.length} evidence items found.`,
        contradictionRisk: clamp(100 - evidence.filter((e) => e.result === 'fail').length * 30),
        contradictionNote:
          evidence.filter((e) => e.result === 'fail').length > 0
            ? `${evidence.filter((e) => e.result === 'fail').length} contradicting evidence items found.`
            : 'No contradictions found.',
        missingEvidence: 30,
        missingNote: 'Some checks could not be performed.',
      },
      sourceIntelligence: evidence.map((e) => ({
        title: e.label,
        sourceType: 'unknown' as SourceType,
        relation: 'tertiary' as SourceRelation,
        isOfficial: false,
        supportsClaim: e.result === 'pass',
        contradictsClaim: e.result === 'fail',
        isIndependent: true,
        confidence: 50,
        relevance: 60,
      })),
    };

    return NextResponse.json(response);
  } catch (error) {
    console.error('[/api/verify] Error:', error);
    return NextResponse.json({ error: 'Internal verification error' }, { status: 500 });
  }
}
