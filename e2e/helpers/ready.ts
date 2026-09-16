import type { Page } from '@playwright/test';

// Every page sets data-ready on <body> once rendered and its scripts have
// initialised (SPEC B2). Tests wait on this, never on a fixed sleep.
export async function ready(page: Page): Promise<void> {
  await page.waitForSelector('body[data-ready]', { state: 'attached' });
}
