/**
 * OCR Module using Tesseract.js via CDN
 *
 * Provides client-side OCR with:
 * - Confidence scores per word and line
 * - Worker-based processing (off main thread)
 * - Timeout mechanism
 * - Graceful degradation for blurry/low-res images
 */

// Tesseract.js CDN URL
const TESSERACT_CDN = 'https://cdn.jsdelivr.net/npm/tesseract.js@5/dist/tesseract.min.js';

// Load Tesseract.js from CDN
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let TesseractLib: any = null;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
async function loadTesseract(): Promise<any> {
  if (TesseractLib) return TesseractLib;

  // Check if already loaded via script tag
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  if (typeof window !== 'undefined' && (window as any).Tesseract) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    TesseractLib = (window as any).Tesseract;
    return TesseractLib;
  }

  // Load via dynamic script tag
  return new Promise((resolve, reject) => {
    const script = document.createElement('script');
    script.src = TESSERACT_CDN;
    script.onload = () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      TesseractLib = (window as any).Tesseract;
      if (TesseractLib) {
        resolve(TesseractLib);
      } else {
        reject(new Error('Failed to load Tesseract.js'));
      }
    };
    script.onerror = () => reject(new Error('Failed to load Tesseract.js script'));
    document.head.appendChild(script);
  });
}

// Configuration
const OCR_TIMEOUT_MS = 15000; // 15 seconds timeout
const MIN_CONFIDENCE_THRESHOLD = 40; // Below this, text is flagged as low-confidence
const HIGH_CONFIDENCE_THRESHOLD = 80; // Above this, text is considered reliable

// Types for OCR results
export interface OcrWord {
  text: string;
  confidence: number;
  bbox: { x0: number; y0: number; x1: number; y1: number };
}

export interface OcrLine {
  text: string;
  confidence: number;
  words: OcrWord[];
}

export interface OcrBlock {
  text: string;
  confidence: number;
  lines: OcrLine[];
}

export interface OcrResult {
  text: string;
  confidence: number; // Overall confidence (0-100)
  blocks: OcrBlock[];
  words: OcrWord[];
  lines: OcrLine[];
  hasReliableText: boolean; // true if any text meets confidence threshold
  hasHighConfidenceText: boolean; // true if any text is high confidence
  lowConfidenceWords: OcrWord[]; // Words below threshold
  processingTimeMs: number;
  timedOut: boolean;
  error?: string;
}

// Singleton scheduler for managing workers
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let scheduler: any = null;
let schedulerInitializing = false;
let schedulerFailed = false;

/**
 * Initialize the OCR scheduler with workers
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
async function initScheduler(): Promise<any> {
  if (scheduler) return scheduler;
  if (schedulerFailed) throw new Error('OCR scheduler failed to initialize');
  if (schedulerInitializing) {
    // Wait for existing initialization
    await new Promise((resolve) => setTimeout(resolve, 100));
    if (scheduler) return scheduler;
    throw new Error('OCR scheduler initialization in progress');
  }

  schedulerInitializing = true;

  try {
    // Load Tesseract.js from CDN
    const tesseractLib = await loadTesseract();

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    scheduler = (tesseractLib as any).createScheduler();

    // Create a single worker for now (can be expanded for parallelism)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const worker = await (tesseractLib as any).createWorker('eng', 1, {
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      logger: (_m: unknown) => {
        // Optional: log progress
      },
    });

    scheduler.addWorker(worker);
    schedulerInitializing = false;
    return scheduler;
  } catch (error) {
    schedulerFailed = true;
    schedulerInitializing = false;
    throw error;
  }
}

/**
 * Run OCR on an image file with timeout
 */
export async function performOcr(file: File): Promise<OcrResult> {
  const startTime = Date.now();

  try {
    // Initialize scheduler if needed
    const sched = await initScheduler();

    // Create timeout promise
    const timeoutPromise = new Promise<never>((_, reject) => {
      setTimeout(() => reject(new Error('OCR_TIMEOUT')), OCR_TIMEOUT_MS);
    });

    // Run OCR with timeout
    const ocrPromise = runOcrWithDetails(sched, file);

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const result: any = await Promise.race([ocrPromise, timeoutPromise]);

    return {
      ...result,
      processingTimeMs: Date.now() - startTime,
      timedOut: false,
    };
  } catch (error) {
    const processingTimeMs = Date.now() - startTime;

    if (error instanceof Error && error.message === 'OCR_TIMEOUT') {
      // Return partial result on timeout
      return {
        text: '',
        confidence: 0,
        blocks: [],
        words: [],
        lines: [],
        hasReliableText: false,
        hasHighConfidenceText: false,
        lowConfidenceWords: [],
        processingTimeMs,
        timedOut: true,
        error: 'OCR processing timed out. Image may be too large or complex.',
      };
    }

    return {
      text: '',
      confidence: 0,
      blocks: [],
      words: [],
      lines: [],
      hasReliableText: false,
      hasHighConfidenceText: false,
      lowConfidenceWords: [],
      processingTimeMs,
      timedOut: false,
      error: `OCR failed: ${error instanceof Error ? error.message : 'Unknown error'}`,
    };
  }
}

