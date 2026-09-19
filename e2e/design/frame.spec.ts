import { expect, test } from './fixtures';
import { diffStyles, expectMatchesMockup, readStyles } from './mockup';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// The frame (PLAN P4-05, gates 4.13, 4.14): the sidebar and the top bar,
// seen on every screen, agree with docs/design/mockup.html.

const TEXT = ['font-family', 'font-weight', 'font-size', 'letter-spacing', 'text-transform', 'color', 'line-height'];

test.beforeEach(async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  await ready(page);
});

test('gate 4.13: the sidebar is the mockup sidebar', async ({ page, mockup }) => {
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.side', screen: 'tasks', app: '.sidebar' },
    ['background-color', 'color', 'display', 'flex-direction', 'padding', 'row-gap', 'width'],
  );
});

test('gate 4.13: the wordmark is a small wide-spaced COMP above a large accent HQ', async ({ page, mockup }) => {
  await expectMatchesMockup(mockup, page, { mockup: '.side .mark', screen: 'tasks', app: '.wordmark' }, ['padding', 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.side .mark small', screen: 'tasks', app: '.wordmark small' }, [...TEXT, 'display']);
  await expectMatchesMockup(mockup, page, { mockup: '.side .mark strong', screen: 'tasks', app: '.wordmark strong' }, TEXT);
  // COMP sits above HQ, and HQ is the larger of the two.
  const small = await page.locator('.wordmark small').boundingBox();
  const big = await page.locator('.wordmark strong').boundingBox();
  expect(small!.y + small!.height).toBeLessThanOrEqual(big!.y + big!.height * 0.5);
  expect(big!.height).toBeGreaterThan(small!.height);
  await expect(page.locator('.wordmark')).toHaveText('Comp HQ');
});

test('gate 4.13: a section link is drawn as the mockup draws it, and lights up when pointed at', async ({ page, mockup }) => {
  const pair = { mockup: '.nav:not([aria-current])', screen: 'tasks', app: '.sidebar nav a:not([aria-current])' };
  const props = ['font-size', 'font-weight', 'color', 'padding', 'column-gap', 'border-radius', 'background-color', 'display', 'width'];
  await expectMatchesMockup(mockup, page, pair, props);

  await mockup.page.hover(pair.mockup);
  await page.locator(pair.app).first().hover();
  const hovered = diffStyles(await readStyles(mockup.page, pair.mockup, ['background-color', 'color']), await readStyles(page, pair.app, ['background-color', 'color']));
  expect(hovered, 'hover colours').toEqual([]);
});

test('gate 4.13: the current section is a solid accent block with ink text, and Settings sits at the foot', async ({ page, mockup }) => {
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.nav[aria-current="page"]', screen: 'tasks', app: '.sidebar nav a[aria-current="page"]' },
    ['background-color', 'color', 'font-size', 'font-weight', 'padding', 'border-radius'],
  );
  await expect(page.locator('.sidebar nav a[aria-current="page"]')).toHaveText('Tasks');

  const settings = await page.locator('.sidebar nav a', { hasText: 'Settings' }).boundingBox();
  const calendar = await page.locator('.sidebar nav a', { hasText: 'Calendar' }).boundingBox();
  const side = await page.locator('.sidebar').boundingBox();
  expect(settings!.y).toBeGreaterThan(calendar!.y + calendar!.height + 40); // a gap, not the next row
  expect(side!.y + side!.height - (settings!.y + settings!.height)).toBeLessThan(90); // near the bottom
});

test('gate 4.13: the version number is set small in IBM Plex Mono at the foot of the sidebar', async ({ page, mockup }) => {
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.side .ver', screen: 'tasks', app: '.sidebar .ver' },
    ['font-family', 'font-weight', 'font-size', 'color', 'padding'],
  );
  await expect(page.locator('.sidebar .ver')).toContainText(/^Version \S+/);
});

test('gate 4.14: the top bar is the mockup top bar', async ({ page, mockup }) => {
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.top', screen: 'tasks', app: '.topbar' },
    ['display', 'align-items', 'column-gap', 'padding', 'background-color', 'border-bottom-width', 'border-bottom-style', 'border-bottom-color'],
  );
});

