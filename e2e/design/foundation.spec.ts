import { expect, test } from './fixtures';
import { expectMatchesMockup } from './mockup';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// Design foundations (PLAN P4-02, gates 4.01, 4.02, 4.03): the app's real
// pages agree with docs/design/mockup.html, property by property.

test('gate 4.01: the three font files are served by the app and load', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/settings/about');
  await ready(page);
  const loaded = await page.evaluate(async () => {
    const wanted = ['900 1em "Big Shoulders Display"', '400 1em "Atkinson Hyperlegible Next"', '500 1em "IBM Plex Mono"'];
    const out: Record<string, boolean> = {};
    for (const f of wanted) {
      await document.fonts.load(f);
      out[f] = document.fonts.check(f);
    }
    return out;
  });
  expect(loaded).toEqual({
    '900 1em "Big Shoulders Display"': true,
    '400 1em "Atkinson Hyperlegible Next"': true,
    '500 1em "IBM Plex Mono"': true,
  });
});

test('gate 4.01: no page fetches a font, icon or anything else from the internet', async ({ page, server }) => {
  const external: string[] = [];
  const own = new URL(server.baseURL).origin;
  page.on('request', (req) => {
    const url = req.url();
    if (/^https?:/.test(url) && new URL(url).origin !== own) external.push(url);
  });
  await signInAsNewPerson(page, server.baseURL, '/');
  for (const path of ['/', '/kb', '/tasks', '/tasks/board', '/calendar', '/settings/about']) {
    await page.goto(server.baseURL + path);
    await ready(page);
    await page.evaluate(() => document.fonts.ready);
  }
  expect(external).toEqual([]);
});

test('gate 4.01: a big heading is set as the mockup sets its titles', async ({ page, server, mockup }) => {
  await signInAsNewPerson(page, server.baseURL, '/settings/about');
  await ready(page);
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.article h1', screen: 'kb', app: 'main h1' },
    ['font-family', 'font-weight', 'font-size', 'line-height', 'letter-spacing', 'text-transform', 'color'],
  );
});

test('gate 4.01: a section heading is set as the mockup sets its panel heads', async ({ page, server, mockup }) => {
  await signInAsNewPerson(page, server.baseURL, '/');
  await ready(page);
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.panel-head h2', screen: 'briefing', app: '.briefing-section h2' },
    ['font-family', 'font-weight', 'font-size', 'line-height', 'text-transform', 'color'],
  );
});

test('gate 4.01: body text is the readable face at the mockup size, on paper', async ({ page, server, mockup }) => {
  await signInAsNewPerson(page, server.baseURL, '/settings/about');
  await ready(page);
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.window', screen: 'tasks', app: 'body' },
    ['font-family', 'font-size', 'font-weight', 'color', 'background-color'],
  );
});

test('gate 4.02: a version number is set in IBM Plex Mono with figures of equal width', async ({ page, server, mockup }) => {
  await signInAsNewPerson(page, server.baseURL, '/settings/about');
  await ready(page);
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.date', screen: 'briefing', app: '#app-version' },
    ['font-family', 'font-weight', 'font-variant-numeric'],
  );
  const family = await page.locator('#app-version').evaluate((el) => getComputedStyle(el).fontFamily);
  expect(family.toLowerCase()).toContain('ibm plex mono');
});

test('gate 4.03: the current section is an accent block with ink text, never white', async ({ page, server, mockup }) => {
  await signInAsNewPerson(page, server.baseURL, '/settings/about');
  await ready(page);
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.nav[aria-current="page"]', screen: 'briefing', app: '.sidebar nav a[aria-current="page"]' },
    ['background-color', 'color'],
  );
  const bg = await page
    .locator('.sidebar nav a[aria-current="page"]')
    .evaluate((el) => getComputedStyle(el).backgroundColor);
  expect(bg).toBe('rgb(255, 107, 26)'); // #FF6B1A
});
