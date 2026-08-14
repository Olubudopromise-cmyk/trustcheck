import type { ImageMetadata, ImageProvenance, ImageType } from '../types';
import { containsFaces } from './faceDetection';
import { performOcr as performOcrWithConfidence } from './ocr';

// Image processing result interface
export interface ImageProcessingResult {
  imageType: ImageType;
  extractedText?: string;
  ocrConfidence?: number; // Overall OCR confidence (0-100)
  hasHighConfidenceText?: boolean;
  metadata?: ImageMetadata;
  provenance?: ImageProvenance;
  imageUrl?: string;
  sourceImage?: string;
  error?: string;
}

// Extract EXIF metadata from image file
export async function extractExifMetadata(file: File): Promise<ImageMetadata> {
  const metadata: ImageMetadata = {};

  try {
    // Read file as ArrayBuffer
    const buffer = await file.arrayBuffer();
    const view = new DataView(buffer);

    // Check for JPEG EXIF header
    if (view.getUint16(0) === 0xffd8) {
      // Look for EXIF data
      let offset = 2;
      while (offset < buffer.byteLength - 1) {
        const marker = view.getUint16(offset);
        if (marker === 0xffe1) {
          // EXIF marker found
          const exifLength = view.getUint16(offset + 2);
          const exifData = buffer.slice(offset + 4, offset + 2 + exifLength);

          // Parse EXIF data for GPS and other metadata
          const exifInfo = parseExifData(exifData);
          if (exifInfo.gpsLatitude) metadata.gpsLatitude = exifInfo.gpsLatitude;
          if (exifInfo.gpsLongitude) metadata.gpsLongitude = exifInfo.gpsLongitude;
          if (exifInfo.captureDate) metadata.captureDate = exifInfo.captureDate;
          if (exifInfo.device) metadata.device = exifInfo.device;

          break;
        }
        offset += 2 + view.getUint16(offset + 2);
      }
    }
  } catch (error) {
    console.warn('Failed to extract EXIF metadata:', error);
  }

  return metadata;
}

// Parse EXIF data (simplified implementation)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function parseExifData(_exifData: ArrayBuffer): Partial<ImageMetadata> {
  const result: Partial<ImageMetadata> = {};

  // In a real implementation, you would parse the EXIF structure
  // For now, we'll return empty metadata
  // TODO: Implement actual EXIF parsing with a library like exif-js

  return result;
}

// Perform OCR on image using Tesseract.js with confidence scores
export async function performOcr(file: File): Promise<string> {
  try {
    const result = await performOcrWithConfidence(file);

    // Return formatted result based on confidence
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

    // Return the extracted text with confidence indicator
    const confidenceLabel =
      result.confidence >= 80 ? 'High' : result.confidence >= 50 ? 'Medium' : 'Low';
    return `${confidenceLabel} confidence OCR (${result.confidence}%): ${result.text}`;
  } catch (error) {
    console.warn('OCR processing failed:', error);
    return 'OCR processing failed.';
  }
}

// Load image from file
function loadImage(file: File): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = reject;
    img.src = URL.createObjectURL(file);
  });
}

// Classify image type based on content analysis
export async function classifyImageType(file: File): Promise<ImageType> {
  try {
    const img = await loadImage(file);
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');

    if (!ctx) {
      return 'unknown';
    }

    canvas.width = img.width;
    canvas.height = img.height;
    ctx.drawImage(img, 0, 0);

    const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);

    // Simple color and shape analysis for classification
    // In production, you would use a proper image classification model

    // Check for text-like patterns (documents, screenshots)
    const hasTextPatterns = analyzeTextPatterns(imageData);
    if (hasTextPatterns) {
      return 'document';
    }

    // Check for building-like patterns
    const hasBuildingPatterns = analyzeBuildingPatterns(imageData);
    if (hasBuildingPatterns) {
      return 'building';
    }

    // Default to unknown for now
    return 'unknown';
  } catch (error) {
    console.warn('Image classification failed:', error);
    return 'unknown';
  }
}

// Analyze image for text patterns (simplified)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function analyzeTextPatterns(_imageData: ImageData): boolean {
  // This would analyze the image for text-like patterns
  // For now, return false as this requires more sophisticated analysis
  return false;
}

// Analyze image for building patterns (simplified)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function analyzeBuildingPatterns(_imageData: ImageData): boolean {
  // This would analyze the image for building-like patterns
  // For now, return false as this requires more sophisticated analysis
  return false;
}

// Check if image contains any faces (using real ML-based face detection)
// If ANY face is detected (even incidental), refuse to process as evidence
export async function isPersonImage(file: File): Promise<boolean> {
  try {
    // Use MediaPipe face detection - fails closed if detection fails
    const hasFaces = await containsFaces(file);
    return hasFaces;
  } catch (error) {
    // Fail closed: if face detection fails, assume face might be present
    console.warn('Face detection failed, assuming face present for safety:', error);
    return true;
  }
}

// Main image processing function
export async function processImage(file: File): Promise<ImageProcessingResult> {
  const result: ImageProcessingResult = {
    imageType: 'unknown',
  };

  try {
    // Check if image is primarily a person
    const isPerson = await isPersonImage(file);
    if (isPerson) {
      result.imageType = 'person';
      result.error =
        'Image appears to be a person photo. TrustCheck does not process person images for evidence.';
      return result;
    }

    // Classify image type
    result.imageType = await classifyImageType(file);

    // Extract metadata
    result.metadata = await extractExifMetadata(file);

    // Perform OCR with confidence scores
    const ocrResult = await performOcrWithConfidence(file);
    result.extractedText = ocrResult.text;
    result.ocrConfidence = ocrResult.confidence;
    result.hasHighConfidenceText = ocrResult.hasHighConfidenceText;

    // Note: Reverse image search would be implemented here
    // For now, we'll set a placeholder
    result.provenance = {
      foundElsewhere: false,
      summary: 'Reverse image search not yet implemented',
    };
  } catch (error) {
    result.error = `Image processing failed: ${error instanceof Error ? error.message : 'Unknown error'}`;
  }

  return result;
}