test('gate 4.14: the search box is the mockup search box, with a "/" hint, and rings when it has the keyboard', async ({ page, mockup }) => {
  const box = ['display', 'align-items', 'column-gap', 'width', 'background-color', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius', 'padding', 'font-size', 'color'];
  await expectMatchesMockup(mockup, page, { mockup: '.search', screen: 'tasks', app: '.topbar .search' }, box);
  await expectMatchesMockup(mockup, page, { mockup: '.search svg', screen: 'tasks', app: '.topbar .search .icon' }, ['width', 'height']);
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.search kbd', screen: 'tasks', app: '.topbar .search kbd' },
    ['font-family', 'font-weight', 'font-size', 'color', 'padding', 'border-top-width', 'border-top-color', 'border-radius'],
  );
  await expect(page.locator('.topbar .search kbd')).toHaveText('/');
  await expect(page.locator('#search-box')).toHaveAttribute('placeholder', 'Search articles');

  // Focused: the mockup's `.search.active` — a navy outline and the soft accent ring.
  await page.locator('#search-box').focus();
  await mockup.show('kb'); // the mockup shows the search box active on its Knowledge Base screen
  const active = ['border-top-color', 'box-shadow', 'color'];
  const diffs = diffStyles(await readStyles(mockup.page, '.search.active', active), await readStyles(page, '.topbar .search', active));
  expect(diffs, 'focused search box').toEqual([]);
});

test('gate 4.14: pressing / anywhere puts the keyboard in the search box, but not while typing in a field', async ({ page, server }) => {
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await page.locator('.sidebar .ver').click(); // a spot that is not a field
  await page.keyboard.press('/');
  await expect(page.locator('#search-box')).toBeFocused();
  await expect(page.locator('#search-box')).toHaveValue(''); // the key opens the box; it isn't typed into it

  await page.locator('#new-task-title').focus();
  await page.keyboard.type('a/b');
  await expect(page.locator('#new-task-title')).toHaveValue('a/b');
  await expect(page.locator('#new-task-title')).toBeFocused();
});

test('gate 4.14: "You: name", their circle and Change sit on the right, drawn as the mockup draws them', async ({ page, mockup }) => {
  await expectMatchesMockup(mockup, page, { mockup: '.you', screen: 'tasks', app: '.you' }, ['display', 'align-items', 'column-gap', 'font-size']);
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.you a', screen: 'tasks', app: '.you a' },
    ['color', 'font-weight', 'text-decoration-line', 'text-decoration-color', 'text-decoration-thickness', 'text-underline-offset'],
  );
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.you .av.me', screen: 'tasks', app: '.you .av' },
    ['width', 'height', 'border-radius', 'background-color', 'color', 'font-size', 'font-weight'],
  );

  const you = await page.locator('.you').boundingBox();
  const top = await page.locator('.topbar').boundingBox();
  expect(you!.x + you!.width).toBeGreaterThan(top!.x + top!.width - 40); // right-hand end
  const name = (await page.locator('.you strong').textContent())!;
  await expect(page.locator('.you .av')).toHaveText(name.trim()[0].toUpperCase());
  await expect(page.locator('.topbar')).toContainText(`You: ${name}`);
  await expect(page.locator('.you a')).toHaveText('Change');
});

test('the frame is the same on every screen', async ({ page, server }) => {
  for (const path of ['/', '/kb', '/tasks', '/tasks/board', '/tasks/team', '/calendar', '/settings/about', '/does-not-exist']) {
    await page.goto(server.baseURL + path);
    await ready(page);
    await expect(page.locator('.sidebar .wordmark strong'), path).toHaveText('HQ');
    await expect(page.locator('.sidebar'), path).toHaveCSS('background-color', 'rgb(20, 27, 45)');
    await expect(page.locator('.topbar .search kbd'), path).toHaveText('/');
    await expect(page.locator('.you .av'), path).toBeVisible();
  }
});
