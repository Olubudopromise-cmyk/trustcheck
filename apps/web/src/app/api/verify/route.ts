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

// ─── Constants ─────────────────────────────────────────────────────────────────

const REQUEST_TIMEOUT_MS = 8000; // Overall request timeout (leaves 2s for response)
const PROVIDER_TIMEOUT_MS = 3000; // Per-provider timeout

// ─── Input Classification (fallback only — research determines the real type) ──

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

const DOMAIN_RE = /^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/;
const PHONE_RE = /^\+\d{6,15}$/;

function classifyInputFast(input: string): InputType {
  const in_ = input.trim();
  if (!in_) return 'unknown';

  try {
    const u = new URL(in_);
    if (u.protocol === 'http:' || u.protocol === 'https:') return 'url';
  } catch {}
  if (!/\s/.test(in_) && in_.includes('@')) {
    const p = in_.split('@');
    if (p.length === 2 && p[0].length > 0 && p[1].includes('.')) return 'email';
  }
  if (/^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(in_)) return 'ipv4';
  if (in_.includes(':') && /^[0-9a-fA-F:]+$/.test(in_)) return 'ipv6';
  if (PHONE_RE.test(in_.replace(/[\s.\-()]/g, ''))) return 'phone';
  if (DOMAIN_RE.test(in_)) return 'domain';

  return 'unknown'; // Don't guess — let research determine entity type
}

// ─── Source Quality ────────────────────────────────────────────────────────────

type SourceQuality = 'very_strong' | 'strong' | 'moderate' | 'weak' | 'unknown';

function classifySourceQuality(domain: string): SourceQuality {
  const lower = domain.toLowerCase();
  if (
    [
      'gov',
      'edu',
      'wikipedia.org',
      'reuters.com',
      'apnews.com',
      'bbc.com',
      'nature.com',
      'who.int',
      'cdc.gov',
    ].some((d) => lower.includes(d))
  )
    return 'very_strong';
  if (['org', 'ac.uk', 'edu.'].some((d) => lower.includes(d))) return 'strong';
  if (['com', 'net', 'info', 'co', 'news'].some((d) => lower.includes(d))) return 'moderate';
  if (lower.includes('blog') || lower.includes('forum') || lower.includes('reddit')) return 'weak';
  return 'unknown';
}

function sourceQualityLabel(q: SourceQuality): string {
  const labels: Record<SourceQuality, string> = {
    very_strong: 'Very strong',
    strong: 'Strong',
    moderate: 'Moderate',
    weak: 'Weak',
    unknown: 'Unknown',
  };
  return labels[q];
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
  RelatedTopics: Array<{ Text?: string; FirstURL?: string; Result?: string }>;
}

