// formatRelativeTime renders a timestamp as a short, human-friendly relative
// label (e.g. "Just now", "2 min ago", "1 hour ago", "Yesterday", "3 days ago").
// No external library: pure arithmetic on epoch milliseconds.
export function formatRelativeTime(timestamp: number): string {
  const minutes = Math.floor((Date.now() - timestamp) / 60_000);
  if (minutes < 1) {
    return 'Just now';
  }
  if (minutes < 60) {
    return `${minutes} min ago`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return hours === 1 ? '1 hour ago' : `${hours} hours ago`;
  }
  const days = Math.floor(hours / 24);
  if (days === 1) {
    return 'Yesterday';
  }
  return `${days} days ago`;
}
