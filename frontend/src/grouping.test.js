import { describe, it, expect } from 'vitest';
import { EMPTY_SET, buildGroupedItems, buildGroupedCatItems } from './grouping.js';

// ── fixtures ──────────────────────────────────────────────────────────────

function makeEntry(id, feedId, feedTitle, cat, opts = {}) {
  return {
    id,
    feed_id: feedId,
    title: opts.title ?? `Entry ${id}`,
    status: opts.status ?? 'unread',
    starred: opts.starred ?? false,
    published_at: opts.published_at ?? '2024-01-01T00:00:00Z',
    feed: {
      title: feedTitle,
      category: cat ? { title: cat } : null,
    },
  };
}

// ── buildGroupedItems ─────────────────────────────────────────────────────

describe('buildGroupedItems', () => {
  it('returns empty array for no entries', () => {
    expect(buildGroupedItems([], EMPTY_SET)).toEqual([]);
  });

  it('produces one header and one item for a single entry', () => {
    const entries = [makeEntry(1, 10, 'Feed A', null)];
    const items = buildGroupedItems(entries, EMPTY_SET);
    expect(items).toHaveLength(2);
    expect(items[0]).toMatchObject({ type: 'header', title: 'Feed A', feedId: 10, count: 1, collapsed: false });
    expect(items[1]).toMatchObject({ type: 'item', id: 1, cursorIdx: 0 });
  });

  it('groups entries by feed_id', () => {
    const entries = [
      makeEntry(1, 10, 'Feed A', null),
      makeEntry(2, 20, 'Feed B', null),
      makeEntry(3, 10, 'Feed A', null),
    ];
    const items = buildGroupedItems(entries, EMPTY_SET);
    const headers = items.filter(i => i.type === 'header');
    expect(headers).toHaveLength(2);
    const feedAHeader = headers.find(h => h.feedId === 10);
    expect(feedAHeader.count).toBe(2);
    const feedBHeader = headers.find(h => h.feedId === 20);
    expect(feedBHeader.count).toBe(1);
  });

  it('sorts feed headers alphabetically by title', () => {
    const entries = [
      makeEntry(1, 30, 'Zebra Feed', null),
      makeEntry(2, 10, 'Alpha Feed', null),
      makeEntry(3, 20, 'Mango Feed', null),
    ];
    const items = buildGroupedItems(entries, EMPTY_SET);
    const headers = items.filter(i => i.type === 'header').map(h => h.title);
    expect(headers).toEqual(['Alpha Feed', 'Mango Feed', 'Zebra Feed']);
  });

  it('assigns cursorIdx based on original entry array position', () => {
    const entries = [
      makeEntry(1, 10, 'Feed A', null),
      makeEntry(2, 20, 'Feed B', null),
      makeEntry(3, 10, 'Feed A', null),
    ];
    const items = buildGroupedItems(entries, EMPTY_SET);
    const entryItems = items.filter(i => i.type === 'item');
    // entry id=1 was at index 0, entry id=3 was at index 2
    expect(entryItems.find(i => i.id === 1).cursorIdx).toBe(0);
    expect(entryItems.find(i => i.id === 3).cursorIdx).toBe(2);
    expect(entryItems.find(i => i.id === 2).cursorIdx).toBe(1);
  });

  it('collapses a feed group when its feedId is in the collapsed set', () => {
    const entries = [
      makeEntry(1, 10, 'Feed A', null),
      makeEntry(2, 10, 'Feed A', null),
    ];
    const collapsed = new Set([10]);
    const items = buildGroupedItems(entries, collapsed);
    expect(items).toHaveLength(1);
    expect(items[0]).toMatchObject({ type: 'header', collapsed: true, count: 2 });
  });

  it('expands a feed group when its feedId is not in the collapsed set', () => {
    const entries = [makeEntry(1, 10, 'Feed A', null), makeEntry(2, 10, 'Feed A', null)];
    const items = buildGroupedItems(entries, EMPTY_SET);
    expect(items).toHaveLength(3); // 1 header + 2 items
    expect(items[0].collapsed).toBe(false);
  });

  it('marks unread entries correctly', () => {
    const entries = [
      makeEntry(1, 10, 'Feed A', null, { status: 'unread' }),
      makeEntry(2, 10, 'Feed A', null, { status: 'read' }),
    ];
    const items = buildGroupedItems(entries, EMPTY_SET).filter(i => i.type === 'item');
    expect(items.find(i => i.id === 1).unread).toBe(true);
    expect(items.find(i => i.id === 2).unread).toBe(false);
  });

  it('prefixes sub with star for starred entries', () => {
    const entry = makeEntry(1, 10, 'Feed A', null, { starred: true });
    const items = buildGroupedItems([entry], EMPTY_SET);
    expect(items[1].sub).toMatch(/^★/);
  });

  it('passes EMPTY_SET so all groups expand regardless of outer collapsed state', () => {
    const entries = [makeEntry(1, 10, 'Feed A', null)];
    const collapsed = new Set([10]);
    // With real collapsed set → collapsed
    expect(buildGroupedItems(entries, collapsed)).toHaveLength(1);
    // With EMPTY_SET → expanded
    expect(buildGroupedItems(entries, EMPTY_SET)).toHaveLength(2);
  });
});

