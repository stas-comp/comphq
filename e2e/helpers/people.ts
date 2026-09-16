import type { Page } from '@playwright/test';
import { uniqueName } from './unique-name';

/**
 * Creates a fresh, uniquely named person via "My name isn't here" and
 * lands back on `path` (SPEC gate 1.04). Most tests that need any
 * identified person at all should use this rather than seeding one
 * directly, so each test's data — and its person — never collides with
 * another's (SPEC §2.8: "each test creates its own uniquely named data").
 */
export async function signInAsNewPerson(page: Page, baseURL: string, path = '/'): Promise<string> {
  const name = uniqueName('Test');
  await page.goto(baseURL + '/who?next=' + encodeURIComponent(path));
  await page.click('#add-name-link');
  await page.fill('#add-name-input', name);
  await page.click('#add-name-form button[type="submit"]');
  return name;
}
