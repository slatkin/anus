export const EMPTY_SET = new Set();

// Namespaced, stable collapse keys. Feed keys can never collide with category keys,
// so each grouping mode keeps independent collapse state in the shared set.
export const feedKey = (e) => 'feed:' + e.feed_id;
export const catKey  = (e) => 'cat:'  + (e.feed.category?.id ?? 0); // 0 = uncategorized

export function buildGroupedItems(entries, collapsed, timeAgo = () => '') {
  const byFeed = new Map();
  const order = [];
  entries.forEach((e, idx) => {
    const key = feedKey(e);
    if (!byFeed.has(key)) {
      byFeed.set(key, { title: e.feed.title, rows: [] });
      order.push(key);
    }
    byFeed.get(key).rows.push({
      type:      'item',
      cursorIdx: idx,
      id:        e.id,
      title:     e.title,
      sub:       (e.starred ? '★  ' : '') + timeAgo(e.published_at),
      unread:    e.status === 'unread',
    });
  });
  order.sort((a, b) => byFeed.get(a).title.localeCompare(byFeed.get(b).title));
  const out = [];
  for (const feedId of order) {
    const { title, rows } = byFeed.get(feedId);
    const isCollapsed = collapsed.has(feedId);
    out.push({ type: 'header', title, feedId, collapsed: isCollapsed, count: rows.length });
    if (!isCollapsed) out.push(...rows);
  }
  return out;
}

export function buildGroupedCatItems(entries, collapsed, timeAgo = () => '') {
  const byCat = new Map();
  const order = [];
  entries.forEach((e, idx) => {
    const key = catKey(e);
    if (!byCat.has(key)) { byCat.set(key, { title: e.feed.category?.title || 'All', rows: [] }); order.push(key); }
    byCat.get(key).rows.push({
      type:      'item',
      cursorIdx: idx,
      id:        e.id,
      title:     e.title,
      sub:       (e.starred ? '★  ' : '') + e.feed.title + '  ·  ' + timeAgo(e.published_at),
      unread:    e.status === 'unread',
    });
  });
  order.sort((a, b) => byCat.get(a).title.localeCompare(byCat.get(b).title));
  const out = [];
  for (const key of order) {
    const { title, rows } = byCat.get(key);
    const isCollapsed = collapsed.has(key);
    out.push({ type: 'header', title, feedId: key, collapsed: isCollapsed, count: rows.length });
    if (!isCollapsed) out.push(...rows);
  }
  return out;
}
