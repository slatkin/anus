// Pure paging calculation functions — import these into App.svelte and test with Vitest.

export const COL_MIN     = 420; // min column width — 2 cols appear only when each is at least this wide
export const COL_MAX     = 520; // max column width — readability cap (~65ch at 16px)
export const COL_PAD     = 48;  // horizontal padding applied to .reader-content via inline style
export const COL_GAP     = 48;  // column-gap applied to .reader-content via inline style (3em @ 16px)
export const COL_PAD_TOP = 36;  // vertical padding top
export const COL_PAD_BOT = 64;  // vertical padding bottom — matches .bottom-pad-mask height

// (x - 2*COL_PAD + COL_GAP) is the "layout span" shared by the column, stride, and page formulae.
function layoutSpan(x) {
  return x - 2 * COL_PAD + COL_GAP;
}

/**
 * How many columns fit in readerWidth, capped at 3.
 * Uses COL_MIN as the threshold: a new column appears only when there's room for another COL_MIN-wide column.
 */
export function calcCols(readerWidth) {
  if (readerWidth <= 0) return 1;
  return Math.min(3, Math.max(1, Math.floor(layoutSpan(readerWidth) / (COL_MIN + COL_GAP))));
}

/**
 * Fill available space equally, never exceeding COL_MAX.
 */
export function calcColWidth(readerWidth, cols) {
  if (readerWidth <= 2 * COL_PAD) return COL_MIN;
  return Math.min(COL_MAX, Math.round((readerWidth - 2 * COL_PAD - (cols - 1) * COL_GAP) / cols));
}

/**
 * Exact pixel width for the .reader-content div.
 * Less than readerWidth when colWidth is capped at COL_MAX (wide screen centering case).
 */
export function calcContentWidth(cols, colWidth) {
  return cols * colWidth + (cols - 1) * COL_GAP + 2 * COL_PAD;
}

/**
 * Horizontal distance (px) that a translateX must advance per page turn.
 * Derivation: stride = cols*(colWidth+GAP) = clientWidth - 2*PAD + GAP.
 * Pass contentEl.clientWidth as clientWidth.
 */
export function calcPageStride(clientWidth) {
  return layoutSpan(clientWidth);
}

/**
 * Number of pages in the overflow multi-column layout.
 * Pass contentEl.scrollWidth and the already-computed pageStride.
 */
export function calcTotalPages(scrollWidth, pageStride) {
  if (pageStride <= 0) return 1;
  const raw = layoutSpan(scrollWidth) / pageStride;
  // ceil so a partial last page (e.g. a pushed figure) is always reachable.
  // Subtract a small tolerance so sub-pixel rounding doesn't create phantom pages.
  return Math.max(1, Math.ceil(raw - 0.05));
}
