export const EMPTY_SET = new Set();

export function buildGroupedItems(entries, collapsed, timeAgo = () => '') {
  const byFeed = new Map();
  const order = [];
  entries.forEach((e, idx) => {
    if (!byFeed.has(e.feed_id)) {
      byFeed.set(e.feed_id, { title: e.feed.title, rows: [] });
      order.push(e.feed_id);
    }
    byFeed.get(e.feed_id).rows.push({
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
    const catTitle = e.feed.category?.title || 'All';
    if (!byCat.has(catTitle)) { byCat.set(catTitle, { title: catTitle, rows: [] }); order.push(catTitle); }
    byCat.get(catTitle).rows.push({
      type:      'item',
      cursorIdx: idx,
      id:        e.id,
      title:     e.title,
      sub:       (e.starred ? '★  ' : '') + e.feed.title + '  ·  ' + timeAgo(e.published_at),
      unread:    e.status === 'unread',
    });
  });
  order.sort((a, b) => a.localeCompare(b));
  const out = [];
  for (const key of order) {
    const { title, rows } = byCat.get(key);
    const isCollapsed = collapsed.has(key);
    out.push({ type: 'header', title, feedId: key, collapsed: isCollapsed, count: rows.length });
    if (!isCollapsed) out.push(...rows);
  }
  return out;
}