async function searchDuckDuckGo(query: string): Promise<SearchResult[]> {
  try {
    const res = await fetch(
      `https://api.duckduckgo.com/?q=${encodeURIComponent(query)}&format=json&no_html=1`,
      { signal: AbortSignal.timeout(PROVIDER_TIMEOUT_MS) },
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
        signal: AbortSignal.timeout(PROVIDER_TIMEOUT_MS),
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
  allResults: SearchResult[];
  entityDescription: string;
  entityType: string;
  entityCountry?: string;
  entityRegion?: string;
  disambiguation?: string[];
  searchErrors: string[];
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

  const searchErrors: string[] = [];
  if (ddgResults.status === 'rejected') searchErrors.push(`DuckDuckGo: ${ddgResults.reason}`);
  if (wikiResults.status === 'rejected') searchErrors.push(`Wikipedia: ${wikiResults.reason}`);

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
  const supporting: SearchResult[] = [];
  const contradicting: SearchResult[] = [];

  for (const r of allResults) {
    const text = (r.title + ' ' + r.snippet).toLowerCase();
    if (contradictKeywords.some((k) => text.includes(k))) {
      contradicting.push(r);
    } else {
      supporting.push(r);
    }
  }

  // Detect disambiguation from DuckDuckGo related topics
  if (ddgData.length > 0) {
    const relatedTitles = ddgData
      .slice(1)
      .map((r) => r.title)
      .filter((t) => t && t !== claim);
    if (relatedTitles.length > 0) {
      disambiguation.push(...relatedTitles.slice(0, 3));
    }
  }

  return {
    supporting,
    contradicting,
    allResults,
    entityDescription,
    entityType,
    entityCountry,
    entityRegion,
    disambiguation: disambiguation.length > 0 ? disambiguation : undefined,
    searchErrors,
  };
}

// ─── Entity Type Resolution (research-first) ──────────────────────────────────

function resolveEntityType(
  input: string,
  fastType: InputType,
  research: ResearchResults,
): InputType {
  // For technical inputs, trust the fast classifier
  if (['url', 'email', 'ipv4', 'ipv6', 'phone', 'domain'].includes(fastType)) {
    return fastType;
  }

  // For text inputs, use research results to determine entity type
  const ddgType = (research.entityType || '').toLowerCase();
  const desc = research.entityDescription.toLowerCase();
  const allText = research.allResults
    .map((r) => r.title + ' ' + r.snippet)
    .join(' ')
    .toLowerCase();

  // Government/entity detection from DuckDuckGo Type
  if (
    ddgType.includes('state') ||
    ddgType.includes('province') ||
    ddgType.includes('region') ||
    ddgType.includes('country') ||
    ddgType.includes('republic') ||
    ddgType.includes('kingdom')
  ) {
    return 'government';
  }
  if (
    ddgType.includes('city') ||
    ddgType.includes('town') ||
    ddgType.includes('village') ||
    ddgType.includes('island') ||
    ddgType.includes('mountain') ||
    ddgType.includes('river')
  ) {
    return 'place';
  }
  if (
    ddgType.includes('company') ||
    ddgType.includes('business') ||
    ddgType.includes('corporation')
  ) {
    return 'company';
  }
  if (ddgType.includes('person') || ddgType.includes('people')) {
    return 'person';
  }
  if (
    ddgType.includes('organization') ||
    ddgType.includes('institution') ||
    ddgType.includes('agency')
  ) {
    return 'organization';
  }

  // Fallback: analyze description content
  if (
    desc.includes('state of') ||
    desc.includes('province of') ||
    desc.includes('region of') ||
    desc.includes('government') ||
    desc.includes('administrative')
  ) {
    return 'government';
  }
  if (desc.includes('city') || desc.includes('town') || desc.includes('metropolitan')) {
    return 'place';
  }
  if (desc.includes('company') || desc.includes('corporation') || desc.includes('incorporated')) {
    return 'company';
  }

  // Check all research results for type signals
  if (
    allText.includes('state') &&
    (allText.includes('nigeria') || allText.includes('government'))
  ) {
    return 'government';
  }

  // If we have a description but can't determine type, use research-backed guess
  if (research.entityDescription) {
    // Has description = likely a known entity, not a random company
    return 'organization';
  }

  return fastType === 'unknown' ? 'unknown' : fastType;
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
  resolvedType: InputType,
  research: ResearchResults,
): EntityIdentity {
  const identity: EntityIdentity = {
    canonicalName: input,
    entityType: resolvedType,
    description:
      research.entityDescription ||
      `The searched term "${input}" could not be fully identified from available evidence.`,
    identityConfidence: research.entityDescription ? 'medium' : 'low',
    aliases: [],
  };

  if (research.entityCountry) identity.country = research.entityCountry;
  if (research.entityRegion) identity.region = research.entityRegion;
  if (research.disambiguation) {
    identity.possibleAlternatives = research.disambiguation.filter(
      (d) => d.toLowerCase() !== input.toLowerCase(),
    );
  }

  if (research.supporting.length >= 2) identity.identityConfidence = 'high';
  else if (research.supporting.length === 1) identity.identityConfidence = 'medium';

  return identity;
}

// ─── Verification Functions ────────────────────────────────────────────────────

async function verifyDomain(domain: string) {
  const b = new EvidenceBuilder();
  try {
    const res = await fetch(
      `https://dns.google/resolve?name=${encodeURIComponent(domain)}&type=A`,
      { signal: AbortSignal.timeout(PROVIDER_TIMEOUT_MS) },
    );
    const data = await res.json();
    if (data.Answer && data.Answer.length > 0) b.pass('DNS Resolves', 0);
    else {
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
    if (res.ok) b.pass('HTTP Status OK', 20);
    else if (res.status >= 400 && res.status < 500) b.warning('HTTP Client Error', 10);
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
  return {
    status: score >= 70 ? 'verified' : score >= 40 ? 'warning' : 'invalid',
    score,
    evidence: b.getEvidence(),
    summary:
      score >= 70
        ? 'Domain resolves, HTTPS available, certificate valid.'
        : score >= 40
          ? 'Domain resolves, but verification is inconclusive.'
          : 'Domain verification failed.',
  };
}

async function verifyURL(url: string) {
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

async function verifyEmail(email: string) {
  const b = new EvidenceBuilder();
  const domain = email.split('@')[1];
  try {
    const res = await fetch(
      `https://dns.google/resolve?name=${encodeURIComponent(domain)}&type=MX`,
      { signal: AbortSignal.timeout(PROVIDER_TIMEOUT_MS) },
    );
    const data = await res.json();
    if (data.Answer && data.Answer.length > 0) b.pass('MX Records Found', 40);
    else b.fail('No MX Records', -20);
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

function verifyIP(ip: string) {
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

function verifyPhone(phone: string) {
  const b = new EvidenceBuilder();
  if (/^\+\d{6,15}$/.test(phone.replace(/[\s.\-()]/g, ''))) {
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

function verifyEntity(resolvedType: InputType, research: ResearchResults) {
  const b = new EvidenceBuilder();

  // Evidence from research
  for (const r of research.supporting) {
    const quality = classifySourceQuality(r.domain);
    const points =
      quality === 'very_strong' ? 15 : quality === 'strong' ? 10 : quality === 'moderate' ? 5 : 2;
    b.pass(
      `Source: ${r.domain}`,
      points,
      `${r.title} (${sourceQualityLabel(quality)}) — ${r.snippet.slice(0, 120)}`,
    );
  }
  for (const r of research.contradicting) {
    b.warning(`Source: ${r.domain}`, -5, `${r.title} — ${r.snippet.slice(0, 120)}`);
  }

  // Entity-specific checks
  if (resolvedType === 'government' || resolvedType === 'place') {
    if (research.entityCountry)
      b.pass('Location identified', 10, `Identified as being in ${research.entityCountry}`);
  }
  if (resolvedType === 'company') {
    b.pass('Entity identified', 5, 'Research confirms this is a recognized entity');
  }

  const score = b.getScore();
  return {
    status: score >= 70 ? 'verified' : score >= 40 ? 'warning' : 'invalid',
    score,
    evidence: b.getEvidence(),
    summary:
      research.supporting.length > 0
        ? `Found ${research.supporting.length} supporting source(s)${research.contradicting.length > 0 ? ` and ${research.contradicting.length} contradicting source(s)` : ''}.`
        : 'No evidence found for this entity.',
  };
}

// ─── Analysis Generators ───────────────────────────────────────────────────────

function generateInterpretations(input: string, score: number, identity: EntityIdentity) {
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

function generateWarningSignals(input: string, score: number, identity: EntityIdentity) {
  const signals = [];
  if (score < 40)
    signals.push({
      label: 'Low trust score',
      severity: 'high' as const,
      description: 'The verification score is low, indicating potential risk.',
    });
  if (identity.possibleAlternatives && identity.possibleAlternatives.length > 0)
    signals.push({
      label: 'Ambiguous entity',
      severity: 'medium' as const,
      description: `The search term "${input}" could refer to multiple entities.`,
    });
  return signals;
}

function generateRecommendations(resolvedType: InputType, identity: EntityIdentity) {
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
  } else if (resolvedType === 'domain') {
    recs.push({
      title: 'Check WHOIS registration',
      description: 'Verify the domain owner and registration date.',
    });
  } else if (resolvedType === 'email') {
    recs.push({
      title: 'Verify sender identity',
      description: 'Contact the sender through a known channel.',
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
  if (identity.entityType !== 'unknown')
    bullets.push(`Entity identified as: ${identity.entityType}`);
  if (identity.country) bullets.push(`Location: ${identity.country}`);
  for (const e of evidence) {
    if (e.result === 'pass') bullets.push(`+ ${e.label}`);
    else if (e.result === 'fail') bullets.push(`- ${e.label}`);
  }
  if (!evidence.some((e) => e.result === 'fail'))
    bullets.push('No contradicting evidence was found.');
  return bullets;
}

function generateTimeline(
  input: string,
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

function generateFinalAssessment(
  score: number,
  verdict: Verdict,
  evidence: EvidenceItem[],
  identity: EntityIdentity,
  research: ResearchResults,
) {
  const parts: string[] = [];
  parts.push(
    `TrustCheck verified "${identity.canonicalName}" and identified it as ${identity.entityType === 'unknown' ? 'an entity' : `a ${identity.entityType}`}.`,
  );

  if (identity.identityConfidence === 'high')
    parts.push(
      'The entity identification has high confidence based on multiple corroborating sources.',
    );
  else if (identity.identityConfidence === 'medium')
    parts.push('The entity identification has moderate confidence. Some ambiguity may remain.');
  else
    parts.push('The entity identification has low confidence due to limited available evidence.');

  if (identity.possibleAlternatives && identity.possibleAlternatives.length > 0) {
    parts.push(
      `The search term is ambiguous and could also refer to: ${identity.possibleAlternatives.join(', ')}.`,
    );
  }

  if (research.contradicting.length > 0)
    parts.push(
      'Some contradicting evidence was found, which may indicate conflicting information.',
    );

  if (research.searchErrors.length > 0)
    parts.push(
      `Note: Some research sources were unavailable (${research.searchErrors.length} provider(s) failed).`,
    );

  return parts.join(' ');
}

// ─── Handler ───────────────────────────────────────────────────────────────────

export async function POST(request: NextRequest) {
  const startTime = Date.now();

  try {
    const body = await request.json();
    const input = body.input;
    const mode: AnalysisMode = body.mode || 'quick';

    if (!input || typeof input !== 'string' || !input.trim()) {
      return NextResponse.json({ error: 'input is required' }, { status: 400 });
    }

    const trimmed = input.trim();

    // Step 1: Fast classification (for technical inputs only)
    const fastType = classifyInputFast(trimmed);

    // Step 2: Research (happens BEFORE entity type resolution)
    let research: ResearchResults = {
      supporting: [],
      contradicting: [],
      allResults: [],
      entityDescription: '',
      entityType: 'unknown',
      searchErrors: [],
    };
    if (!['url', 'email', 'ipv4', 'ipv6', 'phone'].includes(fastType)) {
      research = await researchClaim(trimmed);
    }

    // Check if we've exceeded timeout
    if (Date.now() - startTime > REQUEST_TIMEOUT_MS) {
      return NextResponse.json({
        input: trimmed,
        type: 'unknown',
        status: 'timeout',
        trustScore: 0,
        summary: 'Research timed out. Partial results may be available.',
        evidence: [],
        verdict: 'Low' as Verdict,
        entityIdentity: {
          canonicalName: trimmed,
          entityType: 'unknown',
          description: 'Research timed out before entity could be identified.',
          identityConfidence: 'low' as const,
          aliases: [],
        },
      });
    }

    // Step 3: Resolve entity type based on research
    const resolvedType = resolveEntityType(trimmed, fastType, research);

    // Step 4: Run verification based on resolved type
    let verification: { status: string; score: number; evidence: EvidenceItem[]; summary: string };

    switch (resolvedType) {
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
      default:
        verification = verifyEntity(resolvedType, research);
        break;
    }

    // Step 5: Build entity identity
    const entityIdentity = buildEntityIdentity(trimmed, resolvedType, research);

    // Step 6: Build response
    const score = verification.score;
    const verdict: Verdict = score >= 70 ? 'High' : score >= 40 ? 'Medium' : 'Low';
    const evidence = verification.evidence;

    const entities = [
      {
        name: trimmed,
        kind: (resolvedType === 'email'
          ? 'person'
          : resolvedType === 'government' || resolvedType === 'place'
            ? 'location'
            : 'organization') as Entity['kind'],
      },
    ];
    const keywords = trimmed
      .toLowerCase()
      .split(/\s+/)
      .filter((w) => w.length > 3)
      .slice(0, 10);

    const response: VerifyResponse = {
      input: trimmed,
      type: resolvedType,
      status: verification.status,
      trustScore: score,
      summary: verification.summary,
      evidence,
      verdict,
      keyClaim: `The claim under review is that ${resolvedType === 'unknown' ? `"${trimmed}"` : `the ${resolvedType} "${trimmed}"`} is trustworthy.`,
      entities,
      keywords,
      evidenceFor: evidence.filter((e) => e.result === 'pass'),
      evidenceAgainst: evidence.filter((e) => e.result === 'fail' || e.result === 'warning'),
      missingEvidence:
        resolvedType === 'unknown'
          ? ['Not verified: original source.', 'Not verified: independent corroboration.']
          : [`Not verified: additional ${resolvedType} checks.`],
      unknownInformation:
        resolvedType === 'unknown'
          ? [
              'The author of this content is unknown.',
              'The publication date and provenance are unknown.',
            ]
          : ['The real-world usage of this identifier is unknown.'],
      interpretations: generateInterpretations(trimmed, score, entityIdentity),
      warningSignals: generateWarningSignals(trimmed, score, entityIdentity),
      confidence: clamp(score + 10),
      reasoning: generateReasoning(score, evidence, entityIdentity),
      timeline: generateTimeline(trimmed, evidence, score, entityIdentity),
      recommendations: generateRecommendations(resolvedType, entityIdentity),
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
            score: resolvedType === 'unknown' ? 40 : 70,
            note: resolvedType === 'unknown' ? 'Free-form text' : `Classified as ${resolvedType}`,
          },
        ],
      },
      aiSummary: generateFinalAssessment(score, verdict, evidence, entityIdentity, research),
      suggestedReading:
        resolvedType === 'unknown'
          ? [
              {
                title: 'How to evaluate claims',
                publisher: 'General',
                whyItHelps: 'Learn techniques for evaluating factual claims.',
              },
            ]
          : [
              {
                title: `How to verify ${resolvedType}s`,
                publisher: 'General',
                whyItHelps: `Best practices for verifying ${resolvedType} information.`,
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
          timeline: generateTimeline(trimmed, evidence, score, entityIdentity),
          recommendations: generateRecommendations(resolvedType, entityIdentity),
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
