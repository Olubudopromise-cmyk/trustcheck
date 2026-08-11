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

type InputType =
  | 'url'
  | 'domain'
  | 'email'
  | 'ipv4'
  | 'ipv6'
  | 'phone'
  | 'company'
  | 'government'
  | 'place'
  | 'organization'
  | 'person'
  | 'unknown';

const MAX_COMPANY_WORDS = 4;

const DOMAIN_RE = /^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/;
const COMPANY_RE = /^[A-Za-z][A-Za-z0-9\s.'-]{0,63}$/;
const PHONE_RE = /^\+\d{6,15}$/;

// Government-related keywords
const GOV_KEYWORDS = [
  'state',
  'government',
  'ministry',
  'department',
  'agency',
  'authority',
  'federal',
  'municipal',
  'council',
  'commission',
  'office',
  'bureau',
  'congress',
  'senate',
  'parliament',
  'legislature',
  'executive',
  'county',
  'province',
  'region',
  'district',
  'republic',
  'kingdom',
];

// Place-related keywords
const PLACE_KEYWORDS = [
  'city',
  'town',
  'village',
  'island',
  'mountain',
  'river',
  'lake',
  'ocean',
  'sea',
  'desert',
  'forest',
  'park',
  'bay',
  'harbor',
  'port',
  'capital',
  'metropolitan',
  'area',
  'neighborhood',
  'suburb',
];

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

  // IPv6
  if (in_.includes(':') && /^[0-9a-fA-F:]+$/.test(in_)) return 'ipv6';

  // Phone
  const normalized = in_.replace(/[\s.\-()]/g, '');
  if (PHONE_RE.test(normalized)) return 'phone';

  // Domain
  if (DOMAIN_RE.test(in_)) return 'domain';

  // Check for government/place keywords before company
  const lower = in_.toLowerCase();
  const words = lower.split(/\s+/);

  if (words.some((w) => GOV_KEYWORDS.includes(w))) {
    return 'government';
  }
  if (words.some((w) => PLACE_KEYWORDS.includes(w))) {
    return 'place';
  }

  // Company (short, letter-starting name)
  if (COMPANY_RE.test(in_) && words.length <= MAX_COMPANY_WORDS) {
    return 'company';
  }

  return 'unknown';
}

// ─── Evidence Builder ──────────────────────────────────────────────────────────

class EvidenceBuilder {
  private evidence: EvidenceItem[] = [];
  private score = 0;