/**
 * Run OCR and extract detailed results
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
async function runOcrWithDetails(sched: any, file: File): Promise<OcrResult> {
  // Get image as buffer
  const imageData = await file.arrayBuffer();

  // Run OCR with detailed output
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const { data }: any = await sched.addJob(
    'recognize',
    imageData,
    {},
    {
      text: true,
      blocks: true,
      hocr: false,
      tsv: false,
    },
  );

  // Parse results
  const words: OcrWord[] = [];
  const lines: OcrLine[] = [];
  const blocks: OcrBlock[] = [];
  const lowConfidenceWords: OcrWord[] = [];

  let totalConfidence = 0;
  let wordCount = 0;
  let hasReliableText = false;
  let hasHighConfidenceText = false;

  // Process blocks
  if (data.blocks) {
    for (const block of data.blocks) {
      const blockLines: OcrLine[] = [];

      for (const line of block.lines) {
        const lineWords: OcrWord[] = [];

        for (const word of line.words) {
          const wordConfidence = word.confidence || 0;
          const ocrWord: OcrWord = {
            text: word.text,
            confidence: wordConfidence,
            bbox: word.bbox,
          };

          words.push(ocrWord);
          lineWords.push(ocrWord);

          // Track confidence
          totalConfidence += wordConfidence;
          wordCount++;

          // Check thresholds
          if (wordConfidence >= MIN_CONFIDENCE_THRESHOLD) {
            hasReliableText = true;
          }
          if (wordConfidence >= HIGH_CONFIDENCE_THRESHOLD) {
            hasHighConfidenceText = true;
          }
          if (wordConfidence < MIN_CONFIDENCE_THRESHOLD) {
            lowConfidenceWords.push(ocrWord);
          }
        }

        const lineConfidence =
          lineWords.length > 0
            ? lineWords.reduce((sum, w) => sum + w.confidence, 0) / lineWords.length
            : 0;

        lines.push({
          text: line.text,
          confidence: lineConfidence,
          words: lineWords,
        });

        blockLines.push(lines[lines.length - 1]);
      }

      const blockConfidence =
        blockLines.length > 0
          ? blockLines.reduce((sum, l) => sum + l.confidence, 0) / blockLines.length
          : 0;

      blocks.push({
        text: block.text,
        confidence: blockConfidence,
        lines: blockLines,
      });
    }
  }

  // Calculate overall confidence
  const overallConfidence = wordCount > 0 ? totalConfidence / wordCount : 0;

  // Clean up text - remove excessive whitespace
  const cleanText = (data.text || '').replace(/\s+/g, ' ').trim();

  // Check if we have any meaningful text
  const hasMeaningfulText = cleanText.length > 0 && wordCount > 0;

  return {
    text: cleanText,
    confidence: Math.round(overallConfidence),
    blocks,
    words,
    lines,
    hasReliableText: hasMeaningfulText && hasReliableText,
    hasHighConfidenceText: hasHighConfidenceText,
    lowConfidenceWords,
    processingTimeMs: 0, // Will be set by caller
    timedOut: false,
  };
}

/**
 * Format OCR result for display
 */
export function formatOcrResult(result: OcrResult): string {
  if (result.error) {
    return result.error;
  }

  if (result.timedOut) {
    return 'OCR processing timed out. Image may be too large or complex.';
  }

  if (!result.text || result.text.length === 0) {
    return 'No readable text found in image.';
  }

  if (!result.hasReliableText) {
    return `Text detected but low confidence (${result.confidence}%). Results may be unreliable.`;
  }

  // Build formatted result with confidence info
  const parts: string[] = [];

  if (result.hasHighConfidenceText) {
    const highConfWords = result.words.filter((w) => w.confidence >= HIGH_CONFIDENCE_THRESHOLD);
    if (highConfWords.length > 0) {
      parts.push(`High confidence text: "${highConfWords.map((w) => w.text).join(' ')}"`);
    }
  }

  if (
    result.lowConfidenceWords.length > 0 &&
    result.lowConfidenceWords.length < result.words.length
  ) {
    parts.push(
      `Low confidence text (${result.lowConfidenceWords.length} words): "${result.lowConfidenceWords.map((w) => w.text).join(' ')}"`,
    );
  }

  if (parts.length === 0) {
    parts.push(result.text);
  }

  return parts.join('\n');
}

/**
 * Get OCR status for UI display
 */
export function getOcrStatus(): 'ready' | 'loading' | 'failed' {
  if (schedulerFailed) return 'failed';
  if (scheduler) return 'ready';
  return 'loading';
}

/**
 * Pre-load OCR worker for faster subsequent recognition
 */
export async function preloadOcr(): Promise<void> {
  try {
    await initScheduler();
  } catch {
    // Ignore preload failures
  }
}

/**
 * Terminate OCR worker to free resources
 */
export async function terminateOcr(): Promise<void> {
  if (scheduler) {
    await scheduler.terminate();
    scheduler = null;
  }
}
