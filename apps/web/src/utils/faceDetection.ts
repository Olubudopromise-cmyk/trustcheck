/**
 * Face Detection Module using MediaPipe Face Detector
 *
 * Uses MediaPipe's face detection via CDN to detect faces in images.
 * This replaces the skin-tone heuristic with a real ML-based face detector.
 *
 * Key behavior:
 * - If ANY face is detected (even incidental), refuse to process as evidence
 * - Fails closed: if detection fails or is ambiguous, refuse processing
 * - Downloads model on first use, caches for subsequent calls
 */

// MediaPipe CDN URLs
const MEDIAPIPE_WASM_CDN = 'https://cdn.jsdelivr.net/npm/@mediapipe/tasks-vision@latest/wasm';
const FACE_DETECTOR_MODEL_URL =
  'https://storage.googleapis.com/mediapipe-models/face_detector/blaze_face_short_range/float16/latest/blaze_face_short_range.tflite';

// Detection confidence threshold - lower means more sensitive (catches more faces)
const MIN_DETECTION_CONFIDENCE = 0.3;

// Singleton detector instance
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let faceDetector: any = null;
let detectorLoading: Promise<unknown> | null = null;
let detectorLoadFailed = false;

/**
 * Load MediaPipe vision tasks and create face detector
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
async function loadFaceDetector(): Promise<any> {
  // Return cached detector if available
  if (faceDetector) return faceDetector;

  // If previous load failed, don't retry
  if (detectorLoadFailed) {
    throw new Error('Face detector failed to load previously');
  }

  // If currently loading, wait for it
  if (detectorLoading) return detectorLoading;

  detectorLoading = (async () => {
    try {
      // Dynamically import MediaPipe tasks-vision
      // @ts-expect-error - MediaPipe types may not be available
      const vision = await import('@mediapipe/tasks-vision');
      const { FaceDetector, FilesetResolver } = vision;

      // Load WASM fileset from CDN
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const filesetResolver: any = await (FilesetResolver as any).forVisionTasks(
        MEDIAPIPE_WASM_CDN,
      );

      // Create face detector
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const detector: any = await (FaceDetector as any).createFromOptions(filesetResolver, {
        baseOptions: {
          modelAssetPath: FACE_DETECTOR_MODEL_URL,
          delegate: 'GPU', // Use GPU if available, falls back to CPU
        },
        runningMode: 'IMAGE',
        minDetectionConfidence: MIN_DETECTION_CONFIDENCE,
        minSuppressionThreshold: 0.3,
      });

      faceDetector = detector;
      return detector;
    } catch (error) {
      detectorLoadFailed = true;
      console.error('Failed to load face detector:', error);
      throw error;
    }
  })();

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return detectorLoading as Promise<any>;
}

/**
 * Detect faces in an image file
 * @param file - Image file to analyze
 * @returns Array of face detections with bounding boxes
 */
export async function detectFaces(file: File): Promise<
  Array<{
    boundingBox: { x: number; y: number; width: number; height: number };
    confidence: number;
  }>
> {
  try {
    const detector = await loadFaceDetector();

    // Load image into HTMLImageElement
    const img = await loadImageElement(file);

    // Run face detection
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const result: any = detector.detect(img);

    // Clean up
    URL.revokeObjectURL(img.src);

    // Return detections with bounding boxes
    if (result.detections && result.detections.length > 0) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return result.detections.map((detection: any) => ({
        boundingBox: detection.boundingBox,
        confidence: detection.categories?.[0]?.score ?? 0,
      }));
    }

    return [];
  } catch (error) {
    console.warn('Face detection failed:', error);
    // Fail closed - if detection fails, assume face might be present
    throw new Error('Face detection failed');
  }
}

/**
 * Load image file into HTMLImageElement
 */
function loadImageElement(file: File): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error('Failed to load image'));
    img.src = URL.createObjectURL(file);
  });
}

/**
 * Check if image contains any faces
 * @param file - Image file to check
 * @returns true if faces detected, false otherwise
 */
export async function containsFaces(file: File): Promise<boolean> {
  try {
    const faces = await detectFaces(file);
    return faces.length > 0;
  } catch (error) {
    // Fail closed - if we can't detect faces, assume face might be present
    console.warn('Face detection failed, failing closed:', error);
    return true; // Assume face present to be safe
  }
}

/**
 * Get face detection status for UI display
 */
export function getFaceDetectorStatus(): 'ready' | 'loading' | 'failed' {
  if (faceDetector) return 'ready';
  if (detectorLoadFailed) return 'failed';
  if (detectorLoading) return 'loading';
  return 'loading'; // Not started yet
}

/**
 * Pre-load face detector for faster subsequent detection
 * Call this on app startup or when user hovers over upload button
 */
export function preloadFaceDetector(): void {
  loadFaceDetector().catch(() => {
    // Ignore preload failures - will fail closed on actual detection
  });
}