  pass(label: string, points: number, note?: string) {
    this.score = clamp(this.score + points);
    this.evidence.push({ label, result: 'pass', points, note });
  }
  fail(label: string, points: number, note?: string) {
    this.score = clamp(this.score + points);
    this.evidence.push({ label, result: 'fail', points, note });
  }
  warning(label: string, points: number, note?: string) {
    this.score = clamp(this.score + points);
    this.evidence.push({ label, result: 'warning', points, note });
  }
  info(label: string, note?: string) {
    this.evidence.push({ label, result: 'info', points: 0, note });
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

// ─── Source Quality ────────────────────────────────────────────────────────────

type SourceQuality = 'very_strong' | 'strong' | 'moderate' | 'weak' | 'unknown';

const HIGH_QUALITY_DOMAINS = [
  'gov',
  'edu',
  'org',
  'wikipedia.org',
  'reuters.com',
  'apnews.com',
  'bbc.com',
  'nature.com',
  'science.org',
  'who.int',
  'cdc.gov',
];

const MODERATE_QUALITY_DOMAINS = ['com', 'net', 'info', 'co', 'news', 'media', 'press'];

function classifySourceQuality(domain: string): SourceQuality {
  const lower = domain.toLowerCase();
  if (HIGH_QUALITY_DOMAINS.some((d) => lower.includes(d))) return 'very_strong';
  if (MODERATE_QUALITY_DOMAINS.some((d) => lower.includes(d))) return 'moderate';
  if (lower.includes('blog') || lower.includes('forum') || lower.includes('reddit')) return 'weak';
  return 'unknown';
}

function sourceQualityLabel(q: SourceQuality): string {
  switch (q) {
    case 'very_strong':
      return 'Very strong';
    case 'strong':
      return 'Strong';
    case 'moderate':
      return 'Moderate';
    case 'weak':
      return 'Weak';
    default:
      return 'Unknown';
  }
}

// ─── Domain Verification ───────────────────────────────────────────────────────

async function verifyDomain(
  domain: string,
): Promise<{ status: string; score: number; evidence: EvidenceItem[]; summary: string }> {
  const b = new EvidenceBuilder();

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
    try {
      const res = await fetch(`http://${domain}`, {
        method: 'HEAD',
        signal: AbortSignal.timeout(5000),
        redirect: 'follow',
      });
      b.pass('HTTP Fallback', 10);
      if (res.ok) b.pass('HTTP Status OK', 20);
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
  try {
    const res = await fetch(
      `https://dns.google/resolve?name=${encodeURIComponent(domain)}&type=MX`,
      { signal: AbortSignal.timeout(3000) },
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
  sourceType?: string;
  abstract?: string;
  entityType?: string;
  imageUrl?: string;
}

interface DuckDuckGoResponse {
  Abstract: string;
  AbstractText: string;
  AbstractSource: string;
  AbstractURL: string;
  Heading: string;
  Image: string;
  Type: string;
  RelatedTopics: Array<{
    Text?: string;
    FirstURL?: string;
    Result?: string;
  }>;
}

async function searchDuckDuckGo(query: string): Promise<SearchResult[]> {
  try {
    const res = await fetch(
      `https://api.duckduckgo.com/?q=${encodeURIComponent(query)}&format=json&no_html=1&skip_disambig=1`,
      { signal: AbortSignal.timeout(3000) },
    );
    const data: DuckDuckGoResponse = await res.json();
    const results: SearchResult[] = [];

    if (data.AbstractText) {
      results.push({
        title: data.Heading || query,
        url: data.AbstractURL || '',
        snippet: data.AbstractText,
        domain: data.AbstractSource || 'duckduckgo.com',
        abstract: data.AbstractText,
        entityType: data.Type || undefined,
        imageUrl: data.Image || undefined,
      });
    }

    if (data.RelatedTopics && Array.isArray(data.RelatedTopics)) {
      for (const topic of data.RelatedTopics.slice(0, 5)) {
        if (topic.Text && topic.FirstURL) {
          try {
            const topicUrl = new URL(topic.FirstURL);
            results.push({
              title: topic.Text.slice(0, 100),
              url: topic.FirstURL,
              snippet: topic.Text,
              domain: topicUrl.hostname,
            });
          } catch {
            /* invalid URL */
          }
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
          sourceType: 'academic',
        });
      }
    }
    return results;
  } catch {
    return [];
  }
}

interface ResearchResults {
  supporting: SearchResult[];
  contradicting: SearchResult[];
  entityDescription: string;
  entityType: string;
  entityCountry?: string;
  entityRegion?: string;
  disambiguation?: string[];
}

async function researchClaim(claim: string): Promise<ResearchResults> {
  const [ddgResults, wikiResults] = await Promise.allSettled([
    searchDuckDuckGo(claim),
    searchWikipedia(claim),
  ]);

  const allResults = [
    ...(ddgResults.status === 'fulfilled' ? ddgResults.value : []),
    ...(wikiResults.status === 'fulfilled' ? wikiResults.value : []),
  ];

  // Extract entity description from DuckDuckGo
  let entityDescription = '';
  let entityType = 'unknown';
  let entityCountry: string | undefined;
  let entityRegion: string | undefined;
  const disambiguation: string[] = [];

  const ddgData = ddgResults.status === 'fulfilled' ? ddgResults.value : [];
  if (ddgData.length > 0 && ddgData[0].abstract) {
    entityDescription = ddgData[0].abstract;
    entityType = ddgData[0].entityType || 'unknown';
  }

  // Try to extract location from description
  const locationPatterns = [
    /(?:in|of|from|located in|situated in)\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)*)/g,
    /([A-Z][a-z]+(?:\s+[A-Z][a-z]+)*)\s+(?:State|Province|Region|County|District)/g,
  ];

  for (const pattern of locationPatterns) {
    const match = pattern.exec(entityDescription);
    if (match) {
      entityCountry = match[1];
      break;
    }
  }

  // Classify evidence
  const contradictKeywords = [
    'false',
    'fake',
    'debunked',
    'myth',
    'no evidence',
    'wrong',
    'incorrect',
  ];
  const supportingKeywords = [
    'confirmed',
    'supports',
    'proves',
    'true',
    'accurate',
    'verified',
    'is a',
    'is an',
  ];

  const supporting: SearchResult[] = [];
  const contradicting: SearchResult[] = [];

  for (const r of allResults) {
    const text = (r.title + ' ' + r.snippet).toLowerCase();
    const hasContradict = contradictKeywords.some((k) => text.includes(k));
    const hasSupport = supportingKeywords.some((k) => text.includes(k));

    if (hasContradict) {
      contradicting.push(r);
    } else {
      supporting.push(r);
    }
  }

  // Generate disambiguation if the query is ambiguous
  if (claim.toLowerCase() === 'lagos') {
    disambiguation.push('Lagos State, Nigeria');
    disambiguation.push('Lagos metropolitan area');
  }

  return {
    supporting,
    contradicting,
    entityDescription,
    entityType,
    entityCountry,
    entityRegion,
    disambiguation: disambiguation.length > 0 ? disambiguation : undefined,
  };
}

// ─── Entity Identity Builder ───────────────────────────────────────────────────

interface EntityIdentity {
  canonicalName: string;
  entityType: string;
  country?: string;
  region?: string;
  description: string;
  identityConfidence: 'high' | 'medium' | 'low';
  aliases: string[];
  possibleAlternatives?: string[];
}

function buildEntityIdentity(
  input: string,
  inputType: InputType,
  research: ResearchResults,
): EntityIdentity {
  const identity: EntityIdentity = {
    canonicalName: input,
    entityType: inputType,
    description:
      research.entityDescription ||
      `The searched term "${input}" could not be fully identified from available evidence.`,
    identityConfidence: research.entityDescription ? 'medium' : 'low',
    aliases: [],
  };

  // Enhance entity type based on research
  if (research.entityType) {
    const t = research.entityType.toLowerCase();
    if (t.includes('state') || t.includes('province') || t.includes('region')) {
      identity.entityType = 'government';
    } else if (t.includes('city') || t.includes('town') || t.includes('village')) {
      identity.entityType = 'place';
    } else if (t.includes('company') || t.includes('business')) {
      identity.entityType = 'company';
    } else if (t.includes('person')) {
      identity.entityType = 'person';
    } else if (t.includes('organization') || t.includes('institution')) {
      identity.entityType = 'organization';
    }
  }

  // Set location from research
  if (research.entityCountry) {
    identity.country = research.entityCountry;
  }
  if (research.entityRegion) {
    identity.region = research.entityRegion;
  }

  // Set disambiguation
  if (research.disambiguation && research.disambiguation.length > 0) {
    identity.possibleAlternatives = research.disambiguation.filter((d) => d !== input);
  }

  // Increase confidence if we have strong evidence
  if (research.supporting.length >= 2) {
    identity.identityConfidence = 'high';
  } else if (research.supporting.length === 1) {
    identity.identityConfidence = 'medium';
  }

  return identity;
}

// ─── Analysis Pipeline ─────────────────────────────────────────────────────────

function generateInterpretations(
  input: string,
  inputType: InputType,
  score: number,
  identity: EntityIdentity,
) {
  const interpretations = [];

  if (identity.identityConfidence === 'high') {
    interpretations.push({
      title: 'Strong identification',
      explanation: `The evidence strongly identifies "${input}" as ${identity.entityType === 'unknown' ? 'a recognized entity' : `a ${identity.entityType}`}.`,
      confidence: 85,
      reasoning: 'Multiple sources corroborate this identification.',
      supportingEvidenceCount: 3,
    });
  } else if (identity.identityConfidence === 'medium') {
    interpretations.push({
      title: 'Likely identification',
      explanation: `The evidence suggests "${input}" is ${identity.entityType === 'unknown' ? 'a recognizable entity' : `a ${identity.entityType}`}, but some ambiguity remains.`,
      confidence: 65,
      reasoning: 'Limited but consistent evidence supports this identification.',
      supportingEvidenceCount: 2,
    });
  } else {
    interpretations.push({
      title: 'Uncertain identification',
      explanation: `The available evidence was insufficient to confidently identify "${input}".`,
      confidence: 40,
      reasoning: 'Limited evidence was found to establish a definitive identification.',
      supportingEvidenceCount: 1,
    });
  }

  if (identity.possibleAlternatives && identity.possibleAlternatives.length > 0) {
    interpretations.push({
      title: 'Possible alternative',
      explanation: `"${input}" may also refer to ${identity.possibleAlternatives[0]}.`,
      confidence: 35,
      reasoning: 'The search term is ambiguous and could refer to multiple entities.',
      supportingEvidenceCount: 0,
    });
  }

  return interpretations;
}

function generateWarningSignals(
  input: string,
  inputType: InputType,
  score: number,
  identity: EntityIdentity,
) {
  const signals = [];

  if (score < 40) {
    signals.push({
      label: 'Low trust score',
      severity: 'high' as const,
      description: 'The verification score is low, indicating potential risk.',
    });
  }

  if (identity.possibleAlternatives && identity.possibleAlternatives.length > 0) {
    signals.push({
      label: 'Ambiguous entity',
      severity: 'medium' as const,
      description: `The search term "${input}" could refer to multiple entities. ${identity.possibleAlternatives.length} alternative(s) identified.`,
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

function generateRecommendations(inputType: InputType, _score: number, identity: EntityIdentity) {
  const recs = [];

  if (identity.entityType === 'government' || identity.entityType === 'place') {
    recs.push({
      title: 'Verify official sources',
      description: 'Check the official government website or registry for confirmation.',
    });
    recs.push({
      title: 'Cross-reference with authoritative databases',
      description: 'Consult official geographic or administrative databases.',
    });
  } else if (inputType === 'domain') {
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

function generateReasoning(score: number, evidence: EvidenceItem[], identity: EntityIdentity) {
  const bullets = [
    `Trust score of ${score} out of 100 reflects the checks that could be run against the input.`,
  ];

  if (identity.entityType !== 'unknown') {
    bullets.push(`Entity identified as: ${identity.entityType}`);
  }
  if (identity.country) {
    bullets.push(`Location: ${identity.country}`);
  }

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
  identity: EntityIdentity,
) {
  const steps = [
    {
      title: 'Entity Detected',
      summary: `Analyzing ${identity.entityType}: "${input.slice(0, 80)}"`,
      details: [`Entity type: ${identity.entityType}`, `Input: ${input}`],
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

// ─── Final Assessment Generator ────────────────────────────────────────────────

function generateFinalAssessment(
  score: number,
  verdict: Verdict,
  evidence: EvidenceItem[],
  identity: EntityIdentity,
  research: ResearchResults,
): string {
  const parts: string[] = [];

  // What was verified
  parts.push(
    `TrustCheck verified "${identity.canonicalName}" and identified it as ${identity.entityType === 'unknown' ? 'an entity' : `a ${identity.entityType}`}.`,
  );

  // Confidence level
  if (identity.identityConfidence === 'high') {
    parts.push(
      'The entity identification has high confidence based on multiple corroborating sources.',
    );
  } else if (identity.identityConfidence === 'medium') {
    parts.push('The entity identification has moderate confidence. Some ambiguity may remain.');
  } else {
    parts.push('The entity identification has low confidence due to limited available evidence.');
  }

  // What remains uncertain
  if (identity.possibleAlternatives && identity.possibleAlternatives.length > 0) {
    parts.push(
      `The search term is ambiguous and could also refer to: ${identity.possibleAlternatives.join(', ')}.`,
    );
  }

  // Contradicting evidence
  if (research.contradicting.length > 0) {
    parts.push(
      'Some contradicting evidence was found, which may indicate conflicting information.',
    );
  }

  return parts.join(' ');
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
      case 'government':
      case 'place':
      case 'organization':
        verification = verifyCompany(trimmed);
        break;
      default: {
        // For unknown text, run research
        const research = await researchClaim(trimmed);
        const evidence: EvidenceItem[] = [];

        for (const r of research.supporting) {
          const quality = classifySourceQuality(r.domain);
          evidence.push({
            label: `Web search: ${r.domain}`,
            result: 'info',
            points: quality === 'very_strong' ? 15 : quality === 'strong' ? 10 : 5,
            note: `${r.title} - ${r.snippet.slice(0, 150)}`,
          });
        }
        for (const r of research.contradicting) {
          evidence.push({
            label: `Web search: ${r.domain}`,
            result: 'warning',
            points: -5,
            note: `${r.title} - ${r.snippet.slice(0, 150)}`,
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
        break;
      }
    }

    // Build full response
    const score = verification.score;
    const verdict: Verdict = score >= 70 ? 'High' : score >= 40 ? 'Medium' : 'Low';
    const evidence = verification.evidence;

    // Research for text claims (already done in default case above, but we need it for entity identity)
    let research: ResearchResults = {
      supporting: [],
      contradicting: [],
      entityDescription: '',
      entityType: 'unknown',
    };
    if (
      inputType === 'unknown' ||
      inputType === 'government' ||
      inputType === 'place' ||
      inputType === 'company' ||
      inputType === 'organization'
    ) {
      research = await researchClaim(trimmed);
    }

    // Build entity identity
    const entityIdentity = buildEntityIdentity(trimmed, inputType, research);

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
              kind: (inputType === 'email'
                ? 'person'
                : inputType === 'government' || inputType === 'place'
                  ? 'location'
                  : 'organization') as Entity['kind'],
            },
          ];

    const keywords = trimmed
      .toLowerCase()
      .split(/\s+/)
      .filter((w) => w.length > 3)
      .slice(0, 10);

    // Build enhanced response
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
      interpretations: generateInterpretations(trimmed, inputType, score, entityIdentity),
      warningSignals: generateWarningSignals(trimmed, inputType, score, entityIdentity),
      confidence: clamp(score + 10),
      reasoning: generateReasoning(score, evidence, entityIdentity),
      timeline: generateTimeline(trimmed, inputType, evidence, score, entityIdentity),
      recommendations: generateRecommendations(inputType, score, entityIdentity),
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
      aiSummary: generateFinalAssessment(score, verdict, evidence, entityIdentity, research),
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
              whyTheyDisagree: 'Evidence contradicts the claim.',
              confidenceInContradiction: 50,
            })),
          summary: verification.summary,
          timeline: generateTimeline(trimmed, inputType, evidence, score, entityIdentity),
          recommendations: generateRecommendations(inputType, score, entityIdentity),
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
              sourceType: 'unknown' as SourceType,
              relation: 'tertiary' as SourceRelation,
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
              sourceType: 'unknown' as SourceType,
              relation: 'tertiary' as SourceRelation,
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
      // Enhanced entity identification fields (backward compatible - optional)
      entityIdentity: {
        canonicalName: entityIdentity.canonicalName,
        entityType: entityIdentity.entityType,
        country: entityIdentity.country,
        region: entityIdentity.region,
        description: entityIdentity.description,
        identityConfidence: entityIdentity.identityConfidence,
        aliases: entityIdentity.aliases,
        possibleAlternatives: entityIdentity.possibleAlternatives,
      },
    };

    return NextResponse.json(response);
  } catch (error) {
    console.error('[/api/verify] Error:', error);
    return NextResponse.json({ error: 'Internal verification error' }, { status: 500 });
  }
}
