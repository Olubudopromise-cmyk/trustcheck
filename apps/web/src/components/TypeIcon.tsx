'use client';

export const typeLabels: Record<string, string> = {
  domain: 'Domain',
  url: 'URL',
  email: 'Email',
  ipv4: 'IPv4',
  ipv6: 'IPv6',
  company: 'Company',
  phone: 'Phone',
  unknown: 'Unknown',
};

const typeIcons: Record<string, { emoji: string; label: string }> = {
  domain: { emoji: '🌐', label: 'Domain' },
  url: { emoji: '🔗', label: 'URL' },
  email: { emoji: '📧', label: 'Email' },
  ipv4: { emoji: '🌍', label: 'IPv4' },
  ipv6: { emoji: '🌎', label: 'IPv6' },
  company: { emoji: '🏢', label: 'Company' },
  phone: { emoji: '📞', label: 'Phone' },
  unknown: { emoji: '❓', label: 'Unknown' },
};

export function typeLabel(t: string): string {
  return typeLabels[t] || typeLabels[Object.keys(typeLabels)[0]] || t;
}

export default function TypeIcon({ type: t }: { type: string }) {
  const icon = typeIcons[t] || typeIcons.unknown;
  return (
    <span role="img" aria-label={icon.label} className="text-3xl leading-none">
      {icon.emoji}
    </span>
  );
}
