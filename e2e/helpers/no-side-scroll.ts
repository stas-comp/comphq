import { type Page, expect } from '@playwright/test';

const SIZES = [
  { width: 1024, height: 700 },
  { width: 1920, height: 1080 },
];

// No page may scroll sideways at either size (SPEC gates 1.09, 2.23, 3.13,
// and 7.50). Measures the whole page, documentElement as well as body
// (D-97, which supersedes D-48): documentElement.scrollWidth is what the
// browser actually lets a person scroll. D-48 once saw it report ~29,800
// px on Team and switched to body.scrollWidth, which cannot see that kind
// of overflow, and so hid a real sideways scroll (the white areas).
export async function expectNoSideScroll(page: Page): Promise<void> {
  for (const size of SIZES) {
    await page.setViewportSize(size);
    const m = await page.evaluate(() => ({
      client: document.documentElement.clientWidth,
      root: document.documentElement.scrollWidth,
      body: document.body.scrollWidth,
    }));
    expect(m.root, `the page scrolls sideways at ${size.width}x${size.height} (documentElement)`).toBeLessThanOrEqual(m.client);
    expect(m.body, `the page scrolls sideways at ${size.width}x${size.height} (body)`).toBeLessThanOrEqual(m.client);
  }
}
