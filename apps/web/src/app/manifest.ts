import type { MetadataRoute } from 'next';

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: 'TrustCheck',
    short_name: 'TrustCheck',
    description:
      'One place to sanity-check domains, emails, IPs, phone numbers, and businesses before you trust them.',
    start_url: '/',
    display: 'standalone',
    background_color: '#f8fafc',
    theme_color: '#0891b2',
    icons: [{ src: '/icon.svg', sizes: 'any', type: 'image/svg+xml' }],
  };
}
