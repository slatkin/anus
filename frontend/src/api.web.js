async function post(path, body) {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(await res.text());
}

export async function FetchEntries() {
  const res = await fetch('/api/entries');
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function RefreshAndFetch() {
  const res = await fetch('/api/refresh-and-fetch', { method: 'POST' });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function FetchArticleContent(id, url) {
  const res = await fetch(`/api/fetch-content?id=${id}&url=${encodeURIComponent(url)}`);
  if (!res.ok) throw new Error(await res.text());
  const data = await res.json();
  return data.content;
}

export async function MarkRead(ids) {
  return post('/api/mark-read', { ids });
}

export async function MarkUnread(ids) {
  return post('/api/mark-unread', { ids });
}

export async function ToggleStar(id) {
  return post('/api/toggle-star', { id });
}

export async function SaveEntry(id) {
  return post('/api/save-entry', { id });
}

export function OpenURL(url) {
  window.open(url, '_blank', 'noopener,noreferrer');
}
