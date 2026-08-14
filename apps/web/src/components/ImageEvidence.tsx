'use client';

import { memo } from 'react';
import type { EvidenceItem, ImageType } from '../types';

interface ImageEvidenceProps {
  evidence: EvidenceItem;
}

function ImageEvidence({ evidence }: ImageEvidenceProps) {
  if (!evidence.imageType) {
    return null;
  }

  return (
    <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/50">
      <div className="flex items-start gap-3">
        {/* Image preview */}
        {(evidence.imageUrl || evidence.sourceImage) && (
          <div className="shrink-0">
            <img
              src={evidence.imageUrl || evidence.sourceImage}
              alt="Evidence image"
              className="h-20 w-20 rounded-lg object-cover shadow-sm"
            />
          </div>
        )}

        <div className="min-w-0 flex-1">
          {/* Image type badge */}
          <div className="mb-2 flex items-center gap-2">
            <span className="inline-flex items-center rounded-full bg-cyan-100 px-2.5 py-0.5 text-xs font-medium text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-300">
              {getImageTypeLabel(evidence.imageType)}
            </span>
            {evidence.extractedText && (
              <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800 dark:bg-green-900/30 dark:text-green-300">
                OCR Text Found
              </span>
            )}
          </div>

          {/* Extracted text (OCR) */}
          {evidence.extractedText && (
            <div className="mb-3">
              <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                Extracted Text (OCR)
              </h4>
              <p className="mt-1 rounded bg-white p-2 text-sm text-slate-700 shadow-sm dark:bg-slate-900 dark:text-slate-300">
                {evidence.extractedText}
              </p>
            </div>
          )}

          {/* Metadata (EXIF) */}
          {evidence.metadata && (
            <div className="mb-3">
              <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                Image Metadata
              </h4>
              <div className="mt-1 space-y-1">
                {evidence.metadata.gpsLatitude && evidence.metadata.gpsLongitude && (
                  <p className="text-sm text-slate-700 dark:text-slate-300">
                    <span className="font-medium">Location:</span>{' '}
                    {evidence.metadata.gpsLatitude.toFixed(6)},{' '}
                    {evidence.metadata.gpsLongitude.toFixed(6)}
                  </p>
                )}
                {evidence.metadata.captureDate && (
                  <p className="text-sm text-slate-700 dark:text-slate-300">
                    <span className="font-medium">Captured:</span> {evidence.metadata.captureDate}
                  </p>
                )}
                {evidence.metadata.device && (
                  <p className="text-sm text-slate-700 dark:text-slate-300">
                    <span className="font-medium">Device:</span> {evidence.metadata.device}
                  </p>
                )}
                {!evidence.metadata.gpsLatitude &&
                  !evidence.metadata.captureDate &&
                  !evidence.metadata.device && (
                    <p className="text-sm text-slate-500 dark:text-slate-400">
                      No location metadata found in image
                    </p>
                  )}
              </div>
            </div>
          )}

          {/* Provenance (Reverse image search) */}
          {evidence.provenance && (
            <div className="mb-3">
              <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                Image Provenance
              </h4>
              <div className="mt-1">
                <p className="text-sm text-slate-700 dark:text-slate-300">
                  {evidence.provenance.summary}
                </p>
                {evidence.provenance.foundElsewhere && evidence.provenance.otherOccurrences && (
                  <p className="mt-1 text-sm text-amber-600 dark:text-amber-400">
                    Found on {evidence.provenance.otherOccurrences} other page(s) — may be a stock
                    or reused image
                  </p>
                )}
                {evidence.provenance.isStockPhoto && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">
                    ⚠️ This appears to be a stock photo — lowers trust rather than raising it
                  </p>
                )}
              </div>
            </div>
          )}

          {/* Evidence note */}
          {evidence.note && (
            <div className="border-t border-slate-200 pt-3 dark:border-slate-700">
              <p className="text-sm text-slate-600 dark:text-slate-400">{evidence.note}</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function getImageTypeLabel(imageType: ImageType): string {
  const labels: Record<ImageType, string> = {
    storefront: 'Storefront',
    document: 'Document',
    logo: 'Logo',
    building: 'Building',
    screenshot: 'Screenshot',
    product: 'Product',
    person: 'Person',
    unknown: 'Unknown',
  };
  return labels[imageType];
}

export default memo(ImageEvidence);
