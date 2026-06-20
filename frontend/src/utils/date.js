export function timeAgo(iso) {
  const ms  = Date.now() - new Date(iso).getTime();
  const min = Math.floor(ms / 60000);
  const hr  = Math.floor(ms / 3600000);
  const day = Math.floor(ms / 86400000);
  if (min < 1)  return 'just now';
  if (min < 60) return `${min}m ago`;
  if (hr  < 24) return `${hr}h ago`;
  if (day <  7) return `${day}d ago`;
  return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

export function fullDate(iso) {
  return new Date(iso).toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
}
