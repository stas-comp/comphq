import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

// SPEC gate 1.05 (picker/assignment-list part): renaming shows the new
// name in the picker and the top bar everywhere, not just for the person
// who renamed themselves.
test('renaming a person shows the new name in the picker and top bar', async ({ page, server }) => {
  const oldName = await signInAsNewPerson(page, server.baseURL, '/settings/people');
  await ready(page);

  const newName = uniqueName('Renamed');
  const row = page.locator('.settings-person-row', { has: page.locator(`input[value="${oldName}"]`) });
  await row.locator('input[name="name"]').fill(newName);
  await row.locator('.rename-form button[type="submit"]').click();
  await ready(page);

  // The renaming person's own top bar updates immediately...
  await expect(page.locator('.topbar')).toContainText(`You: ${newName}`);

  // ...and the new name (not the old one) is what the picker offers.
  await page.goto(server.baseURL + '/who');
  await ready(page);
  await expect(page.locator('.person-button', { hasText: newName })).toHaveCount(1);
  await expect(page.locator('.person-button', { hasText: oldName })).toHaveCount(0);
});

// SPEC gate 1.05: a removed person leaves the picker.
test('removing a person leaves the picker', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/settings/people');
  await ready(page);

  // Add a second person to remove, from a separate context so removing
  // them doesn't sign the current page out.
  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  const toRemove = await signInAsNewPerson(otherPage, server.baseURL, '/');
  await otherContext.close();

  await page.goto(server.baseURL + '/settings/people');
  await ready(page);
  const row = page.locator('.settings-person-row', { has: page.locator(`input[value="${toRemove}"]`) });
  await row.locator('.remove-form button[type="submit"]').click();
  await ready(page);

  await page.goto(server.baseURL + '/who');
  await ready(page);
  await expect(page.locator('.person-button', { hasText: toRemove })).toHaveCount(0);
});

test('About shows the version', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/settings/about');
  await ready(page);
  // Not a literal string: the per-worker local binary is always built
  // "0.0.0-test" (e2e/helpers/global-setup.ts), but PLAN.md P1-38's
  // offline-container run points this same test at a real container
  // image carrying its own version (e.g. "0.0.2-container-test"). Gate
  // 1.37 only needs a real version number to be shown.
  await expect(page.locator('#app-version')).toHaveText(/^\d+\.\d+\.\d+/);
});

test('Copy puts the exact command text on the clipboard', async ({ page, server, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await signInAsNewPerson(page, server.baseURL, '/settings/setup');
  await ready(page);

  const expected = await page.locator('#chrome-command').textContent();
  await page.click('.copy-button[data-target="chrome-command"]');

  // copy.js (SPEC B6) tries the Clipboard API first, falling back to
  // selecting the text when it's unavailable — which is what a real
  // owner actually sees, since Comp HQ is reached over plain HTTP at a
  // NAS IP address, not a secure context. Chromium always treats a
  // loopback address as secure regardless of scheme, which is why every
  // worker-spawned local server takes the "Copied." path; only PLAN.md
  // P1-38's offline-container run (reached via a Compose service
  // hostname, not an IP) exercises the fallback for real. Both paths
  // must land on the exact same text.
  const statusLocator = page.locator('#copy-status');
  await expect(statusLocator).toHaveText(/^(Copied\.|Selected — press Ctrl\+C to copy\.)$/);
  const status = await statusLocator.textContent();

  if (status === 'Copied.') {
    const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
    expect(clipboardText).toBe(expected);
  } else {
    const selectedText = await page.evaluate(() => window.getSelection()?.toString());
    expect(selectedText).toBe(expected);
  }
  expect(expected).toContain('chrome.exe');
});
