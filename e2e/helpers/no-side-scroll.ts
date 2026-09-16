import { type Page, expect } from '@playwright/test';

const SIZES = [
  { width: 1024, height: 700 },
  { width: 1920, height: 1080 },
];

// No page may scroll sideways at either size (SPEC gates 1.09, 2.23, 3.13).
export async function expectNoSideScroll(page: Page): Promise<void> {
  for (const size of SIZES) {
    await page.setViewportSize(size);
    const overflowing = await page.evaluate(
      () => document.documentElement.scrollWidth > document.documentElement.clientWidth,
    );
    expect(overflowing, `page scrolls sideways at ${size.width}x${size.height}`).toBe(false);
  }
}
