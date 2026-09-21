import fs from 'node:fs';
import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// Set up this computer, rewritten (PLAN P5-07, SPEC B11.4, gate 5.25): the
// ways to try, in order, each headed by the browser it is for, with the
// command shortcut kept and an icon to download.

test.beforeEach(async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/settings/setup');
  await ready(page);
});

test('gate 5.25: the page leads with installing as an app, then the command, then the taskbar, then the icon', async ({ page }) => {
  const headings = await page.locator('.setup-way h2').allTextContents();
  expect(headings).toEqual([
    '1. Install it as an app — Edge or Chrome',
    '2. Make a shortcut with a command — Chrome, Edge or Brave',
    '3. Put it on the taskbar',
    '4. If Windows shows a blank icon',
  ]);

  // Each route says which browser it is for.
  const install = page.locator('#way-install');
  await expect(install.getByRole('heading', { name: 'In Microsoft Edge' })).toBeVisible();
  await expect(install.getByRole('heading', { name: 'In Google Chrome' })).toBeVisible();
  await expect(install).toContainText('Apps');
  await expect(install).toContainText('Install this site as an app');
  await expect(install).toContainText('Save and share');
  await expect(install).toContainText('Install page as app');

  const shortcut = page.locator('#way-shortcut');
  for (const browser of ['Google Chrome', 'Microsoft Edge', 'Brave']) {
    await expect(shortcut.getByRole('heading', { name: browser, exact: true })).toBeVisible();
  }
});

test('gate 5.25: no route assumes the one before it worked — each is a section of its own that can be followed alone', async ({ page }) => {
  for (const id of ['way-install', 'way-shortcut', 'way-taskbar', 'way-icon']) {
    const section = page.locator(`#${id}`);
    await expect(section).toBeVisible();
    // Its own steps, in a list of its own.
    expect(await section.locator('ol li').count(), id).toBeGreaterThanOrEqual(2);
  }
  // Way 1 says what to do if it doesn't work, and points at the way that doesn't need it.
  await expect(page.locator('#way-install')).toContainText('use way 2');
  // Way 2 gives everything needed for a shortcut, by itself.
  await expect(page.locator('#way-shortcut')).toContainText('New');
  await expect(page.locator('#way-shortcut')).toContainText('Shortcut');
});

test('gate 5.25: each command is built from this server’s own address, and each has its own Copy button', async ({ page, server }) => {
  const host = new URL(server.baseURL).host;
  const commands: [string, RegExp][] = [
    ['chrome-command', /chrome\.exe" --app=http:\/\//],
    ['edge-command', /msedge\.exe" --app=http:\/\//],
    ['brave-command', /brave\.exe" --app=http:\/\//],
  ];
  for (const [id, pattern] of commands) {
    const text = (await page.locator(`#${id}`).textContent()) ?? '';
    expect(text, id).toMatch(pattern);
    expect(text.endsWith(`--app=http://${host}/`), `${id} ends with this server's address`).toBe(true);
    await expect(page.locator(`.copy-button[data-target="${id}"]`)).toBeVisible();
  }
});

test('Copy puts each of the three commands on the clipboard, exactly', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  for (const id of ['chrome-command', 'edge-command', 'brave-command']) {
    const expected = await page.locator(`#${id}`).textContent();
    await page.click(`.copy-button[data-target="${id}"]`);
    await expect(page.locator('#copy-status')).toHaveText(/^(Copied\.|Selected — press Ctrl\+C to copy\.)$/);
    const status = await page.locator('#copy-status').textContent();
    if (status === 'Copied.') {
      expect(await page.evaluate(() => navigator.clipboard.readText()), id).toBe(expected);
    } else {
      expect(await page.evaluate(() => window.getSelection()?.toString()), id).toBe(expected);
    }
    // Clear the status so the next button's own message is what is checked.
    await page.evaluate(() => (document.getElementById('copy-status')!.textContent = ''));
  }
});

test('gate 5.25: Download icon saves a real .ico called Comp HQ.ico', async ({ page }) => {
  const [download] = await Promise.all([page.waitForEvent('download'), page.getByRole('link', { name: 'Download icon' }).click()]);
  expect(download.suggestedFilename()).toBe('Comp HQ.ico');
  const file = await download.path();
  const bytes = fs.readFileSync(file);
  expect([...bytes.subarray(0, 4)]).toEqual([0, 0, 1, 0]); // an icon file
  expect(bytes.readUInt16LE(4), 'a multi-size icon').toBeGreaterThanOrEqual(4);
  // The three steps that set it by hand.
  const way = page.locator('#way-icon');
  for (const words of ['Properties', 'Change Icon', 'Browse']) await expect(way).toContainText(words);
});

test('the Setup page passes the accessibility check and has no sideways scroll', async ({ page }) => {
  await axeCheck(page);
  await expectNoSideScroll(page);
});
