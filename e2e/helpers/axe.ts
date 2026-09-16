import { type Page, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

// Fails on serious or critical violations only (SPEC B7 layer 6).
export async function axeCheck(page: Page): Promise<void> {
  const results = await new AxeBuilder({ page }).analyze();
  const serious = results.violations.filter(
    (v) => v.impact === 'serious' || v.impact === 'critical',
  );
  expect(serious, JSON.stringify(serious, null, 2)).toEqual([]);
}
