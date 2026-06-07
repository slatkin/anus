<script>
  import { onMount, onDestroy, tick } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import { FetchCached, FetchEntries, RefreshAndFetch, FetchArticleContent, MarkRead, MarkUnread, ToggleStar, SaveEntry, OpenURL, Show, GetConfig, SaveConfig } from './api.js';
  import { BookOpen, Bookmark, ExternalLink, EyeOff, Minus, Plus, Settings } from 'lucide-svelte';
  import { COL_PAD, COL_GAP, COL_PAD_TOP, COL_PAD_BOT, calcCols, calcColWidth, calcContentWidth, calcPageStride, calcTotalPages } from './paging.js';

  const MODE_ENTRIES = 'entries';
  const MODE_FEEDS   = 'feeds';
  const FOCUS_LIST   = 'list';
  const FOCUS_READER = 'reader';

  let mode   = MODE_ENTRIES;
  let focus  = FOCUS_LIST;
  let loading = true;
  let error   = null;

  let allEntries  = [];
  let entries     = [];
  let feeds       = [];
  let cursor      = 0;
  let feedCursor  = 0;
  let filterFeedID = 0;
  let selectedEntry    = null;
  let selectedIdx      = -1;
  let originalContent  = null;
  let fetchingOriginal = false;

  let statusText    = 'Loading…';
  let statusTimeout = null;
  let now = Date.now();

  let toastMsg     = '';
  let toastVisible = false;
  let toastTimer   = null;

  function showToast(msg, ms = 3000) {
    toastMsg     = msg;
    toastVisible = true;
    clearTimeout(toastTimer);
    if (ms > 0) toastTimer = setTimeout(() => { toastVisible = false; }, ms);
  }

  let readerEl;
  let readerWidth = 0;
  let contentEl;
  let page = 0;
  let totalPages = 1;
  let pageStride = 0;
  let _measureTimer = null;
  let _measureId = 0;
  let itemEls = [];
  let showRead      = localStorage.getItem('showRead')      !== 'false';
  let sortOldest    = localStorage.getItem('sortOldest')    === 'true';
  let grouped       = localStorage.getItem('grouped')       !== 'false';
  let showScrollbar = localStorage.getItem('showScrollbar') === 'true';
  let navPaneEl = null;
  let thumbTop = 0;
  let thumbHeight = 0;

  function updateScrollThumb() {
    if (!navPaneEl) return;
    const { scrollTop, scrollHeight, clientHeight } = navPaneEl;
    thumbHeight = Math.max(30, (clientHeight / scrollHeight) * clientHeight);
    thumbTop = (scrollTop / (scrollHeight - clientHeight)) * (clientHeight - thumbHeight);
  }

  function onThumbMousedown(e) {
    const startY = e.clientY;
    const startScrollTop = navPaneEl.scrollTop;
    e.preventDefault();

    function onMove(e) {
      const { scrollHeight, clientHeight } = navPaneEl;
      navPaneEl.scrollTop = startScrollTop + (e.clientY - startY) * (scrollHeight / clientHeight);
    }
    function onUp() {
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
    }
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
  }
  let collapsedFeeds = new Set(JSON.parse(localStorage.getItem('collapsedFeeds') || '[]'));

  $: localStorage.setItem('showRead',      String(showRead));
  $: localStorage.setItem('sortOldest',    String(sortOldest));
  $: localStorage.setItem('grouped',       String(grouped));
  $: localStorage.setItem('showScrollbar', String(showScrollbar));
  $: localStorage.setItem('collapsedFeeds', JSON.stringify([...collapsedFeeds]));
  let navWidth     = parseInt(localStorage.getItem('navWidth') || '300', 10);
  let navCollapsed = localStorage.getItem('navCollapsed') === 'true';
  let fontSize     = parseInt(localStorage.getItem('readerFontSize') || '16', 10);
  let keptUnread = new Set(JSON.parse(localStorage.getItem('keptUnread') || '[]'));

  $: localStorage.setItem('navWidth',       String(navWidth));
  $: localStorage.setItem('navCollapsed',   String(navCollapsed));
  $: localStorage.setItem('readerFontSize', String(fontSize));
  $: localStorage.setItem('keptUnread', JSON.stringify([...keptUnread]));

  function toggleNav() {
    navCollapsed = !navCollapsed;
  }

  function increaseFontSize() { fontSize = Math.min(fontSize + 2, 28); }
  function decreaseFontSize() { fontSize = Math.max(fontSize - 2, 10); }

  function startNavResize(e) {
    e.preventDefault();
    const startX = e.clientX;
    const startW = navWidth;
    function onMove(ev) {
      navWidth = Math.max(160, Math.min(600, startW + ev.clientX - startX));
    }
    function onUp() {
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
    }
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
  }

  // Column layout — pure functions from paging.js; COL_PAD/COL_GAP applied via inline style
  // so the same constants are used for both rendering and measurement.
  $: cols         = calcCols(readerWidth);
  $: colWidth     = calcColWidth(readerWidth, cols);
  // column-width drives CSS multi-column instead of column-count — WebKit requires this
  // for single-column horizontal overflow pagination to work.
  $: contentWidth = calcContentWidth(cols, colWidth);

  function scheduleMeasure() {
    clearTimeout(_measureTimer);
    _measureTimer = setTimeout(measurePages, 50);
  }

  // Apply padding and gap measurements directly from JS constants — eliminates any
  // discrepancy between the values used in calculation and the values in the DOM.
  function applyMeasure() {
    if (!contentEl || !readerWidth) return;
    pageStride = calcPageStride(contentEl.clientWidth);
    if (pageStride <= 0) return;
    totalPages = calcTotalPages(contentEl.scrollWidth, pageStride);
    if (page >= totalPages) page = Math.max(0, totalPages - 1);

    contentEl.querySelectorAll('img').forEach(img => {
      if (!img.complete) {
        img.addEventListener('load',  scheduleMeasure, { once: true });
        img.addEventListener('error', scheduleMeasure, { once: true });
      }
    });
  }

  async function measurePages() {
    const id = ++_measureId;
    await tick();
    if (id !== _measureId || !contentEl || !readerWidth) return;
    // Double-RAF: gives browser time to complete CSS multi-column reflow.
    await new Promise(r => requestAnimationFrame(r));
    await new Promise(r => requestAnimationFrame(r));
    if (id !== _measureId || !contentEl || !readerWidth) return;

    applyMeasure();

    // Fallback: on initial load CSS multi-column reflow can lag past 2 frames.
    // Re-check once after a short delay; the _measureId guard prevents stale runs.
    setTimeout(() => { if (id === _measureId) applyMeasure(); }, 250);
  }

  // Reset page immediately when column count changes (window resize crossed boundary).
  $: { cols; page = 0; scheduleMeasure(); }

  // Remeasure on any layout-relevant change (including readability mode toggle).
  $: if (selectedEntry) (readerWidth, fontSize, originalContent, scheduleMeasure());

  $: filteredEntries = showRead
    ? entries
    : entries.filter(e => e.status === 'unread' || e.id === selectedEntry?.id);

  function entryOrder(a, b) {
    const pub = new Date(b.published_at || 0) - new Date(a.published_at || 0);
    if (pub !== 0) return pub;
    return new Date(b.fetched_at || 0) - new Date(a.fetched_at || 0);
  }
  $: displayEntries = sortOldest
    ? [...filteredEntries].sort((a, b) => -entryOrder(a, b))
    : [...filteredEntries].sort(entryOrder);

  // Clamp cursor when displayEntries shrinks (e.g. toggle flipped).
  $: if (mode === MODE_ENTRIES && !loading && cursor >= displayEntries.length) {
    cursor = Math.max(0, displayEntries.length - 1);
  }

  // Re-sync cursor to selectedEntry after displayEntries shifts (e.g. prev article
  // removed from filtered list when marked read, or background poll reorders entries).
  $: if (mode === MODE_ENTRIES && selectedEntry) {
    const synced = displayEntries.findIndex(e => e.id === selectedEntry.id);
    if (synced !== -1 && synced !== cursor) cursor = synced;
  }

  onMount(async () => {
    await loadCached();
    await tick();
    Show();
    fetchEntries(true);
    const cfg = await GetConfig().catch(() => null);
    const intervalMs = (cfg?.polling_interval_minutes ?? 10) > 0
      ? (cfg?.polling_interval_minutes ?? 10) * 60 * 1000
      : null;
    const poll  = intervalMs ? setInterval(() => fetchEntries(true), intervalMs) : null;
    const clock = setInterval(() => { now = Date.now(); }, 60 * 1000);
    return () => { if (poll) clearInterval(poll); clearInterval(clock); };
  });

  onDestroy(() => {
    clearTimeout(_measureTimer);
  });

  // ── data ──────────────────────────────────────────────────────────

  async function loadCached() {
    try {
      const result = await FetchCached();
      if (!result.entries?.length) return;
      allEntries = result.entries;
      feeds      = result.feeds ?? [];
      entries    = filterByFeed(allEntries, filterFeedID);
      loading    = false;
      refreshStatus();
      await tick();
      const savedId  = parseInt(localStorage.getItem('lastArticleId') || '0', 10);
      const savedIdx = savedId ? displayEntries.findIndex(e => e.id === savedId) : -1;
      openArticle(savedIdx !== -1 ? savedIdx : 0);
    } catch (_) {
      // cache unavailable — fall through to live fetch
    }
  }

  async function fetchEntries(background = false, doServerRefresh = false) {
    const prevIds    = new Set(allEntries.map(e => e.id));
    const isInitial  = prevIds.size === 0;
    if (!background) { loading = true; statusText = 'Loading…'; }
    if (!isInitial) showToast(doServerRefresh ? 'Polling feeds…' : 'Refreshing…', 0);
    error = null;
    try {
      const result = doServerRefresh ? await RefreshAndFetch() : await FetchEntries();
      allEntries = result.entries ?? [];
      feeds      = result.feeds   ?? [];
      entries    = filterByFeed(allEntries, filterFeedID);
      loading      = false;
      if (cursor >= entries.length) cursor = 0;
      refreshStatus();
      if (!isInitial) {
        const newCount = allEntries.filter(e => !prevIds.has(e.id)).length;
        if (newCount > 0) showToast(`${newCount} new article${newCount !== 1 ? 's' : ''} fetched`, 3500);
        else showToast('Up to date', 2000);
      }
      if (!selectedEntry && entries.length > 0) {
        await tick();
        const savedId  = parseInt(localStorage.getItem('lastArticleId') || '0', 10);
        const savedIdx = savedId ? displayEntries.findIndex(e => e.id === savedId) : -1;
        openArticle(savedIdx !== -1 ? savedIdx : 0);
      }
    } catch (e) {
      error      = String(e);
      statusText = 'Error: ' + error;
      loading    = false;
      if (!isInitial) showToast('Refresh failed', 4000);
    }
  }

  function filterByFeed(all, feedID) {
    return feedID ? all.filter(e => e.feed_id === feedID) : all;
  }

  // ── navigation ────────────────────────────────────────────────────

  // Navigable items in their current display order. In grouped mode this
  // follows group order; in flat mode it matches displayEntries order.
  function navOrder() {
    return displayItems.filter(item => item.type === 'item');
  }

  function moveDown() {
    if (mode === MODE_FEEDS) {
      if (feedCursor < feeds.length - 1) feedCursor++;
      cursor = feedCursor;
      scrollCursorIntoView();
      refreshStatus();
    } else {
      const items = navOrder();
      const pos = items.findIndex(item => item.cursorIdx === cursor);
      if (pos !== -1 && pos < items.length - 1) openArticle(items[pos + 1].cursorIdx);
    }
  }

  function moveUp() {
    if (mode === MODE_FEEDS) {
      if (feedCursor > 0) feedCursor--;
      cursor = feedCursor;
      scrollCursorIntoView();
      refreshStatus();
    } else {
      const items = navOrder();
      const pos = items.findIndex(item => item.cursorIdx === cursor);
      if (pos > 0) openArticle(items[pos - 1].cursorIdx);
    }
  }

  function moveFirst() {
    if (mode === MODE_FEEDS) {
      feedCursor = 0; cursor = 0;
      scrollCursorIntoView(); refreshStatus();
    } else {
      const items = navOrder();
      if (items.length > 0) openArticle(items[0].cursorIdx);
    }
  }

  function moveLast() {
    if (mode === MODE_FEEDS) {
      feedCursor = feeds.length - 1; cursor = feedCursor;
      scrollCursorIntoView(); refreshStatus();
    } else {
      const items = navOrder();
      if (items.length > 0) openArticle(items[items.length - 1].cursorIdx);
    }
  }

  function advanceToNextUnread() {
    const items = navOrder();
    const pos = items.findIndex(item => item.cursorIdx === cursor);
    for (let i = pos + 1; i < items.length; i++) {
      if (displayEntries[items[i].cursorIdx]?.status === 'unread') {
        openArticle(items[i].cursorIdx);
        return;
      }
    }
  }

  function markReadAndAdvance() {
    const entry = currentEntry();
    if (!entry) return;
    if (entry.status === 'unread') {
      mutateEntry(entry.id, e => ({ ...e, status: 'read' }));
      MarkRead([entry.id]).catch(() => {});
    }
    advanceToNextUnread();
  }

  async function scrollCursorIntoView() {
    await tick();
    itemEls[cursor]?.scrollIntoView({ block: 'nearest' });
  }

  function selectCurrent() {
    if (mode === MODE_FEEDS) selectFeed(feedCursor);
    else openArticle(cursor);
  }

  function goBack() {
    if (focus === FOCUS_READER) { focus = FOCUS_LIST; refreshStatus(); return; }
    if (mode === MODE_FEEDS)    { mode  = MODE_ENTRIES; refreshStatus(); return; }
    if (filterFeedID !== 0) {
      filterFeedID = 0;
      entries      = allEntries;
      mode         = MODE_FEEDS;
      cursor       = 0;
      refreshStatus();
    }
  }

  function showFeeds() {
    mode   = MODE_FEEDS;
    cursor = feedCursor;
    refreshStatus();
  }

  function selectFeed(idx) {
    if (idx >= feeds.length) return;
    const feed   = feeds[idx];
    filterFeedID = feed.feed_id;
    entries      = filterByFeed(allEntries, feed.feed_id);
    mode         = MODE_ENTRIES;
    feedCursor   = idx;
    cursor       = 0;
    refreshStatus();
  }

  function openArticle(idx) {
    if (idx < 0 || idx >= displayEntries.length) return;

    const prev = selectedEntry;
    if (prev && prev.status === 'unread' && !keptUnread.has(prev.id)) {
      mutateEntry(prev.id, e => ({ ...e, status: 'read' }));
      MarkRead([prev.id]).catch(() => {});
    }

    selectedIdx      = idx;
    cursor           = idx;
    focus            = FOCUS_READER;
    selectedEntry    = displayEntries[idx];
    originalContent  = null;
    fetchingOriginal = false;
    localStorage.setItem('lastArticleId', String(selectedEntry.id));
    page       = 0;
    totalPages = 1;
    scrollCursorIntoView();
    refreshStatus();
  }

  // ── actions ───────────────────────────────────────────────────────

  function toggleRead() {
    const entry = currentEntry();
    if (!entry) return;
    const newStatus = entry.status === 'read' ? 'unread' : 'read';
    mutateEntry(entry.id, e => ({ ...e, status: newStatus }));
    if (newStatus === 'read') {
      MarkRead([entry.id]).catch(() => {});
      keptUnread.delete(entry.id); keptUnread = keptUnread;
      advanceToNextUnread();
    } else {
      MarkUnread([entry.id]).catch(() => {});
      keptUnread.add(entry.id); keptUnread = keptUnread;
    }
  }

  function handleMailClick() {
    if (!selectedEntry) return;
    if (selectedEntry.status === 'unread') {
      if (keptUnread.has(selectedEntry.id)) keptUnread.delete(selectedEntry.id);
      else keptUnread.add(selectedEntry.id);
      keptUnread = keptUnread;
    } else {
      keptUnread.add(selectedEntry.id);
      keptUnread = keptUnread;
      mutateEntry(selectedEntry.id, e => ({ ...e, status: 'unread' }));
      MarkUnread([selectedEntry.id]).catch(() => {});
    }
  }

  function toggleStar() {
    const entry = currentEntry();
    if (!entry) return;
    mutateEntry(entry.id, e => ({ ...e, starred: !e.starred }));
    ToggleStar(entry.id).catch(() => {});
    setStatus('Starred', 2000);
  }

  function markAllRead() {
    const ids = entries.filter(e => e.status === 'unread' && !keptUnread.has(e.id)).map(e => e.id);
    if (!ids.length) return;
    const idSet = new Set(ids);
    entries    = entries.map(e => idSet.has(e.id) ? { ...e, status: 'read' } : e);
    allEntries = allEntries.map(e => idSet.has(e.id) ? { ...e, status: 'read' } : e);
    MarkRead(ids).catch(() => {});
    setStatus(`Marked ${ids.length} as read`, 2000);
  }

  function markFeedRead(feedId) {
    const ids = entries.filter(e => e.feed_id === feedId && e.status === 'unread' && !keptUnread.has(e.id)).map(e => e.id);
    if (!ids.length) return;
    const idSet = new Set(ids);
    entries    = entries.map(e => idSet.has(e.id) ? { ...e, status: 'read' } : e);
    allEntries = allEntries.map(e => idSet.has(e.id) ? { ...e, status: 'read' } : e);
    MarkRead(ids).catch(() => {});
  }

  function toggleFeedCollapse(feedId) {
    if (collapsedFeeds.has(feedId)) collapsedFeeds.delete(feedId);
    else collapsedFeeds.add(feedId);
    collapsedFeeds = collapsedFeeds;
  }

  function saveEntry() {
    const id = currentEntry()?.id;
    if (id) SaveEntry(id).then(() => setStatus('Saved', 2000)).catch(() => {});
  }

  function openBrowser() {
    const url = currentEntry()?.url;
    if (url) OpenURL(url);
  }

  async function fetchOriginal() {
    if (!selectedEntry) return;
    if (originalContent !== null) { originalContent = null; return; }
    if (fetchingOriginal) return;
    fetchingOriginal = true;
    showToast('Fetching original…', 0);
    try {
      originalContent = await FetchArticleContent(selectedEntry.id, selectedEntry.url);
      toastVisible = false;
    } catch (e) {
      showToast('Failed to fetch original', 3000);
    } finally {
      fetchingOriginal = false;
    }
  }

  function handleArticleClick(e) {
    const yt = e.target.closest('[data-yt-url]');
    if (yt) { e.preventDefault(); OpenURL(yt.dataset.ytUrl); return; }
    const a = e.target.closest('a[href]');
    if (a) { e.preventDefault(); OpenURL(a.href); }
  }

  function handleArticleImgError(e) {
    if (e.target.tagName === 'IMG') e.target.style.display = 'none';
  }

  function currentEntry() {
    return focus === FOCUS_READER ? selectedEntry : (displayEntries[cursor] ?? null);
  }

  function mutateEntry(id, fn) {
    entries    = entries.map(e => e.id === id ? fn(e) : e);
    allEntries = allEntries.map(e => e.id === id ? fn(e) : e);
    if (selectedEntry?.id === id) selectedEntry = fn(selectedEntry);
  }

  // ── status ────────────────────────────────────────────────────────

  function setStatus(msg, ms) {
    statusText = msg;
    clearTimeout(statusTimeout);
    statusTimeout = setTimeout(refreshStatus, ms);
  }

  function refreshStatus() {
    clearTimeout(statusTimeout);
    if (loading) { statusText = 'Loading…'; return; }
    const n   = (mode === MODE_FEEDS ? feeds : displayEntries).length;
    const cur = (mode === MODE_FEEDS ? feedCursor : cursor) + 1;
    if (focus === FOCUS_READER) {
      statusText = `[${selectedIdx + 1}/${displayEntries.length}]  ↑↓ prev/next  space mark read  b back  u read  s star  e save  o open  x original`;
    } else if (mode === MODE_FEEDS) {
      statusText = `${cur}/${n}  enter open  r refresh`;
    } else {
      statusText = `${cur}/${n}  enter open  ↑↓ navigate  space mark read  u toggle  s star  f feeds  r refresh`;
    }
  }

  // ── keyboard ──────────────────────────────────────────────────────

  function handleKeydown(e) {
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
    switch (e.key) {
      case 'ArrowDown':  e.preventDefault(); moveDown(); break;
      case 'ArrowUp':    e.preventDefault(); moveUp();   break;
      case 'Enter':      e.preventDefault(); selectCurrent(); break;
      case 'Escape': case 'Backspace': case 'b': e.preventDefault(); goBack(); break;
      case 'f': showFeeds(); break;
      case 'u': case 'm': toggleRead(); break;
      case 's': toggleStar(); break;
      case 'A': markAllRead(); break;
      case 'e': saveEntry(); break;
      case 'o': openBrowser(); break;
      case 'x': fetchOriginal(); break;
      case 'r': fetchEntries(false, true); break;
      case ' ':        e.preventDefault(); markReadAndAdvance(); break;
      case 'Home':     e.preventDefault(); moveFirst(); break;
      case 'End':      e.preventDefault(); moveLast();  break;
      case 'ArrowRight': e.preventDefault();
        if (page < totalPages - 1) page++;
        break;
      case 'ArrowLeft':  e.preventDefault();
        if (page > 0) page--;
        break;
      case '?': setStatus('↑↓ navigate  enter open  space mark read+next  b/esc back  u read  s star  A all-read  f feeds  e save  o browser  x original  r refresh', 5000); break;
    }
  }

  // ── display ───────────────────────────────────────────────────────

  function timeAgo(iso) {
    const ms   = Date.now() - new Date(iso).getTime();
    const min  = Math.floor(ms / 60000);
    const hr   = Math.floor(ms / 3600000);
    const day  = Math.floor(ms / 86400000);
    if (min < 1)  return 'just now';
    if (min < 60) return `${min}m ago`;
    if (hr  < 24) return `${hr}h ago`;
    if (day <  7) return `${day}d ago`;
    return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  }

  function fullDate(iso) {
    return new Date(iso).toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
  }

  function processContent(html) {
    if (!html) return html;
    html = html.replace(/<a\b[^>]*>\s*View Image in Fullscreen\s*<\/a>/gi, '');
    // Strip width/height attrs so hotlink-protected images (200 OK, blank body)
    // don't reserve space proportional to their declared dimensions.
    html = html.replace(/<img(\s[^>]*?)>/gi, (_, attrs) => {
      const cleaned = attrs.replace(/\s+(width|height)=["'][^"']*["']/gi, '');
      return `<img${cleaned}>`;
    });
    html = html.replace(
      /<iframe[^>]*src=["']https?:\/\/(?:www\.)?youtube(?:-nocookie)?\.com\/embed\/([a-zA-Z0-9_-]+)[^"']*["'][^>]*>(?:<\/iframe>)?/gi,
      (_, id) =>
        `<div class="yt-thumb" data-yt-url="https://www.youtube.com/watch?v=${id}">` +
        `<img src="https://img.youtube.com/vi/${id}/hqdefault.jpg" ` +
        `style="width:100%;aspect-ratio:16/9;object-fit:cover;display:block" alt="Watch on YouTube">` +
        `<span class="yt-play">▶ Watch on YouTube</span>` +
        `</div>`
    );
    // Clean up figure/figcaption: remove non-image nodes before <figcaption>
    // (some feeds duplicate caption text as siblings before the figcaption).
    const doc = new DOMParser().parseFromString('<body>' + html + '</body>', 'text/html');
    doc.querySelectorAll('figure').forEach(fig => {
      const cap = fig.querySelector(':scope > figcaption');
      if (!cap) return;
      for (const node of [...fig.childNodes]) {
        if (node === cap) break;
        const isImg = node.nodeType === 1 &&
          (node.tagName === 'IMG' || node.tagName === 'PICTURE' || node.tagName === 'VIDEO' ||
           (node.tagName === 'A' && node.querySelector('img, picture, video')));
        if (!isImg) node.remove();
      }
    });

    // Ars Technica (and similar feeds) emit images as bare <a><img></a> followed
    // by a plain-text caption node — no <figure> wrapper. The caption text also
    // repeats itself (sometimes multiple times) in the same text node.
    // Wrap each image link + its caption text in <figure><figcaption> so CSS
    // can style them, and strip the duplicated text.
    function deduplicateCaption(rawText) {
      const text = rawText.trim();
      const len = text.length;
      if (!len) return '';
      // Minimum prefix length to test — long enough to avoid coincidental matches.
      const minLen = Math.max(10, Math.floor(len * 0.08));
      for (let i = minLen; i <= Math.floor(len * 0.7); i++) {
        // If the text starting at position i begins with the same characters as
        // the text from position 0, the caption repeats — keep only the first copy.
        if (text.slice(i).trimStart().startsWith(text.slice(0, minLen))) {
          return text.slice(0, i).trimEnd();
        }
      }
      return text;
    }

    doc.querySelectorAll('a').forEach(link => {
      if (link.closest('figure')) return;              // already inside a figure
      if (!link.querySelector('img')) return;          // not an image link
      if (link.textContent.trim()) return;             // link has visible text (alt text etc.)
      const next = link.nextSibling;
      if (!next || next.nodeType !== 3) return;        // no following text node
      const caption = deduplicateCaption(next.textContent);
      if (!caption) return;

      const figure = doc.createElement('figure');
      link.parentNode.insertBefore(figure, link);
      figure.appendChild(link);
      const figcap = doc.createElement('figcaption');
      figcap.textContent = caption;
      figure.appendChild(figcap);
      next.remove();  // removes entire text node (incl. any orphan gallery captions within)
    });

    // Some feeds (e.g. plain-HTML sites) use <br> instead of <p> tags, which
    // produces very tight line spacing because our CSS only styles <p>. Walk
    // block-level containers and wrap inline segments (separated by <br>) into
    // <p> tags, while leaving existing block children in place.
    const IS_BLOCK = new Set(['P','DIV','H1','H2','H3','H4','H5','H6',
      'UL','OL','LI','TABLE','BLOCKQUOTE','FIGURE','FIGCAPTION','PRE',
      'SECTION','ARTICLE','HEADER','FOOTER','ASIDE','NAV']);
    doc.querySelectorAll('body, div, section, article').forEach(block => {
      // Only process blocks that have at least one direct <br> child.
      if (![...block.children].some(c => c.tagName === 'BR')) return;
      // Walk children: collect inline runs between <br> or block elements.
      const children = [...block.childNodes];
      const segments = []; // [{type:'inline'|'block', nodes:[]}]
      let run = [];
      for (const n of children) {
        const isBlock = n.nodeType === 1 && IS_BLOCK.has(n.tagName);
        const isBr    = n.nodeType === 1 && n.tagName === 'BR';
        if (isBr) {
          segments.push({type: 'inline', nodes: run}); run = [];
        } else if (isBlock) {
          if (run.length) { segments.push({type: 'inline', nodes: run}); run = []; }
          segments.push({type: 'block', nodes: [n]});
        } else {
          run.push(n);
        }
      }
      if (run.length) segments.push({type: 'inline', nodes: run});
      // Only rewrite if at least one inline segment has visible content.
      const hasInline = segments.some(s => s.type === 'inline' && s.nodes.some(n =>
        n.nodeType === 1 || (n.nodeType === 3 && n.textContent.trim())
      ));
      if (!hasInline) return;
      block.innerHTML = '';
      for (const seg of segments) {
        if (seg.type === 'block') {
          block.appendChild(seg.nodes[0]);
        } else {
          const hasContent = seg.nodes.some(n =>
            n.nodeType === 1 || (n.nodeType === 3 && n.textContent.trim())
          );
          if (hasContent) {
            const p = doc.createElement('p');
            seg.nodes.forEach(n => p.appendChild(n));
            block.appendChild(p);
          }
        }
      }
    });

    return doc.body.innerHTML;
  }

  $: activeCursor = mode === MODE_FEEDS ? feedCursor : cursor;

  function buildGroupedItems(entries) {
    const byFeed = new Map();
    const order = [];
    entries.forEach((e, idx) => {
      if (!byFeed.has(e.feed_id)) {
        byFeed.set(e.feed_id, { title: e.feed.title, rows: [] });
        order.push(e.feed_id);
      }
      byFeed.get(e.feed_id).rows.push({
        type: 'item',
        cursorIdx: idx,
        id:    e.id,
        title: e.title,
        sub:   (e.starred ? '★  ' : '') + timeAgo(e.published_at),
        unread: e.status === 'unread',
      });
    });
    order.sort((a, b) => byFeed.get(a).title.localeCompare(byFeed.get(b).title));
    const out = [];
    for (const feedId of order) {
      const { title, rows } = byFeed.get(feedId);
      const collapsed = collapsedFeeds.has(feedId);
      out.push({ type: 'header', title, feedId, collapsed });
      if (!collapsed) out.push(...rows);
    }
    return out;
  }

  $: displayItems, updateScrollThumb();
  $: displayItems = (now, collapsedFeeds, mode === MODE_FEEDS
    ? feeds.map((f, i) => ({
        type:      'item',
        cursorIdx: i,
        id:        f.feed_id,
        title:     f.feed_title,
        sub:       `${f.unread} unread`,
        unread:    f.unread > 0,
      }))
    : grouped
      ? buildGroupedItems(displayEntries)
      : displayEntries.map((e, idx) => ({
          type:      'item',
          cursorIdx: idx,
          id:        e.id,
          title:     e.title,
          sub:       (e.starred ? '★  ' : '') + e.feed.title + '  ·  ' + timeAgo(e.published_at),
          unread:    e.status === 'unread',
        })));

  // ── settings ──────────────────────────────────────────────────────────
  let settingsOpen = false;
  let settingsCfg  = null;
  let settingsSaving = false;

  async function openSettings() {
    try {
      settingsCfg = await GetConfig();
    } catch (e) {
      showToast('Failed to load config: ' + e, 4000);
      return;
    }
    settingsOpen = true;
  }

  async function saveSettings() {
    settingsSaving = true;
    try {
      await SaveConfig(settingsCfg);
      settingsOpen = false;
      showToast('Settings saved', 2500);
    } catch (e) {
      showToast('Save failed: ' + e, 4000);
    } finally {
      settingsSaving = false;
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="app">

  <div class="body">

  <div class="left-col" class:nav-collapsed={navCollapsed} style="width: {navCollapsed ? 'var(--collapsed-w)' : navWidth + 'px'}">

      <div class="toolbar toolbar-nav" class:nav-collapsed={navCollapsed}>
        <div class="nav-left">
          <div class="collapse-btn-wrap">
            <button class="nav-arrow-btn nav-collapse-btn" class:flipped={navCollapsed}
                    on:click={toggleNav}
                    title={navCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}>
              <div class="flip-front">
                <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
                  <path d="M11.92,19.92L4,12l7.92-7.92l1.41,1.42L7.83,11H22v2H7.83l5.5,5.5L11.92,19.92M4,12V4H2v16h2V12z"/>
                </svg>
              </div>
              <div class="flip-back">
                <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" style="transform: scaleX(-1)">
                  <path d="M11.92,19.92L4,12l7.92-7.92l1.41,1.42L7.83,11H22v2H7.83l5.5,5.5L11.92,19.92M4,12V4H2v16h2V12z"/>
                </svg>
              </div>
            </button>
          </div>
        </div>
        {#if !navCollapsed}
          <div class="mark-all-group">
            <button class="mark-all-btn" on:click={markAllRead}>mark all read</button>
            <span class="toolbar-sep"></span>
            <button class="toolbar-btn" on:click={() => fetchEntries(false, true)} title="Refresh (r)">↺</button>
          </div>
        {/if}
      </div>
      <div class="toolbar toolbar-filters nav-collapsible">
        <div class="toolbar-toggles">
          <button class="pill" class:active={grouped}    on:click={() => grouped    = !grouped}    title="Group by feed">group feeds</button>
          <button class="pill" class:active={showRead}   on:click={() => showRead   = !showRead}   title="Show or hide read articles">show read</button>
          <button class="pill" class:active={sortOldest} on:click={() => sortOldest = !sortOldest} title="Sort oldest first">oldest first</button>
        </div>
      </div>

      <div class="nav-pane-wrap nav-collapsible">
      <div class="nav-pane" bind:this={navPaneEl} on:scroll={updateScrollThumb}>
      {#if loading}
        <div class="nav-empty">Loading…</div>
      {:else if error}
        <div class="nav-empty nav-error">{error}</div>
      {:else if displayItems.length === 0}
        <div class="nav-empty">No unread articles</div>
      {:else}
        {#each displayItems as item}
          {#if item.type === 'header'}
            <div class="nav-feed-header" role="button" tabindex="0"
              on:dblclick={() => toggleFeedCollapse(item.feedId)}
              on:keydown={e => e.key === 'Enter' && toggleFeedCollapse(item.feedId)}>
              <span class="feed-header-title">{item.title}</span>
              {#if grouped && !item.collapsed}
                <button class="feed-mark-read" on:click|stopPropagation={() => markFeedRead(item.feedId)}>Mark read</button>
              {/if}
            </div>
          {:else}
            <div
              class="nav-item"
              role="button"
              tabindex="0"
              class:selected={item.cursorIdx === activeCursor}
              class:open={item.id === selectedEntry?.id && mode === MODE_ENTRIES}
              bind:this={itemEls[item.cursorIdx]}
              on:click={() => mode === MODE_FEEDS ? selectFeed(item.cursorIdx) : openArticle(item.cursorIdx)}
              on:keydown={e => e.key === 'Enter' && (mode === MODE_FEEDS ? selectFeed(item.cursorIdx) : openArticle(item.cursorIdx))}
            >
              <div class="nav-title" class:unread={item.unread}>{item.title}</div>
              <div class="nav-sub">{item.sub}</div>
            </div>
          {/if}
        {/each}
      {/if}
      </div><!-- /nav-pane -->
      {#if showScrollbar}
        <div class="custom-scrollbar">
          <div class="custom-scrollbar-thumb"
            style="top:{thumbTop}px; height:{thumbHeight}px"
            on:mousedown={onThumbMousedown}>
          </div>
        </div>
      {/if}
      </div><!-- /nav-pane-wrap -->

      <div class="toolbar toolbar-nav-bottom nav-collapsible">
        <div class="nav-ud-btns">
          <button class="nav-arrow-btn" on:click={moveUp}   title="Previous (↑)">↑</button>
          <button class="nav-arrow-btn" on:click={moveDown} title="Next (↓)">↓</button>
        </div>
        <div class="nav-bottom-spacer"></div>
        <button class="nav-arrow-btn" on:click={openSettings} title="Settings">
          <Settings size={14}/>
        </button>
      </div>

  </div><!-- /left-col -->

    <div class="splitter" role="separator" class:hidden={navCollapsed} class:web={import.meta.env.VITE_API !== 'wails'} on:mousedown={startNavResize}></div>

    <div class="reader-pane" bind:this={readerEl} bind:clientWidth={readerWidth}>
      {#if selectedEntry}
        <div class="reader-viewport" style="width: {contentWidth}px">
          <div class="reader-content"
               bind:this={contentEl}
               style="width: {contentWidth}px; column-width: {colWidth}px; column-gap: {COL_GAP}px; padding: {COL_PAD_TOP}px {COL_PAD}px {COL_PAD_BOT}px; height: 100%; transform: translateX(-{page * pageStride}px)">
            <h1 class="article-title">{selectedEntry.title}</h1>
            <div class="article-meta">{selectedEntry.feed.title}  ·  {fullDate(selectedEntry.published_at)}{selectedEntry.fetched_at ? '  ·  Fetched ' + timeAgo(selectedEntry.fetched_at) : ''}</div>
            <div class="reader-controls">
              <button class="ctrl-btn" class:active={originalContent !== null} on:click={fetchOriginal} title="Readability mode">
                <BookOpen size={14}/>
              </button>
              <div class="ctrl-sep"></div>
              <button class="ctrl-btn" on:click={decreaseFontSize} title="Decrease font size"><Minus size={13}/></button>
              <span class="ctrl-label">A</span>
              <button class="ctrl-btn" on:click={increaseFontSize} title="Increase font size"><Plus size={13}/></button>
              <div class="ctrl-sep"></div>
              <button class="ctrl-btn"
                      class:active={keptUnread.has(selectedEntry?.id)}
                      on:click={handleMailClick}
                      title={selectedEntry?.status === 'unread' ? 'Keep unread' : 'Mark as unread'}>
                <EyeOff size={14}/>
              </button>
              <button class="ctrl-btn" on:click={saveEntry} title="Save to Miniflux"><Bookmark size={14}/></button>
              <button class="ctrl-btn" on:click={openBrowser} title="Open in browser"><ExternalLink size={14}/></button>
            </div>
            <div class="article-body" role="presentation" style="font-size: {fontSize}px" on:click={handleArticleClick} on:keydown={handleArticleClick} on:error|capture={handleArticleImgError}>
              {@html processContent(originalContent ?? selectedEntry.content)}
            </div>
          </div>
        </div>
        <div class="bottom-pad-mask"></div>
        {#if toastVisible}
          <div class="toast" transition:fade={{ duration: 150 }}>{toastMsg}</div>
        {/if}
        {#if totalPages > 1}
          <div class="page-nav">
            <button class="page-btn" disabled={page === 0}
                    on:click={() => page--}>‹</button>
            <span class="page-indicator">{page + 1} / {totalPages}</span>
            <button class="page-btn" disabled={page === totalPages - 1}
                    on:click={() => page++}>›</button>
          </div>
        {/if}
      {:else}
        <div class="reader-empty">Select an article to read</div>
      {/if}
    </div>

  </div><!-- /body -->

{#if settingsOpen && settingsCfg}
  <div class="settings-backdrop" role="presentation"
    on:click|self={() => settingsOpen = false}
    on:keydown={e => e.key === 'Escape' && (settingsOpen = false)}
    transition:fade={{ duration: 150 }}>
    <div class="settings-modal" transition:fly={{ y: 20, duration: 180 }}>
      <div class="settings-header">
        <span class="settings-title">Settings</span>
        <button class="settings-close" on:click={() => settingsOpen = false}>✕</button>
      </div>
      <div class="settings-body">
        <label class="settings-label settings-row">
          <span>Display scrollbar in feed list</span>
          <button class="settings-toggle" class:on={showScrollbar} on:click={() => showScrollbar = !showScrollbar} role="switch" aria-checked={showScrollbar}></button>
        </label>
        <label class="settings-label">
          <span>Cache expiry (days)</span>
          <input class="settings-input settings-input-sm" type="number" min="1" bind:value={settingsCfg.cache_expiry_days}/>
        </label>
        <label class="settings-label">
          <span>Polling interval (minutes, 0 = off)</span>
          <input class="settings-input settings-input-sm" type="number" min="0" bind:value={settingsCfg.polling_interval_minutes}/>
        </label>
      </div>
      <div class="settings-footer">
        <button class="settings-save" on:click={saveSettings} disabled={settingsSaving}>
          {settingsSaving ? 'Saving…' : 'Save'}
        </button>
      </div>
    </div>
  </div>
{/if}

</div><!-- /app -->

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }

  :global(html, body) {
    height: 100%;
    overflow: hidden;
    font-family: 'Lexend Deca', system-ui, sans-serif;
    font-weight: 300;
  }

  :global(#app) { height: 100%; }

  /* ── layout ── */
  :global(:root) { --collapsed-w: 44px; }

  .app {
    display: flex;
    flex-direction: column;
    height: 100vh;
    overflow: hidden;
    background: transparent;
    color: #c0caf5;
  }

  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 14px;
    background: #24283b;
    border-bottom: 1px solid #414868;
    flex-shrink: 0;
  }

  .toolbar-toggles {
    display: flex;
    align-items: center;
    gap: 2px;
  }

  .pill {
    padding: 2px 7px;
    border-radius: 4px;
    border: none;
    background: transparent;
    color: #737aa2;
    font-family: inherit;
    font-size: 11px;
    font-weight: 300;
    cursor: pointer;
    letter-spacing: 0.02em;
    transition: background 120ms, color 120ms, transform 80ms;
  }
  .pill:hover        { background: #24283b; color: #a9b1d6; }
  .pill.active       { background: #414868; color: #c0caf5; }
  .pill:active       { transform: scale(0.92); background: #565f89; color: #c0caf5; }

  .toolbar-nav {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-right: 4px;
  }

  .nav-left { display: flex; gap: 2px; align-items: center; }
  .nav-ud-btns { display: flex; gap: 2px; }

  .toolbar-nav.nav-collapsed { align-items: center; justify-content: center; }
  .toolbar-nav.nav-collapsed .nav-left { justify-content: center; }

  .nav-arrow-btn {
    background: none;
    border: none;
    cursor: pointer;
    font-size: 14px;
    padding: 2px 7px;
    border-radius: 4px;
    color: #737aa2;
    font-family: inherit;
    line-height: 1.4;
    transition: background 80ms, color 80ms, transform 80ms;
  }
  .nav-arrow-btn:hover  { background: #24283b; color: #c0caf5; }
  .nav-arrow-btn:active { background: #414868; color: #c0caf5; transform: scale(0.88); }
  .collapse-btn-wrap { perspective: 200px; }
  .nav-collapse-btn {
    display: flex; align-items: center; justify-content: center;
    padding: 2px 5px; background: #2d3f76; color: #7aa2f7;
    position: relative; transform-style: preserve-3d;
    transition: transform 280ms cubic-bezier(0.4, 0, 0.2, 1), background 80ms, color 80ms;
  }
  .nav-collapse-btn.flipped { transform: rotateY(180deg); }
  .nav-collapse-btn:hover { background: #3d59a1 !important; color: #c0caf5 !important; }
  .flip-front, .flip-back { backface-visibility: hidden; display: flex; align-items: center; justify-content: center; }
  .flip-back { position: absolute; inset: 0; transform: rotateY(180deg); display: flex; align-items: center; justify-content: center; }

  .mark-all-btn {
    background: none;
    border: none;
    cursor: pointer;
    font-size: 11px;
    font-family: inherit;
    font-weight: 300;
    color: #737aa2;
    padding: 2px 4px;
    border-radius: 4px;
    letter-spacing: 0.02em;
    transition: color 80ms, transform 80ms;
  }
  .mark-all-btn:hover  { color: #f7768e; }
  .mark-all-btn:active { color: #f7768e; transform: scale(0.92); }

  .mark-all-group {
    display: flex;
    align-items: center;
    gap: 0;
  }

  .toolbar-sep {
    display: inline-block;
    width: 1px;
    height: 12px;
    background: #3b3f5c;
    margin: 0 2px;
    vertical-align: middle;
  }

  .toolbar-btn {
    background: none;
    border: none;
    cursor: pointer;
    font-size: 18px;
    line-height: 1;
    padding: 2px 6px;
    border-radius: 4px;
    color: #737aa2;
    transition: background 80ms, color 80ms, transform 80ms;
  }
  .toolbar-btn:hover  { background: #414868; color: #c0caf5; }
  .toolbar-btn:active { background: #565f89; color: #c0caf5; transform: scale(0.88); }

  .body {
    display: flex;
    flex: 1;
    overflow: hidden;
    min-height: 0;
  }

  .left-col {
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    overflow: hidden;
    background: #1a1b26;
    transition: width 280ms cubic-bezier(0.4, 0, 0.2, 1);
    will-change: width;
    position: relative;
  }

  .nav-collapsible {
    opacity: 1;
    pointer-events: auto;
    transition: opacity 180ms ease;
  }
  .left-col.nav-collapsed .nav-collapsible {
    opacity: 0;
    pointer-events: none;
  }

  .splitter {
    width: 7px;
    flex-shrink: 0;
    cursor: col-resize;
    background: transparent;
    transition: background 120ms ease;
  }
  .splitter.web { background: #1a1b26; }
  .splitter:hover, .splitter:active { background: rgba(122, 162, 247, 0.25); }
  .splitter.hidden { width: 0; pointer-events: none; }

  /* ── nav pane ── */
  .nav-pane-wrap {
    position: relative;
    flex: 1;
    min-height: 0;
    display: flex;
  }
  .nav-pane {
    flex: 1;
    overflow-y: auto;
    background: #1a1b26;
    min-height: 0;
    scrollbar-width: none;
  }
  :global(.nav-pane::-webkit-scrollbar) { display: none; }
  .custom-scrollbar {
    position: absolute;
    right: 0;
    top: 0;
    bottom: 0;
    width: 4px;
    background: #1a1b26;
    z-index: 10;
    flex-shrink: 0;
  }
  .custom-scrollbar-thumb {
    position: absolute;
    width: 4px;
    background: #c0caf5;
    border-radius: 4px;
    cursor: pointer;
    user-select: none;
  }

  .nav-feed-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    padding: 8px 14px 5px;
    font-size: 12px;
    font-weight: 400;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: #7aa2f7;
    background: #24283b;
    border-bottom: 1px solid #414868;
    position: sticky;
    top: 0;
    z-index: 1;
    cursor: default;
    user-select: none;
  }

  .feed-header-title {
    min-width: 0;
    flex: 1;
  }

  .feed-mark-read {
    font-size: 10px;
    font-family: inherit;
    font-weight: 300;
    letter-spacing: 0.02em;
    text-transform: none;
    color: #737aa2;
    background: none;
    border: none;
    cursor: pointer;
    padding: 0;
    flex-shrink: 0;
  }
  .feed-mark-read:hover { color: #f7768e; }

  .nav-empty {
    padding: 20px 14px;
    font-size: 13px;
    color: #737aa2;
  }
  .nav-error { color: #f7768e; }

  .nav-item {
    padding: 9px 14px 8px;
    border-bottom: 1px solid #24283b;
    cursor: pointer;
    user-select: none;
    transition: background 0.08s;
  }
  .nav-item:hover    { background: #24283b; }
  .nav-item.selected { background: #414868; }
  .nav-item.open     { background: #292e42; }

  .nav-title {
    font-size: 13px;
    font-weight: 400;
    line-height: 1.35;
    color: #737aa2;
    word-break: break-word;
  }
  .nav-title.unread {
    font-weight: 400;
    color: #c0caf5;
  }

  .nav-sub {
    font-size: 11px;
    color: #414868;
    margin-top: 3px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .nav-item.selected .nav-sub { color: #7a89b8; }
  .nav-item.open     .nav-sub { color: #6b7499; }
  .nav-item.open     .nav-title { color: #73daca; }

  .bottom-pad-mask {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 64px;
    background: #fdf6e3;
    pointer-events: none;
  }

  /* ── reader pane ── */
  .reader-pane {
    flex: 1;
    overflow: hidden;
    background: #fdf6e3;
    color: #5c4b36;
    min-width: 0;
    position: relative;
  }

  .page-nav {
    position: absolute;
    bottom: 16px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 8px;
    background: rgba(253, 246, 227, 0.85);
    backdrop-filter: blur(4px);
    border: 1px solid #d5c9a8;
    border-radius: 20px;
    padding: 4px 10px;
    user-select: none;
  }

  .page-btn {
    background: none;
    border: none;
    cursor: pointer;
    font-size: 20px;
    line-height: 1;
    color: #8a7355;
    padding: 0 4px;
    transition: color 80ms, transform 80ms;
  }
  .page-btn:disabled { color: #c9b89a; cursor: default; }
  .page-btn:not(:disabled):hover  { color: #3a2c1a; }
  .page-btn:not(:disabled):active { color: #3a2c1a; transform: scale(0.85); }

  .page-indicator {
    font-size: 12px;
    color: #7a6345;
    min-width: 40px;
    text-align: center;
  }

  .reader-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    font-size: 14px;
    color: #b8a07a;
  }

  .reader-viewport {
    height: 100%;
    overflow: hidden;
    margin: 0 auto;
  }

  .reader-content {
    height: 100%;
    /* padding and column-gap set via inline style from JS constants */
    column-fill: auto;
    orphans: 3;
    widows: 3;
    transition: transform 200ms ease;
    will-change: transform;
  }

  .reader-content .article-body :global(h1),
  .reader-content .article-body :global(h2),
  .reader-content .article-body :global(h3),
  .reader-content .article-body :global(h4) {
    break-after: avoid;
  }

  .reader-content .article-body :global(pre),
  .reader-content .article-body :global(blockquote),
  .reader-content .article-body :global(figure) {
    break-inside: avoid;
  }

  .article-title {
    font-size: 26px;
    font-weight: 700;
    line-height: 1.3;
    color: #3a2c1a;
    margin-bottom: 2px;
  }

  .article-meta {
    font-size: 12px;
    color: #9a7a58;
    margin-bottom: 0;
  }
  .reader-controls {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 0;
    margin-bottom: 20px;
    border-top: 1px dotted #c4a882;
    margin-top: 10px;
  }
  .ctrl-sep { width: 1px; height: 14px; background: #c4a882; margin: 0 2px; }
  .ctrl-label { font-size: 13px; color: #9a7a58; font-weight: 600; line-height: 1; }
  .ctrl-btn {
    background: none;
    border: none;
    padding: 3px 4px;
    color: #9a7a58;
    cursor: pointer;
    border-radius: 3px;
    display: flex;
    align-items: center;
  }
  .ctrl-btn:hover { background: #2a1f14; color: #c4a882; }
  .ctrl-btn.active { background: #9a7a58; color: #fdf6e3; border-radius: 4px; }

  /* ── article body (global: rendered HTML) ── */
  .article-body :global(p) {
    font-size: inherit;
    line-height: 1.75;
    margin-bottom: 1.1em;
  }
  .article-body :global(a) {
    color: #8b5e3c;
    text-decoration: underline;
    text-underline-offset: 2px;
  }
  .article-body :global(a:hover) { color: #5c3a1e; }

  .article-body :global(img) {
    max-width: 75%;
    height: auto;
    display: block;
    margin: 1.2em auto;
    border-radius: 4px;
  }

  .article-body :global(h1),
  .article-body :global(h2),
  .article-body :global(h3),
  .article-body :global(h4),
  .article-body :global(h5),
  .article-body :global(h6) {
    font-weight: 700;
    color: #3a2c1a;
    line-height: 1.3;
    margin: 1.4em 0 0.5em;
  }
  .article-body :global(h1) { font-size: 22px; }
  .article-body :global(h2) { font-size: 19px; }
  .article-body :global(h3) { font-size: 17px; }
  .article-body :global(h4),
  .article-body :global(h5),
  .article-body :global(h6) { font-size: 15px; }

  .article-body :global(blockquote) {
    border-left: 3px solid #c4a882;
    margin: 1.2em 0;
    padding: 2px 0 2px 16px;
    color: #7a6248;
    font-style: italic;
  }

  .article-body :global(pre) {
    background: #f0e4c8;
    border: 1px solid #d6c4a0;
    border-radius: 5px;
    padding: 14px 18px;
    overflow-x: auto;
    font-size: 13px;
    line-height: 1.5;
    margin: 1.2em 0;
    font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  }
  .article-body :global(code) {
    background: #f0e4c8;
    border-radius: 3px;
    padding: 1px 5px;
    font-size: 13px;
    font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  }
  .article-body :global(pre code) { background: none; padding: 0; }

  .article-body :global(ul),
  .article-body :global(ol) { margin: 0.8em 0 0.8em 1.6em; }

  .article-body :global(li) {
    font-size: 16px;
    line-height: 1.75;
    margin-bottom: 0.25em;
  }

  .article-body :global(hr) {
    border: none;
    border-top: 1px solid #dac8a8;
    margin: 1.8em 0;
  }

  .article-body :global(table) {
    border-collapse: collapse;
    width: 100%;
    margin: 1.2em 0;
    font-size: 14px;
  }
  .article-body :global(th),
  .article-body :global(td) {
    border: 1px solid #d6c4a0;
    padding: 7px 12px;
    text-align: left;
  }
  .article-body :global(th) { background: #f0e4c8; font-weight: 700; }

  .article-body :global(figure) { margin: 1.2em auto; max-width: 85%; }

  .article-body :global(.yt-thumb) {
    display: block;
    position: relative;
    width: 85%;
    margin: 1.4em auto;
    break-inside: avoid;
    border-radius: 4px;
    overflow: hidden;
    cursor: pointer;
  }
  .article-body :global(.yt-play) {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    padding: 6px 10px;
    background: rgba(0,0,0,0.55);
    color: #fff;
    font-size: 13px;
    text-align: center;
  }
  .article-body :global(.yt-thumb:hover .yt-play) { background: rgba(0,0,0,0.75); }
  .article-body :global(figcaption) {
    font-size: 11px;
    color: #b8997a;
    font-style: italic;
    text-align: center;
    margin-top: 5px;
    line-height: 1.4;
  }

  /* ── toast ── */
  .toast {
    position: absolute;
    top: 16px;
    left: 50%;
    transform: translateX(-50%);
    background: rgba(36, 40, 59, 0.92);
    backdrop-filter: blur(4px);
    color: #c0caf5;
    font-size: 13px;
    padding: 8px 18px;
    border-radius: 20px;
    border: 1px solid #414868;
    white-space: nowrap;
    pointer-events: none;
    z-index: 20;
  }

  .toolbar-nav-bottom {
    border-top: 1px solid #414868;
    border-bottom: none;
    justify-content: flex-end;
    padding: 6px 4px;
  }

  .nav-bottom-spacer { flex: 1; }

  .settings-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.55);
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .settings-modal {
    background: #1a1b26;
    border: 1px solid #414868;
    border-radius: 8px;
    width: 420px;
    max-width: 95vw;
    max-height: 85vh;
    display: flex;
    flex-direction: column;
    box-shadow: 0 8px 32px rgba(0,0,0,0.5);
  }

  .settings-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    border-bottom: 1px solid #414868;
    flex-shrink: 0;
  }

  .settings-title {
    font-size: 15px;
    font-weight: 500;
    color: #fdfdfd;
    letter-spacing: 0.03em;
  }

  .settings-close {
    background: none;
    border: none;
    cursor: pointer;
    color: #737aa2;
    font-size: 14px;
    padding: 2px 6px;
    border-radius: 4px;
    transition: color 80ms, background 80ms;
  }
  .settings-close:hover { background: #24283b; color: #c0caf5; }

  .settings-body {
    overflow-y: auto;
    padding: 14px 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    flex: 1;
  }

  .settings-label {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 13px;
    color: #fdfdfd;
    letter-spacing: 0.02em;
  }

  .settings-row {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }



  .settings-input {
    background: #24283b;
    border: 1px solid #414868;
    border-radius: 4px;
    color: #c0caf5;
    font-family: inherit;
    font-size: 14px;
    padding: 6px 10px;
    outline: none;
    transition: border-color 120ms;
  }
  .settings-input:focus { border-color: #7aa2f7; }

  .settings-input-sm { width: 80px; }

  .settings-toggle {
    position: relative;
    width: 36px;
    height: 20px;
    border-radius: 10px;
    border: none;
    background: #414868;
    cursor: pointer;
    flex-shrink: 0;
    transition: background 150ms;
    padding: 0;
  }
  .settings-toggle.on { background: #7aa2f7; }
  .settings-toggle::after {
    content: '';
    position: absolute;
    top: 3px;
    left: 3px;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: #fdfdfd;
    transition: transform 150ms;
  }
  .settings-toggle.on::after { transform: translateX(16px); }


  .settings-footer {
    padding: 10px 16px;
    border-top: 1px solid #414868;
    display: flex;
    justify-content: flex-end;
    flex-shrink: 0;
  }

  .settings-save {
    background: #2d3f76;
    border: none;
    border-radius: 4px;
    color: #7aa2f7;
    cursor: pointer;
    font-family: inherit;
    font-size: 14px;
    font-weight: 500;
    padding: 6px 18px;
    transition: background 80ms, color 80ms;
  }
  .settings-save:hover:not(:disabled)  { background: #3d59a1; color: #c0caf5; }
  .settings-save:disabled { opacity: 0.5; cursor: default; }

</style>
