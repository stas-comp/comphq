import { expect, test } from './fixtures';
import { expectMatchesMockup } from './mockup';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// Icons and the icon-only button rule (PLAN P4-04, gates 4.11, 4.12),
// checked on the test-mode parts page and on the real frame.

test('gate 4.12: every icon file is served by the app, and none comes from the internet', async ({ page, server }) => {
  const external: string[] = [];
  const own = new URL(server.baseURL).origin;
  page.on('request', (req) => {
    if (/^https?:/.test(req.url()) && new URL(req.url()).origin !== own) external.push(req.url());
  });
  await signInAsNewPerson(page, server.baseURL, '/__test/parts');
  await ready(page);
  const names = await page.locator('#parts-icons .icon').evaluateAll((els) => els.map((el) => (el as HTMLElement).dataset.icon!));
  expect(names.length).toBe(15);
  for (const name of names) {
    const res = await page.request.get(`${server.baseURL}/static/theme/icons/${name}.svg`);
    expect(res.status(), name).toBe(200);
    expect(res.headers()['content-type'], name).toContain('image/svg+xml');
  }
  expect(external).toEqual([]);
});

// The mask has to actually paint: an icon with its drawing switched off is
// a plain filled square, so a working icon must look different from that,
// and every icon different from every other.
test('gate 4.12: each icon paints its own drawing', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/__test/parts');
  await ready(page);
  const icons = page.locator('#parts-icons .icon');
  const count = await icons.count();
  const shots: Buffer[] = [];
  for (let i = 0; i < count; i++) shots.push(await icons.nth(i).screenshot());
  const distinct = new Set(shots.map((b) => b.toString('base64')));
  expect(distinct.size, 'every icon looks different from every other').toBe(count);

  const first = icons.first();
  const drawn = await first.screenshot();
  await first.evaluate((el) => (el as HTMLElement).style.setProperty('mask', 'none'));
  const square = await first.screenshot();
  expect(drawn.equals(square), 'a drawn icon is not just a filled square').toBe(false);
});

test('gate 4.12: an icon takes the colour of the button that holds it, and is drawn at the mockup size', async ({
  page,
  server,
  mockup,
}) => {
  await signInAsNewPerson(page, server.baseURL, '/__test/parts');
  await ready(page);
  const inPrimary = page.locator('#parts-icon-in-primary .icon');
  const buttonColour = await page.locator('#parts-icon-in-primary').evaluate((el) => getComputedStyle(el).color);
  await expect(inPrimary).toHaveCSS('background-color', buttonColour);
  expect(buttonColour).toBe('rgb(20, 27, 45)'); // ink on the accent, never white (gate 4.03)

  // The sidebar icon is 20px, as the mockup's `.nav svg`, and is ink on the current section.
  await page.goto(server.baseURL + '/settings/about');
  await ready(page);
  await expectMatchesMockup(mockup, page, { mockup: '.nav svg', screen: 'briefing', app: '.sidebar nav a .icon' }, ['width', 'height']);
  const navColour = await page.locator('.sidebar nav a[aria-current="page"]').evaluate((el) => getComputedStyle(el).color);
  await expect(page.locator('.sidebar nav a[aria-current="page"] .icon')).toHaveCSS('background-color', navColour);
});

test('gate 4.11: there are four icon-only buttons, each with the same title and spoken name naming the task', async ({
  page,
  server,
}) => {
  await signInAsNewPerson(page, server.baseURL, '/__test/parts');
  await ready(page);
  const buttons = page.locator('#parts-icon-buttons .icon-btn');
  await expect(buttons).toHaveCount(5); // the four, plus a disabled top card's "move up"
  const labels = await buttons.evaluateAll((els) =>
    els.map((el) => ({ title: el.getAttribute('title'), aria: el.getAttribute('aria-label'), text: el.textContent!.trim() })),
  );
  expect(labels.map((l) => l.title)).toEqual([
    'Move Print exam papers up',
    'Move Print exam papers down',
    'Remove Print exam papers',
    'Give back Print exam papers',
    'Move Top card up',
  ]);
  for (const l of labels) {
    expect(l.aria).toBe(l.title);
    expect(l.text).toBe('');
  }
  await expect(buttons.last()).toBeDisabled();
  // Named for a screen reader as well: the accessible name is the label.
  await expect(page.getByRole('button', { name: 'Remove Print exam papers' })).toHaveCount(1);
});

test('gate 4.11: an icon-only button is a 24px bordered box, as the mockup draws the card movers', async ({
  page,
  server,
  mockup,
}) => {
  await signInAsNewPerson(page, server.baseURL, '/__test/parts');
  await ready(page);
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.card .mv i', screen: 'tasks', app: '#parts-icon-buttons .icon-btn' },
    ['width', 'height', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius', 'color', 'display'],
  );
});

test('standard page checks for the parts page with icons', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/__test/parts');
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);
});
