import type { VerifyResponse } from '../types';

const FILENAME_FALLBACK = 'report';

const EVIDENCE_ICONS = {
  pass: '\u2713',
  warning: '\u26a0',
  fail: '\u2717',
  info: '\u24d8',
} as const;
const EVIDENCE_COLORS = {
  pass: '#16a34a',
  warning: '#ca8a04',
  fail: '#dc2626',
  info: '#2563eb',
} as const;

// sanitizeFilename strips characters that are invalid in filenames and
// collapses whitespace, keeping safe symbols like "." and "@" so downloads
// look like trustcheck-google.com.json / trustcheck-test@gmail.com.json.
export function sanitizeFilename(input: string): string {
  const cleaned = input
    .normalize('NFKC')
    .replace(/[\\/:*?"<>|\u0000-\u001f\u007f]/g, '')
    .replace(/\s+/g, '-')
    .replace(/[^\w@.\-]/g, '')
    .replace(/-{2,}/g, '-')
    .replace(/^-+|-+$/g, '');
  return cleaned || FILENAME_FALLBACK;
}

// escapeHtml neutralizes dynamic values before they are embedded in the
// printable report so no report can carry stray markup.
function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

export function formatReportDate(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

// buildJSON appends the generation timestamp to the result payload.
export function buildJSON(result: VerifyResponse): string {
  return JSON.stringify({ ...result, generatedAt: new Date().toISOString() }, null, 2);
}

export function downloadJSON(result: VerifyResponse): void {
  const blob = new Blob([buildJSON(result)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = `trustcheck-${sanitizeFilename(result.input)}.json`;
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
  window.setTimeout(() => URL.revokeObjectURL(url), 0);
}

// buildBatchJSON wraps a list of results with a generation timestamp so batch
// exports carry their own audit trail. Successful results only.
export function buildBatchJSON(results: VerifyResponse[]): string {
  return JSON.stringify({ generatedAt: new Date().toISOString(), results }, null, 2);
}

export function downloadBatchJSON(results: VerifyResponse[]): void {
  const blob = new Blob([buildBatchJSON(results)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = 'trustcheck-batch-report.json';
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
  window.setTimeout(() => URL.revokeObjectURL(url), 0);
}

// renderReportHTML builds a self-contained, printable report styled with the
// project's palette (slate text, cyan brand, green/yellow/red/blue evidence).
export function renderReportHTML(result: VerifyResponse): string {
  const evidenceItems =
    result.evidence.length === 0
      ? '<li style="color:#64748b">No verification details available.</li>'
      : result.evidence
          .map((item) => {
            const icon = EVIDENCE_ICONS[item.result] ?? EVIDENCE_ICONS.info;
            const color = EVIDENCE_COLORS[item.result] ?? EVIDENCE_COLORS.info;
            return `<li style="color:${color}"><span aria-hidden="true">${icon}</span> ${escapeHtml(
              item.label,
            )}</li>`;
          })
          .join('\n');

  const rows = [
    ['Input', escapeHtml(result.input)],
    ['Type', escapeHtml(result.type)],
    ['Status', escapeHtml(result.status)],
    ['Trust Score', `${result.trustScore} / 100`],
    ['Summary', escapeHtml(result.summary)],
  ]
    .map(
      ([label, value]) =>
        `<div class="row"><div class="row-label">${label}</div><div class="row-value">${value}</div></div>`,
    )
    .join('\n');

  return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8" />
<title>TrustCheck — Verification Report</title>
<style>
  @media print {
    @page { margin: 16mm; }
    body { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
  }
  * { box-sizing: border-box; }
  body { margin: 0; color: #0f172a; font-family: Arial, Helvetica, sans-serif; font-size: 14px; line-height: 1.5; }
  .report { max-width: 700px; margin: 0 auto; padding: 24px; }
  .brand { color: #0891b2; font-size: 12px; font-weight: 700; letter-spacing: 0.3em; text-transform: uppercase; }
  h1 { font-size: 24px; font-weight: 700; margin: 8px 0 0; }
  hr { border: none; border-top: 2px solid #0f172a; margin: 16px 0 24px; }
  .row { display: flex; gap: 16px; padding: 10px 0; border-bottom: 1px solid #e2e8f0; }
  .row-label { width: 140px; flex: none; color: #64748b; font-weight: 600; }
  .row-value { flex: 1; min-width: 0; word-break: break-word; }
  h2 { font-size: 16px; font-weight: 700; margin: 24px 0 12px; }
  ul { margin: 0; padding-left: 4px; list-style: none; }
  li { padding: 6px 0; border-bottom: 1px dashed #e2e8f0; }
  .generated { margin-top: 28px; color: #64748b; font-size: 12px; }
</style>
</head>
<body>
  <main class="report">
    <p class="brand">TrustCheck</p>
    <h1>Verification Report</h1>
    <hr />
${rows}
    <h2>Evidence</h2>
    <ul>
${evidenceItems}
    </ul>
    <p class="generated">Generated ${formatReportDate(new Date())}</p>
  </main>
</body>
</html>`;
}

// downloadPDF has no PDF library available in this project, so it renders the
// printable HTML report in a new window (using a print stylesheet) and invokes
// the browser's print dialog, where the user can "Save as PDF".
export async function downloadPDF(result: VerifyResponse): Promise<void> {
  const win = window.open('', '_blank', 'width=800,height=900');
  if (!win) {
    return;
  }
  win.document.open();
  win.document.write(renderReportHTML(result));
  win.document.close();
  win.focus();
  // Give the new window a beat to lay out before opening the print dialog.
  await new Promise((resolve) => window.setTimeout(resolve, 300));
  win.print();
}
