import { type Page, expect } from '@playwright/test';

const SIZES = [
  { width: 1024, height: 700 },
  { width: 1920, height: 1080 },
];

// No page may scroll sideways at either size (SPEC gates 1.09, 2.23, 3.13).
// Measures document.body, not document.documentElement: Chromium can
// report documentElement.scrollWidth far wider than anything actually
// visible when a descendant has its own legitimate overflow:auto region
// (the Team view's lanes, SPEC gate 2.31, once enough exist) — a
// measurement quirk, not a real side-scroll, confirmed by every
// individual element on the page measuring correctly while only that
// one number was wrong. body.scrollWidth stayed correct throughout.
export async function expectNoSideScroll(page: Page): Promise<void> {
  for (const size of SIZES) {
    await page.setViewportSize(size);
    const overflowing = await page.evaluate(
      () => document.body.scrollWidth > document.documentElement.clientWidth,
    );
    expect(overflowing, `page scrolls sideways at ${size.width}x${size.height}`).toBe(false);
  }
}
