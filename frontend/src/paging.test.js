import { describe, it, expect } from 'vitest';
import {
  COL_MIN, COL_MAX, COL_PAD, COL_GAP,
  calcCols, calcColWidth, calcContentWidth, calcPageStride, calcTotalPages,
} from './paging.js';

// Thresholds derived from constants so tests survive constant changes.
const threshold2 = 2 * COL_PAD + 2 * COL_MIN + COL_GAP; // min width for 2 cols
const threshold3 = 2 * COL_PAD + 3 * COL_MIN + 2 * COL_GAP; // min width for 3 cols

// Simulate browser scrollWidth for a multicolumn element with N display-pages of content.
// Browsers include the leading COL_PAD in scrollWidth but omit the trailing one when
// content overflows horizontally. For exactly 1 page (no overflow) scrollWidth = contentWidth.
function browserScrollWidth(cols, colWidth, totalPages) {
  if (totalPages <= 1) return calcContentWidth(cols, colWidth);
  const stride = calcPageStride(cols, colWidth);
  return COL_PAD + totalPages * stride - COL_GAP;
}

describe('calcCols', () => {
  it('returns 1 for zero or negative width', () => {
    expect(calcCols(0)).toBe(1);
    expect(calcCols(-100)).toBe(1);
  });

  it('returns 1 when narrower than the 2-column threshold', () => {
    expect(calcCols(threshold2 - 1)).toBe(1);
  });

  it('returns 2 at the 2-column threshold', () => {
    expect(calcCols(threshold2)).toBe(2);
    expect(calcCols(threshold2 + 100)).toBe(2);
  });

  it('returns 3 at the 3-column threshold', () => {
    expect(calcCols(threshold3)).toBe(3);
    expect(calcCols(threshold3 + 200)).toBe(3);
  });

  it('caps at 3 even on very wide screens', () => {
    expect(calcCols(4000)).toBe(3);
  });
});

describe('calcColWidth', () => {
  it('returns COL_MIN when readerWidth is too narrow to hold padding', () => {
    expect(calcColWidth(0, 1)).toBe(COL_MIN);
    expect(calcColWidth(2 * COL_PAD, 1)).toBe(COL_MIN); // exactly at guard boundary
    expect(calcColWidth(2 * COL_PAD + 1, 1)).toBeGreaterThan(0); // just past boundary → positive
  });

  it('single column fills available space up to COL_MAX', () => {
    const w = calcColWidth(threshold2 - 1, 1); // just below 2-col threshold → 1 col
    expect(w).toBeLessThanOrEqual(COL_MAX);
    expect(w).toBeGreaterThanOrEqual(COL_MIN);
  });

  it('caps at COL_MAX on wide screens', () => {
    // Very wide: each column would exceed COL_MAX, gets capped.
    expect(calcColWidth(4000, 1)).toBe(COL_MAX);
    expect(calcColWidth(4000, 2)).toBe(COL_MAX);
  });

  it('distributes space evenly across columns', () => {
    const cols = 2;
    const w = calcColWidth(threshold2, cols);
    // Should split the available column area equally.
    const available = threshold2 - 2 * COL_PAD - (cols - 1) * COL_GAP;
    expect(w).toBe(Math.min(COL_MAX, Math.round(available / cols)));
  });
});

describe('calcContentWidth', () => {
  it('single column: colWidth plus both paddings', () => {
    expect(calcContentWidth(1, 400)).toBe(400 + 2 * COL_PAD);
  });

  it('two columns: two colWidths plus one gap plus both paddings', () => {
    expect(calcContentWidth(2, 300)).toBe(2 * 300 + COL_GAP + 2 * COL_PAD);
  });

  it('three columns: three colWidths plus two gaps plus both paddings', () => {
    expect(calcContentWidth(3, 250)).toBe(3 * 250 + 2 * COL_GAP + 2 * COL_PAD);
  });
});

describe('calcPageStride', () => {
  it('equals cols*(colWidth+GAP)', () => {
    expect(calcPageStride(1, COL_MIN)).toBe(1 * (COL_MIN + COL_GAP));
    expect(calcPageStride(2, COL_MIN)).toBe(2 * (COL_MIN + COL_GAP));
    expect(calcPageStride(3, 400)).toBe(3 * (400 + COL_GAP));
  });

  it('matches the layout formula for multi-column', () => {
    const cols = 2;
    const colWidth = calcColWidth(threshold2, cols);
    const stride = calcPageStride(cols, colWidth);
    expect(stride).toBe(cols * (colWidth + COL_GAP));
  });
});

describe('calcTotalPages', () => {
  it('returns 1 when content fits exactly in one page', () => {
    const cols = 1;
    const colWidth = COL_MIN;
    const stride = calcPageStride(cols, colWidth);
    const scrollWidth = browserScrollWidth(cols, colWidth, 1);
    expect(calcTotalPages(scrollWidth, stride)).toBe(1);
  });

  it('returns 1 for zero or negative pageStride', () => {
    expect(calcTotalPages(1000, 0)).toBe(1);
    expect(calcTotalPages(1000, -10)).toBe(1);
  });

  it('returns 2 when content spans exactly two pages', () => {
    const cols = 1;
    const colWidth = COL_MIN;
    const stride = calcPageStride(cols, colWidth);
    const scrollWidth = browserScrollWidth(cols, colWidth, 2);
    expect(calcTotalPages(scrollWidth, stride)).toBe(2);
  });

  it('a sliver of content on page N+1 still produces N+1 pages', () => {
    const cols = 1;
    const colWidth = COL_MIN;
    const stride = calcPageStride(cols, colWidth);
    // 1 full page + 46% of a second column
    const scrollWidth = browserScrollWidth(cols, colWidth, 1) + Math.round(stride * 0.46);
    expect(calcTotalPages(scrollWidth, stride)).toBe(2);
  });

  it('sub-pixel rounding does not create phantom extra pages', () => {
    const cols = 1;
    const colWidth = COL_MIN;
    const stride = calcPageStride(cols, colWidth);
    // Exactly 2 pages + 2px FP noise — must NOT become 3.
    const scrollWidth = browserScrollWidth(cols, colWidth, 2) + 2;
    expect(calcTotalPages(scrollWidth, stride)).toBe(2);
  });

  it('works correctly for multi-column (2 cols) layout across 3 pages', () => {
    const cols = 2;
    const colWidth = calcColWidth(threshold2, cols);
    const stride = calcPageStride(cols, colWidth);
    const scrollWidth = browserScrollWidth(cols, colWidth, 3);
    expect(calcTotalPages(scrollWidth, stride)).toBe(3);
  });
});
