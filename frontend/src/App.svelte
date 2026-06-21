<script>
  import { onMount, onDestroy, tick } from 'svelte';
  import { FetchCached, FetchEntries, RefreshAndFetch, ClearCache, FetchArticleContent, MarkRead, MarkUnread, ToggleStar, SaveEntry, SearchEntries, OpenURL, Show, GetConfig, SaveConfig } from './api.js';
  import { ChevronsDownUp, ChevronsUpDown, Search, Settings } from 'lucide-svelte';
  import NavFeedHeader from './components/NavFeedHeader.svelte';
  import NavItem from './components/NavItem.svelte';
  import Toast from './components/Toast.svelte';
  import PageNav from './components/PageNav.svelte';
  import SearchBar from './components/SearchBar.svelte';
  import ReaderControls from './components/ReaderControls.svelte';
  import SettingsModal from './components/SettingsModal.svelte';
  import { COL_PAD, COL_GAP, COL_PAD_TOP, COL_PAD_BOT, calcCols, calcColWidth, calcContentWidth, calcPageStride, calcTotalPages } from './paging.js';
  import { EMPTY_SET, feedKey, catKey, buildGroupedItems as _buildGroupedItems, buildGroupedCatItems as _buildGroupedCatItems } from './grouping.js';
  import { timeAgo, fullDate } from './utils/date.js';
  import { processContent, highlightTerms } from './utils/content.js';
  import { showRead, sortOldest, grouped, groupedCats, showScrollbar, navWidth, navCollapsed, fontSize, collapsedFeeds, keptUnread } from './stores/preferences.js';

  const MODE_ENTRIES = 'entries';
  const MODE_FEEDS   = 'feeds';
  const FOCUS_LIST   = 'list';
  const FOCUS_READER = 'reader';

  let mode   = MODE_ENTRIES;
  let focus  = FOCUS_LIST;
  let loading = true;
  let error   = null;

  let searchOpen      = false;
  let searchQuery     = '';
  let searchResults   = null;
  let searchFired     = false;
  let searchDebounce;
  let searchInputEl;
  $: searchInputColor = !searchFired ? '#e0af68' : (searchResults?.length ? '#9ece6a' : '#f7768e');

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
  let readerHeight = 0;
  let contentEl;
  let page = 0;
  let totalPages = 1;
  let pageStride = 0;
  let _measureTimer = null;
  let _measureId = 0;
  let itemEls = [];
  let navPaneEl = null;
  let thumbTop = 0;
  let thumbHeight = 0;
  let stickyOffset = 0;
  let needsScroll = false;

  function updateScrollThumb() {
    if (!navPaneEl) return;
    const { scrollTop, scrollHeight, clientHeight } = navPaneEl;
    needsScroll = scrollHeight > clientHeight;
    const header = navPaneEl.querySelector('.nav-feed-header');
    stickyOffset = header ? header.offsetHeight : 0;
    const trackTop = stickyOffset + 2;
    const trackH = clientHeight - trackTop - 2;
    thumbHeight = Math.max(30, (clientHeight / scrollHeight) * trackH);
    thumbTop = trackTop + (scrollTop / (scrollHeight - clientHeight)) * (trackH - thumbHeight);
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
  function toggleNav() {
    $navCollapsed = !$navCollapsed;
  }

  function increaseFontSize() { $fontSize = Math.min($fontSize + 2, 28); }
  function decreaseFontSize() { $fontSize = Math.max($fontSize - 2, 10); }

  function startNavResize(e) {
    e.preventDefault();
    const startX = e.clientX;
    const startW = $navWidth;
    function onMove(ev) {
      $navWidth = Math.max(160, Math.min(600, startW + ev.clientX - startX));
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
    pageStride = calcPageStride(cols, colWidth);
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

  // An unbreakable figure/image that doesn't fit at the bottom of a column jumps whole to
  // the next column, leaving a gap. With sequential column-fill the text before it fills
  // the prior column independently of the media's size, so the leftover gap is stable —
  // shrinking the media to fit it pulls the block back up. Single pass, bounded by a
  // min-scale floor so we never shrink media into a postage stamp.
  const FIT_FLOOR   = 0.75;  // don't shrink media below this fraction of its fitted height
  const FIT_MIN_GAP = 80;    // ignore trivial gaps (px)
  const FIT_FUDGE   = 8;     // small px cushion so the refit reliably clears the gap

  async function fitColumnFigures(id) {
    const body = contentEl?.querySelector('.article-body');
    if (!body) return;

    // The block that flows in the column is the media's top-level ancestor under body.
    const topBlock = (el) => {
      let n = el;
      while (n.parentElement && n.parentElement !== body) n = n.parentElement;
      return n;
    };

    // Candidate media: figures and bare images, keyed by their flow block.
    const seen = new Set();
    const candidates = [];
    for (const img of body.querySelectorAll('img')) {
      const block = topBlock(img.closest('figure') ?? img);
      if (seen.has(block)) continue;
      seen.add(block);
      candidates.push({ block, img });
    }
    if (!candidates.length) return;

    // Reset to baseline so the pass is idempotent (recompute from natural size each run).
    let reset = false;
    for (const { img } of candidates) {
      if (img.style.maxHeight) { img.style.maxHeight = ''; reset = true; }
    }
    if (reset) {
      await new Promise(r => requestAnimationFrame(r));
      if (id !== _measureId) return;
    }

    const colContentH = readerHeight - COL_PAD_TOP - COL_PAD_BOT;
    if (colContentH <= 0) return;
    const colBottom = COL_PAD_TOP + colContentH;
    const base   = contentEl.getBoundingClientRect();
    const stride = colWidth + COL_GAP;
    const colOf  = (left) => Math.round((left - base.left - COL_PAD) / stride);
    const children = [...body.children];

    for (const { block, img } of candidates) {
      const br     = block.getBoundingClientRect();
      const figCol = colOf(br.left);
      if (figCol <= 0) continue;

      // Block must start its column (its previous sibling lives in an earlier column).
      const prev = block.previousElementSibling;
      let prevMaxCol = -1;
      if (prev) for (const rect of prev.getClientRects()) prevMaxCol = Math.max(prevMaxCol, colOf(rect.left));
      if (prevMaxCol >= figCol) continue;

      // Last content bottom in the previous column (handles fragmented blocks via getClientRects).
      const prevCol = figCol - 1;
      let lastBottom = -Infinity;
      for (const child of children) {
        if (child === block) break;
        for (const rect of child.getClientRects()) {
          if (colOf(rect.left) === prevCol) lastBottom = Math.max(lastBottom, rect.bottom - base.top);
        }
      }
      if (lastBottom === -Infinity) continue;
      const gap = colBottom - lastBottom;
      if (gap < FIT_MIN_GAP) continue;

      const imgH      = img.getBoundingClientRect().height;
      const blockH    = br.height;
      // Overhead the image must share the gap with: caption + internal spacing (blockH-imgH)
      // PLUS the block's own top+bottom margins (outside the border box) + a small cushion.
      const cs        = getComputedStyle(block);
      const margins   = parseFloat(cs.marginTop) + parseFloat(cs.marginBottom);
      const overhead  = (blockH - imgH) + margins + FIT_FUDGE;
      const targetImgH = gap - overhead;
      if (targetImgH >= imgH)            continue;
      if (targetImgH < imgH * FIT_FLOOR) continue;
      img.style.maxHeight = Math.floor(targetImgH) + 'px';
    }
    await new Promise(r => requestAnimationFrame(r));  // let the shrink reflow settle
  }

  async function measurePages() {
    const id = ++_measureId;
    await tick();
    if (id !== _measureId || !contentEl || !readerWidth) return;
    // Await font loading so web fonts don't reflow columns after measurement.
    if (document.fonts?.ready) await document.fonts.ready;
    // Double-RAF: gives browser time to complete CSS multi-column reflow.
    await new Promise(r => requestAnimationFrame(r));
    await new Promise(r => requestAnimationFrame(r));
    if (id !== _measureId || !contentEl || !readerWidth) return;

    await fitColumnFigures(id);
    if (id !== _measureId || !contentEl || !readerWidth) return;

    applyMeasure();

    // Fallback: on initial load CSS multi-column reflow can lag past 2 frames.
    // Re-check once after a short delay; the _measureId guard prevents stale runs.
    setTimeout(() => { if (id === _measureId) applyMeasure(); }, 250);
  }

  // Reset page immediately when column count changes (window resize crossed boundary).
  $: { cols; page = 0; scheduleMeasure(); }

  // Remeasure on any layout-relevant change (including readability mode toggle or vertical resize).
  $: if (selectedEntry) (readerWidth, readerHeight, $fontSize, originalContent, scheduleMeasure());

  $: filteredEntries = $showRead
    ? entries
    : entries.filter(e => e.status === 'unread' || e.id === selectedEntry?.id);

  function entryOrder(a, b) {
    const pub = new Date(b.published_at || 0) - new Date(a.published_at || 0);
    if (pub !== 0) return pub;
    return new Date(b.fetched_at || 0) - new Date(a.fetched_at || 0);
  }
  $: displayEntries = $sortOldest
    ? [...filteredEntries].sort((a, b) => -entryOrder(a, b))
    : [...filteredEntries].sort(entryOrder);

  // activeEntries is the source of truth for navigation when search is active.
  $: activeEntries = searchResults !== null ? searchResults : displayEntries;

  // Clamp cursor when activeEntries shrinks (e.g. toggle flipped).
  $: if (mode === MODE_ENTRIES && !loading && searchResults === null && cursor >= displayEntries.length) {
    cursor = Math.max(0, displayEntries.length - 1);
  }

  // Re-sync cursor to selectedEntry after displayEntries shifts (e.g. prev article
  // removed from filtered list when marked read, or background poll reorders entries).
  $: if (mode === MODE_ENTRIES && selectedEntry && searchResults === null) {
    const synced = displayEntries.findIndex(e => e.id === selectedEntry.id);
    if (synced !== -1 && synced !== cursor) cursor = synced;
  }

  let windowFocused = true;
  onMount(async () => {
    const onBlur      = () => { windowFocused = false; };
    const onFocus     = () => { windowFocused = true; };
    const onKeydown   = () => { document.body.style.cursor = 'none'; windowFocused = false; };
    const onMousemove = () => { document.body.style.cursor = ''; windowFocused = true; };
    window.addEventListener('blur',      onBlur);
    window.addEventListener('focus',     onFocus);
    window.addEventListener('keydown',   onKeydown);
    window.addEventListener('mousemove', onMousemove);
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
        const firstUnread = activeEntries.findIndex(e => e.status === 'unread');
        openArticle(firstUnread !== -1 ? firstUnread : 0);
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
      if (activeEntries[items[i].cursorIdx]?.status === 'unread') {
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

  // Expand the collapsed group containing an entry so it can be scrolled to / shown.
  function expandGroupFor(entry) {
    if (!entry) return;
    const k = $grouped ? feedKey(entry) : $groupedCats ? catKey(entry) : null;
    if (k && $collapsedFeeds.has(k))
      $collapsedFeeds = new Set([...$collapsedFeeds].filter(x => x !== k));
  }

  async function scrollCursorIntoView() {
    if (mode === MODE_ENTRIES) expandGroupFor(activeEntries[cursor]);
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
    if (idx < 0 || idx >= activeEntries.length) return;

    const prev = selectedEntry;
    if (prev && prev.status === 'unread' && !$keptUnread.has(prev.id)) {
      mutateEntry(prev.id, e => ({ ...e, status: 'read' }));
      MarkRead([prev.id]).catch(() => {});
    }

    selectedIdx      = idx;
    cursor           = idx;
    focus            = FOCUS_READER;
    selectedEntry    = activeEntries[idx];
    originalContent  = null;
    fetchingOriginal = false;
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
      $keptUnread.delete(entry.id); $keptUnread = $keptUnread;
      advanceToNextUnread();
    } else {
      MarkUnread([entry.id]).catch(() => {});
      $keptUnread.add(entry.id); $keptUnread = $keptUnread;
    }
  }

  function handleMailClick() {
    if (!selectedEntry) return;
    if (selectedEntry.status === 'unread') {
      if ($keptUnread.has(selectedEntry.id)) $keptUnread.delete(selectedEntry.id);
      else $keptUnread.add(selectedEntry.id);
      $keptUnread = $keptUnread;
    } else {
      $keptUnread.add(selectedEntry.id);
      $keptUnread = $keptUnread;
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
    const ids = entries.filter(e => e.status === 'unread' && !$keptUnread.has(e.id)).map(e => e.id);
    if (!ids.length) return;
    const idSet = new Set(ids);
    entries    = entries.map(e => idSet.has(e.id) ? { ...e, status: 'read' } : e);
    allEntries = allEntries.map(e => idSet.has(e.id) ? { ...e, status: 'read' } : e);
    MarkRead(ids).catch(() => {});
    setStatus(`Marked ${ids.length} as read`, 2000);
  }

  function markFeedRead(feedId) {
    const ids = entries.filter(e => e.feed_id === feedId && e.status === 'unread' && !$keptUnread.has(e.id)).map(e => e.id);
    if (!ids.length) return;
    const idSet = new Set(ids);
    entries    = entries.map(e => idSet.has(e.id) ? { ...e, status: 'read' } : e);
    allEntries = allEntries.map(e => idSet.has(e.id) ? { ...e, status: 'read' } : e);
    MarkRead(ids).catch(() => {});
  }

  function markCatRead(catTitle) {
    const ids = entries.filter(e => (e.feed.category?.title || 'All') === catTitle && e.status === 'unread' && !$keptUnread.has(e.id)).map(e => e.id);
    if (!ids.length) return;
    const idSet = new Set(ids);
    entries    = entries.map(e => idSet.has(e.id) ? { ...e, status: 'read' } : e);
    allEntries = allEntries.map(e => idSet.has(e.id) ? { ...e, status: 'read' } : e);
    MarkRead(ids).catch(() => {});
  }

  async function toggleFeedCollapse(feedId) {
    const isCollapsing = !$collapsedFeeds.has(feedId);

    if (isCollapsing) {
      const headerIdx = displayItems.findIndex(i => i.type === 'header' && i.feedId === feedId);
      let nextCursorIdx = null;
      if (headerIdx !== -1) {
        let pastThisGroup = false;
        for (let i = headerIdx + 1; i < displayItems.length; i++) {
          const item = displayItems[i];
          if (item.type === 'header') { pastThisGroup = true; continue; }
          if (item.type === 'item' && pastThisGroup) { nextCursorIdx = item.cursorIdx; break; }
        }
      }

      $collapsedFeeds.add(feedId);
      $collapsedFeeds = $collapsedFeeds;

      if (nextCursorIdx !== null) {
        await tick();
        await new Promise(r => requestAnimationFrame(r));
        itemEls[nextCursorIdx]?.scrollIntoView({ block: 'start' });
      }
    } else {
      $collapsedFeeds.delete(feedId);
      $collapsedFeeds = $collapsedFeeds;
    }
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
    return focus === FOCUS_READER ? selectedEntry : (activeEntries[cursor] ?? null);
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
    const n   = (mode === MODE_FEEDS ? feeds : activeEntries).length;
    const cur = (mode === MODE_FEEDS ? feedCursor : cursor) + 1;
    if (focus === FOCUS_READER) {
      statusText = `[${selectedIdx + 1}/${activeEntries.length}]  ↑↓ prev/next  space mark read  b back  u read  s star  e save  o open  x original`;
    } else if (mode === MODE_FEEDS) {
      statusText = `${cur}/${n}  enter open  r refresh`;
    } else {
      statusText = `${cur}/${n}  enter open  ↑↓ navigate  space mark read  u toggle  s star  f feeds  r refresh`;
    }
  }

  // ── keyboard ──────────────────────────────────────────────────────

  function openSearch() {
    searchOpen    = true;
    searchResults = [];
    tick().then(() => searchInputEl?.focus());
  }

  async function closeSearch() {
    const saved = selectedEntry;
    searchOpen    = false;
    searchQuery   = '';
    searchResults = null;
    searchFired   = false;
    clearTimeout(searchDebounce);

    if (!saved) return;

    // If the article is read but the "all" filter is off, enable it so it's visible.
    if (saved.status === 'read' && !$showRead) $showRead = true;

    // If it's in a collapsed group, expand that group.
    expandGroupFor(saved);

    await tick();

    const idx = displayEntries.findIndex(e => e.id === saved.id);
    if (idx !== -1) {
      cursor = idx;
      scrollCursorIntoView();
    }
  }

  function onSearchInput() {
    clearTimeout(searchDebounce);
    searchFired = false;
    if (!searchQuery.trim()) { searchResults = []; return; }
    searchDebounce = setTimeout(doSearch, 600);
  }

  async function doSearch() {
    searchFired = true;
    if (!searchQuery.trim()) { searchResults = []; return; }
    try {
      const r = await SearchEntries(searchQuery);
      searchResults = r.entries ?? [];
    } catch (err) { setStatus('Search error: ' + err.message, 3000); }
  }

  function handleKeydown(e) {
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
    switch (e.key) {
      case 'ArrowDown':  e.preventDefault(); moveDown(); break;
      case 'ArrowUp':    e.preventDefault(); moveUp();   break;
      case 'Enter':      e.preventDefault(); selectCurrent(); break;
      case 'Escape': case 'Backspace': case 'b':
        e.preventDefault();
        if (searchOpen) { closeSearch(); return; }
        goBack();
        break;
      case '/': e.preventDefault(); openSearch(); break;
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

  $: {
    const raw = processContent(originalContent ?? selectedEntry?.content ?? '');
    articleHtml = (searchResults !== null && searchQuery.trim() && selectedEntry)
      ? highlightTerms(raw, searchQuery)
      : raw;
  }
  let articleHtml = '';

  $: activeCursor = mode === MODE_FEEDS ? feedCursor : cursor;

  function buildGroupedItems(entries, collapsed = $collapsedFeeds) {
    return _buildGroupedItems(entries, collapsed, timeAgo);
  }

  function buildGroupedCatItems(entries, collapsed = $collapsedFeeds) {
    return _buildGroupedCatItems(entries, collapsed, timeAgo);
  }

  $: displayItems, updateScrollThumb();
  $: displayItems = (now, $collapsedFeeds, $sortOldest, $showRead, $grouped, $groupedCats, searchResults,
    searchResults !== null
      ? ($grouped
          ? buildGroupedItems(searchResults, EMPTY_SET)
          : $groupedCats
          ? buildGroupedCatItems(searchResults, EMPTY_SET)
          : searchResults.map((e, idx) => ({
              type:      'item',
              cursorIdx: idx,
              id:        e.id,
              title:     e.title,
              sub:       (e.starred ? '★  ' : '') + e.feed.title + '  ·  ' + timeAgo(e.published_at),
              unread:    e.status === 'unread',
            })))
      : mode === MODE_FEEDS
      ? feeds.map((f, i) => ({
          type:      'item',
          cursorIdx: i,
          id:        f.feed_id,
          title:     f.feed_title,
          sub:       `${f.unread} unread`,
          unread:    f.unread > 0,
        }))
      : $grouped
        ? buildGroupedItems(displayEntries)
        : $groupedCats
        ? buildGroupedCatItems(displayEntries)
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

  let settingsClearing = false;
  async function clearCache() {
    settingsClearing = true;
    try {
      const result = await ClearCache();
      allEntries = result.entries ?? [];
      feeds      = result.feeds   ?? [];
      entries    = filterByFeed(allEntries, filterFeedID);
      if (cursor >= entries.length) cursor = 0;
      refreshStatus();
      settingsOpen = false;
      showToast('Cache cleared', 2500);
    } catch (e) {
      showToast('Clear cache failed: ' + e, 4000);
    } finally {
      settingsClearing = false;
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="app">

  <div class="body">

  <div class="left-col" class:nav-collapsed={$navCollapsed} style="width: {$navCollapsed ? 'var(--collapsed-w)' : $navWidth + 'px'}">

      <div class="toolbar toolbar-nav" class:nav-collapsed={$navCollapsed}>
        <div class="nav-left">
          <div class="collapse-btn-wrap">
            <button class="nav-arrow-btn nav-collapse-btn"
                    on:click={toggleNav}
                    title={$navCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}>
              <div class="flip-icon" class:flipped={$navCollapsed}>
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
              </div>
            </button>
          </div>
        </div>
        <div class="toolbar-toggles nav-collapsible">
          <button class="pill" class:active={$showRead}   on:click={() => $showRead   = !$showRead}   title="Show or hide read articles">all</button>
          <button class="pill" class:active={$sortOldest} on:click={() => $sortOldest = !$sortOldest} title="Sort oldest first">oldest</button>
          <span class="pill-label">Group:</span>
          <button class="pill" class:active={$groupedCats} on:click={() => { $groupedCats = !$groupedCats; if ($groupedCats) $grouped = false; }} title="Group by category">tags</button>
          <button class="pill" class:active={$grouped}    on:click={() => { $grouped = !$grouped; if ($grouped) $groupedCats = false; }}    title="Group by feed">feeds</button>
        </div>
      </div>

      {#if searchOpen && !$navCollapsed}
        <SearchBar
          bind:query={searchQuery}
          bind:inputEl={searchInputEl}
          inputColor={searchInputColor}
          on:input={onSearchInput}
          on:search={() => { clearTimeout(searchDebounce); doSearch(); }}
          on:close={closeSearch}
        />
      {/if}

      {#if ($grouped || $groupedCats) && !$navCollapsed && searchResults === null}
        <div class="group-actions-bar">
          <button class="pill" style="padding-right:0;gap:3px;display:flex;align-items:center" on:click={() => { $collapsedFeeds = new Set(displayItems.filter(i => i.type === 'header').map(i => i.feedId)); }} title="Collapse all"><span style="position:relative;top:1px;display:flex"><ChevronsDownUp size={12}/></span>collapse</button><span style="color:#414868;font-size:11px;position:relative;top:-1px">/</span><button class="pill" style="padding-left:0;gap:3px;display:flex;align-items:center" on:click={() => { $collapsedFeeds = new Set(); }} title="Expand all"><span style="position:relative;top:1px;display:flex"><ChevronsUpDown size={12}/></span>expand</button>
        </div>
      {/if}

      <div class="nav-pane-wrap nav-collapsible">
      <div class="nav-pane" class:window-focused={windowFocused} bind:this={navPaneEl} on:scroll={updateScrollThumb}>
      {#if loading}
        <div class="nav-empty">Loading…</div>
      {:else if error}
        <div class="nav-empty nav-error">{error}</div>
      {:else if displayItems.length === 0}
        <div class="nav-empty">{searchResults !== null ? (searchFired ? 'No matches' : '') : 'No unread articles'}</div>
      {:else}
        {#each displayItems as item}
          {#if item.type === 'header'}
            <NavFeedHeader {item} on:collapse={e => toggleFeedCollapse(e.detail)} />
          {:else}
            <NavItem
              {item}
              {windowFocused}
              selected={item.cursorIdx === activeCursor}
              open={item.id === selectedEntry?.id && mode === MODE_ENTRIES}
              bind:el={itemEls[item.cursorIdx]}
              on:select={e => mode === MODE_FEEDS ? selectFeed(e.detail) : openArticle(e.detail)}
            />
          {/if}
        {/each}
      {/if}
      </div><!-- /nav-pane -->
      {#if $showScrollbar && needsScroll}
        <div class="custom-scrollbar">
          <div class="custom-scrollbar-thumb"
            role="scrollbar"
            aria-controls="nav-pane"
            aria-valuenow={thumbTop}
            aria-valuemin={0}
            aria-valuemax={100}
            tabindex="0"
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
        <div class="nav-bottom-right">
          <button class="mark-all-btn" on:click={markAllRead}>mark all read</button>
          <span class="toolbar-sep"></span>
          <button class="toolbar-btn" on:click={() => fetchEntries(false, true)} title="Refresh (r)">↺</button>
          <span class="toolbar-sep"></span>
          <button class="nav-arrow-btn" class:active={searchOpen} on:click={openSearch} title="Search (/)" style="position:relative;top:1px">
            <Search size={13}/>
          </button>
          <span class="toolbar-sep"></span>
          <button class="nav-arrow-btn" on:click={openSettings} title="Settings" style="position:relative;top:2px">
            <Settings size={14}/>
          </button>
        </div>
      </div>

      {#if $navCollapsed}
        <div class="collapsed-nav-btns">
          <button class="nav-arrow-btn" on:click={moveUp}   title="Previous (↑)">↑</button>
          <button class="nav-arrow-btn" on:click={moveDown} title="Next (↓)">↓</button>
        </div>
      {/if}

  </div><!-- /left-col -->

    <!-- svelte-ignore a11y_no_noninteractive_element_interactions a11y_no_noninteractive_tabindex -->
    <div class="splitter" role="separator" aria-label="Resize navigation panel" aria-valuenow={$navWidth} aria-valuemin={160} aria-valuemax={600} tabindex="0" class:hidden={$navCollapsed} class:web={import.meta.env.VITE_API !== 'wails'} on:mousedown={startNavResize} on:keydown={e => (e.key === 'ArrowLeft' || e.key === 'ArrowRight') && startNavResize(e)}></div>

    <div class="reader-pane" bind:this={readerEl} bind:clientWidth={readerWidth} bind:clientHeight={readerHeight}>
      {#if selectedEntry}
        <div class="reader-viewport" style="width: {contentWidth}px">
          <div class="reader-content"
               bind:this={contentEl}
               style="--col-h: {Math.max(0, readerHeight - COL_PAD_TOP - COL_PAD_BOT)}px; width: {contentWidth}px; column-width: {colWidth}px; column-gap: {COL_GAP}px; padding: {COL_PAD_TOP}px {COL_PAD}px {COL_PAD_BOT}px; height: 100%; transform: translateX(-{page * pageStride}px)">
            <h1 class="article-title">{selectedEntry.title}</h1>
            <div class="article-meta">{selectedEntry.feed.title}{selectedEntry.feed.category?.title && selectedEntry.feed.category.title !== 'All' ? '  ·  ' + selectedEntry.feed.category.title : ''}  ·  {fullDate(selectedEntry.published_at)}{selectedEntry.fetched_at ? '  ·  Fetched ' + timeAgo(selectedEntry.fetched_at) : ''}</div>
            <ReaderControls
              entry={selectedEntry}
              originalActive={originalContent !== null}
              keptUnread={$keptUnread}
              on:fetchOriginal={fetchOriginal}
              on:decreaseFontSize={decreaseFontSize}
              on:increaseFontSize={increaseFontSize}
              on:mailClick={handleMailClick}
              on:save={saveEntry}
              on:openBrowser={openBrowser}
            />
            <div class="article-body" role="presentation" style="font-size: {$fontSize}px" on:click={handleArticleClick} on:keydown={handleArticleClick} on:error|capture={handleArticleImgError}>
              {@html articleHtml}
            </div>
          </div>
        </div>
        <div class="bottom-pad-mask"></div>
        <Toast msg={toastMsg} visible={toastVisible} />
        <PageNav bind:page totalPages={totalPages} />
      {:else}
        <div class="reader-empty">Select an article to read</div>
      {/if}
    </div>

  </div><!-- /body -->

{#if settingsOpen && settingsCfg}
  <SettingsModal
    cfg={settingsCfg}
    saving={settingsSaving}
    clearing={settingsClearing}
    on:close={() => settingsOpen = false}
    on:save={saveSettings}
    on:clearCache={clearCache}
  />
{/if}

</div><!-- /app -->

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(*:focus) { outline: none; }

  :global(html, body) {
    height: 100%;
    overflow: hidden;
    font-family: 'Lexend Deca', system-ui, sans-serif;
    font-weight: 300;
    -webkit-font-smoothing: antialiased;
    text-rendering: optimizeLegibility;
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
    align-items: stretch;
    justify-content: space-between;
    padding: 6px 14px;
    background: #24283b;
    border-bottom: 1px solid #414868;
    flex-shrink: 0;
  }

  .pill-label {
    font-size: 11px;
    color: #414868;
    padding: 0 2px 0 6px;
    user-select: none;
  }
  .toolbar-toggles {
    display: flex;
    align-items: center;
    gap: 2px;
    position: relative;
    top: 1px;
  }

  .pill {
    padding: 2px 4px;
    border-radius: 4px;
    border: none;
    background: transparent;
    color: #737aa2;
    font-family: inherit;
    font-size: 11px;
    font-weight: 400;
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
    position: relative;
    height: 38px;
  }


  .nav-left { display: flex; gap: 2px; align-items: stretch; }
  .nav-ud-btns { display: flex; gap: 2px; }
  .collapsed-nav-btns {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    display: flex;
    flex-direction: column;
    gap: 4px;
  }


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
  .nav-arrow-btn:active { background: #414868; color: #c0caf5; }
  .collapse-btn-wrap { position: absolute; left: 0; top: 0; bottom: 0; display: flex; }
  .nav-collapse-btn {
    display: flex; align-items: center; justify-content: center;
    padding: 2px 14px; background: #24283b; color: #7aa2f7;
    border-radius: 0;
  }
  .nav-collapse-btn:hover { background: #24283b !important; }
  .flip-icon {
    perspective: 200px;
    position: relative; width: 16px; height: 16px;
    transform-style: preserve-3d;
    transition: transform 280ms cubic-bezier(0.4, 0, 0.2, 1);
    top: 2px;
  }
  .flip-icon.flipped { transform: rotateY(180deg); }
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
    width: 5px;
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
    background: transparent;
    z-index: 10;
    flex-shrink: 0;
  }
  .custom-scrollbar-thumb {
    position: absolute;
    width: 4px;
    background: #414868;
    border-radius: 4px;
    cursor: pointer;
    user-select: none;
    right: 2px;
  }


  .nav-empty {
    padding: 20px 14px;
    font-size: 13px;
    color: #737aa2;
  }
  .nav-error { color: #f7768e; }


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

  .reader-content .article-body :global(mark) {
    background: #fef08a;
    color: #1c1917;
    border-radius: 2px;
    padding: 0 1px;
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

  /* ── article body (global: rendered HTML) ── */
  .article-body {
    overflow-wrap: break-word;
    hyphens: auto;
  }

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
    max-height: var(--col-h);
    height: auto;
    object-fit: contain;
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
    break-inside: avoid;
    max-height: var(--col-h);
    display: block;
    overflow-x: auto;
  }
  .article-body :global(th),
  .article-body :global(td) {
    border: 1px solid #d6c4a0;
    padding: 7px 12px;
    text-align: left;
  }
  .article-body :global(th) { background: #f0e4c8; font-weight: 700; }

  .article-body :global(figure) { margin: 1.2em auto; max-width: 85%; max-height: var(--col-h); }

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


  .toolbar-nav-bottom {
    border-top: 1px solid #414868;
    border-bottom: none;
    justify-content: flex-end;
    align-items: center;
    padding: 6px 4px;
  }

  .nav-bottom-spacer { flex: 1; }
  .group-actions-bar {
    background: #1a1b26;
    border-bottom: 1px solid #414868;
    padding: 2px 4px 2px 8px;
    display: flex;
    align-items: center;
    gap: 2px;
    flex-shrink: 0;
  }
  .nav-bottom-right { display: flex; align-items: center; gap: 0; }


</style>
