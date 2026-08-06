import './globals.css';
import type { Metadata, Viewport } from 'next';

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || 'http://localhost:3000';

const siteTitle = 'TrustCheck — Verify Anything. Trust Everything.';
const siteDescription =
  'One place to sanity-check domains, emails, IPs, phone numbers, and businesses before you trust them.';

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: {
    default: siteTitle,
    template: '%s · TrustCheck',
  },
  description: siteDescription,
  keywords: [
    'verification',
    'trust score',
    'domain check',
    'email verification',
    'IP lookup',
    'phone validation',
    'business verification',
    'scam detection',
    'link check',
  ],
  openGraph: {
    type: 'website',
    url: siteUrl,
    siteName: 'TrustCheck',
    title: siteTitle,
    description: siteDescription,
    locale: 'en_US',
  },
  twitter: {
    card: 'summary',
    title: siteTitle,
    description: siteDescription,
  },
  icons: {
    icon: '/icon.svg',
  },
  manifest: '/manifest.webmanifest',
};

export const viewport: Viewport = {
  themeColor: '#0891b2',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