// ── buildGroupedCatItems ──────────────────────────────────────────────────

describe('buildGroupedCatItems', () => {
  it('returns empty array for no entries', () => {
    expect(buildGroupedCatItems([], EMPTY_SET)).toEqual([]);
  });

  it('groups entries by category title', () => {
    const entries = [
      makeEntry(1, 10, 'Feed A', 'Tech'),
      makeEntry(2, 20, 'Feed B', 'Sports'),
      makeEntry(3, 30, 'Feed C', 'Tech'),
    ];
    const items = buildGroupedCatItems(entries, EMPTY_SET);
    const headers = items.filter(i => i.type === 'header');
    expect(headers).toHaveLength(2);
    expect(headers.find(h => h.title === 'Tech').count).toBe(2);
    expect(headers.find(h => h.title === 'Sports').count).toBe(1);
  });

  it('falls back to "All" for entries with no category', () => {
    const entries = [makeEntry(1, 10, 'Feed A', null)];
    const items = buildGroupedCatItems(entries, EMPTY_SET);
    expect(items[0]).toMatchObject({ type: 'header', title: 'All', feedId: 'All' });
  });

  it('sorts category headers alphabetically', () => {
    const entries = [
      makeEntry(1, 10, 'Feed X', 'Zebra'),
      makeEntry(2, 20, 'Feed Y', 'Alpha'),
      makeEntry(3, 30, 'Feed Z', 'Mango'),
    ];
    const headers = buildGroupedCatItems(entries, EMPTY_SET)
      .filter(i => i.type === 'header')
      .map(h => h.title);
    expect(headers).toEqual(['Alpha', 'Mango', 'Zebra']);
  });

  it('uses category title as feedId key on headers', () => {
    const entries = [makeEntry(1, 10, 'Feed A', 'News')];
    const header = buildGroupedCatItems(entries, EMPTY_SET)[0];
    expect(header.feedId).toBe('News');
  });

  it('collapses a category group when its title is in the collapsed set', () => {
    const entries = [
      makeEntry(1, 10, 'Feed A', 'Tech'),
      makeEntry(2, 20, 'Feed B', 'Tech'),
    ];
    const collapsed = new Set(['Tech']);
    const items = buildGroupedCatItems(entries, collapsed);
    expect(items).toHaveLength(1);
    expect(items[0]).toMatchObject({ type: 'header', collapsed: true, count: 2 });
  });

  it('expands a category group when its title is not in the collapsed set', () => {
    const entries = [makeEntry(1, 10, 'Feed A', 'Tech'), makeEntry(2, 20, 'Feed B', 'Tech')];
    const items = buildGroupedCatItems(entries, EMPTY_SET);
    expect(items).toHaveLength(3);
    expect(items[0].collapsed).toBe(false);
  });

  it('includes feed title in item sub', () => {
    const entry = makeEntry(1, 10, 'My Feed', 'Tech');
    const items = buildGroupedCatItems([entry], EMPTY_SET).filter(i => i.type === 'item');
    expect(items[0].sub).toContain('My Feed');
  });

  it('mixes entries across "All" and named categories correctly', () => {
    const entries = [
      makeEntry(1, 10, 'Feed A', 'Tech'),
      makeEntry(2, 20, 'Feed B', null),
      makeEntry(3, 30, 'Feed C', 'Tech'),
    ];
    const items = buildGroupedCatItems(entries, EMPTY_SET);
    const headers = items.filter(i => i.type === 'header').map(h => h.title);
    expect(headers).toContain('Tech');
    expect(headers).toContain('All');
  });
});
