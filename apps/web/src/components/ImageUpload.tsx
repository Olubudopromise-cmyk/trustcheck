'use client';

import { memo, useCallback, useRef, useState } from 'react';
import type { ImageProcessingResult } from '../utils/imageProcessing';
import { processImage } from '../utils/imageProcessing';
import { preloadFaceDetector } from '../utils/faceDetection';
import { preloadOcr } from '../utils/ocr';

interface ImageUploadProps {
  onImageProcessed: (result: ImageProcessingResult) => void;
  onError: (error: string) => void;
  disabled?: boolean;
}

function ImageUpload({ onImageProcessed, onError, disabled }: ImageUploadProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [isProcessing, setIsProcessing] = useState(false);
  const [processingStatus, setProcessingStatus] = useState<string>('');
  const [selectedImage, setSelectedImage] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);

  const handleFileSelect = useCallback(
    async (event: React.ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      if (!file) return;

      // Validate file type
      const validTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
      if (!validTypes.includes(file.type)) {
        onError('Please select a valid image file (JPEG, PNG, GIF, or WebP)');
        return;
      }

      // Validate file size (max 10MB)
      if (file.size > 10 * 1024 * 1024) {
        onError('Image file too large. Please select an image under 10MB.');
        return;
      }

      setSelectedImage(file);

      // Create preview URL
      const url = URL.createObjectURL(file);
      setPreviewUrl(url);

      // Start processing
      setIsProcessing(true);
      setProcessingStatus('Analyzing image...');

      try {
        // Process the image
        const result = await processImage(file);

        if (result.error) {
          onError(result.error);
          setSelectedImage(null);
          setPreviewUrl(null);
          return;
        }

        setProcessingStatus('Processing complete');
        onImageProcessed(result);
      } catch (error) {
        onError(
          `Image processing failed: ${error instanceof Error ? error.message : 'Unknown error'}`,
        );
        setSelectedImage(null);
        setPreviewUrl(null);
      } finally {
        setIsProcessing(false);
      }
    },
    [onImageProcessed, onError],
  );

  const handleRemoveImage = useCallback(() => {
    setSelectedImage(null);
    setPreviewUrl(null);
    setProcessingStatus('');
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }, []);

  const handleClick = useCallback(() => {
    if (!disabled && fileInputRef.current) {
      fileInputRef.current.click();
    }
  }, [disabled]);

  const handleMouseEnter = useCallback(() => {
    // Preload face detector and OCR on hover for faster processing
    preloadFaceDetector();
    preloadOcr();
  }, []);

  return (
    <div className="relative">
      {/* Hidden file input */}
      <input
        ref={fileInputRef}
        type="file"
        accept="image/jpeg,image/png,image/gif,image/webp"
        onChange={handleFileSelect}
        className="hidden"
        disabled={disabled}
        aria-label="Upload image for evidence"
      />

      {/* Upload button */}
      {!selectedImage && (
        <button
          type="button"
          onClick={handleClick}
          onMouseEnter={handleMouseEnter}
          disabled={disabled}
          className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-400 transition hover:border-cyan-300 hover:text-cyan-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-500 dark:hover:border-cyan-600 dark:hover:text-cyan-400 dark:ring-offset-slate-900"
          aria-label="Attach image"
          title="Attach image as evidence"
        >
          <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
            />
          </svg>
        </button>
      )}

      {/* Selected image preview */}
      {selectedImage && previewUrl && (
        <div className="relative">
          <div className="flex items-center gap-3 rounded-xl border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900">
            {/* Image preview */}
            <div className="relative h-16 w-16 shrink-0 overflow-hidden rounded-lg border border-slate-200 dark:border-slate-700">
              <img
                src={previewUrl}
                alt="Selected evidence"
                className="h-full w-full object-cover"
              />
              {isProcessing && (
                <div className="absolute inset-0 flex items-center justify-center bg-black/50">
                  <svg className="h-6 w-6 animate-spin text-white" viewBox="0 0 24 24" fill="none">
                    <circle
                      className="opacity-25"
                      cx={12}
                      cy={12}
                      r={10}
                      stroke="currentColor"
                      strokeWidth={4}
                    />
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"
                    />
                  </svg>
                </div>
              )}
            </div>

            {/* Image info */}
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium text-slate-900 dark:text-slate-100">
                {selectedImage.name}
              </p>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {(selectedImage.size / 1024).toFixed(1)} KB
              </p>
              {processingStatus && (
                <p className="mt-1 text-xs text-cyan-600 dark:text-cyan-400">{processingStatus}</p>
              )}
            </div>

            {/* Remove button */}
            <button
              type="button"
              onClick={handleRemoveImage}
              disabled={isProcessing}
              className="shrink-0 rounded-lg p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 disabled:opacity-50 dark:hover:bg-slate-800 dark:hover:text-slate-300"
              aria-label="Remove image"
            >
              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default memo(ImageUpload);
