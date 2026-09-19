import { expect, test } from './fixtures';
import { diffStyles, expectMatchesMockup, readStyles } from './mockup';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

// The Saturday Briefing and the name picker (PLAN P4-13, gates 4.44-4.47),
// compared with docs/design/mockup.html.

type Page = import('@playwright/test').Page;

const TEXT = ['font-family', 'font-weight', 'font-size', 'letter-spacing', 'text-transform', 'color'];
const BOX = ['background-color', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius', 'padding'];

const typed = (d: Date) => `${String(d.getDate()).padStart(2, '0')}/${String(d.getMonth() + 1).padStart(2, '0')}/${d.getFullYear()}`;

// A job that is overdue, assigned to me, with notes; and an event tomorrow with a note.
async function briefingWithItems(page: Page, baseURL: string) {
  const me = await signInAsNewPerson(page, baseURL, '/tasks/board');
  await ready(page);
  const task = uniqueName('Due job');
  await openNewTask(page);
  await page.fill('#new-task-title', task);
  await page.fill('#new-task-notes', 'Bring the labels.');
  await page.selectOption('#new-task-stage', 'todo');
  await page.selectOption('#new-task-people', [{ label: me }]);
  const twoDaysAgo = new Date();
  twoDaysAgo.setDate(twoDaysAgo.getDate() - 2);
  await page.fill('#new-task-due-date', typed(twoDaysAgo)); // overdue, and so on the Briefing whatever day it is
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  const event = uniqueName('Rota meeting');
  await page.goto(new URL('/calendar/new', page.url()).href);
  await ready(page);
  await page.fill('#event-title', event);
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1); // always inside "this week" on the Briefing
  await page.fill('#event-start-date', typed(tomorrow));
  await page.fill('#event-notes', 'Print the agenda.');
  await page.click('.calendar-event-form button[type="submit"]');
  await ready(page);
  await page.goto(new URL('/briefing', page.url()).href);
  await ready(page);
  return { me, task, event };
}

test('gate 4.44: the date is a very large condensed heading with the day and month in the deeper accent', async ({ page, server, mockup }) => {
  await briefingWithItems(page, server.baseURL);
  await expectMatchesMockup(mockup, page, { mockup: '.brief-head h1', screen: 'briefing', app: '.briefing-headline' }, [...TEXT, 'line-height', 'margin-top']);
  await expectMatchesMockup(mockup, page, { mockup: '.brief-head h1 span', screen: 'briefing', app: '.brief-date' }, ['color']);
  await expect(page.locator('.brief-date')).toHaveCSS('color', 'rgb(169, 64, 0)'); // #A94000
  await expectMatchesMockup(mockup, page, { mockup: '.brief-head .eyebrow', screen: 'briefing', app: '.brief-tag' }, TEXT);
  await expectMatchesMockup(mockup, page, { mockup: '.brief-head', screen: 'briefing', app: '.brief-head' }, ['display', 'align-items', 'column-gap', 'flex-wrap']);
  // The h1's text is still the SPEC headline (gate 3.02), and the tag is drawn above the date.
  await expect(page.locator('h1')).toHaveText(/^(Briefing for )?\w+ \d+ \w+( — today's briefing)?$/);
  const tag = (await page.locator('.brief-tag').boundingBox())!;
  const title = (await page.locator('.brief-title').boundingBox())!;
  expect(tag.y + tag.height).toBeLessThanOrEqual(title.y + 2);
  // The Everyone / Just mine switch is the shared segmented control, at the right.
  await expectMatchesMockup(mockup, page, { mockup: '.brief-head .toggle .seg', screen: 'briefing', app: '.briefing-filter' }, ['display', 'border-top-color', 'border-radius']);
  await expect(page.locator('.briefing-filter a')).toHaveText(['Everyone', 'Just mine']);
  await expect(page.locator('.briefing-filter [aria-current]')).toHaveText('Everyone');
  const head = (await page.locator('.brief-head').boundingBox())!;
  const seg = (await page.locator('.briefing-filter').boundingBox())!;
  expect(seg.x + seg.width).toBeGreaterThan(head.x + head.width - 4);
});

test('gate 4.45: each panel heading is condensed capitals over a thick ink rule, with its count beside it', async ({ page, server, mockup }) => {
  await briefingWithItems(page, server.baseURL);
  const heads = '.briefing-section .panel-head';
  await expectMatchesMockup(mockup, page, { mockup: '.panel-head', screen: 'briefing', app: heads }, ['display', 'align-items', 'column-gap', 'border-bottom-width', 'border-bottom-style', 'border-bottom-color', 'padding-bottom']);
  await expectMatchesMockup(mockup, page, { mockup: '.panel-head h2', screen: 'briefing', app: `${heads} h2` }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.panel-head .count', screen: 'briefing', app: `${heads} .count` }, TEXT);
  expect(await page.locator(heads).count()).toBe(3);
  await expect(page.locator('#briefing-mustdo ~ .count')).toContainText(/\d+ tasks? · due before next Saturday/);
  await expect(page.locator('#briefing-coming ~ .count')).toHaveText('shown ahead of time');
  await expect(page.locator('#briefing-week ~ .count')).toHaveText(/^to \w{3} \d+ \w{3}$/);
});

test('gate 4.46: items are white cards — title, then date and people, stamps aligned right, an event note in a pale strip, an overdue card outlined in red', async ({
  page,
  server,
  mockup,
}) => {
  const { me, task, event } = await briefingWithItems(page, server.baseURL);
  const taskCard = page.locator('.briefing-card', { hasText: task });
  const eventCard = page.locator('.briefing-card', { hasText: event });
  await expectMatchesMockup(mockup, page, { mockup: '.item:not(.late)', screen: 'briefing', app: '.briefing-event' }, [...BOX.slice(0, 5), 'padding', 'display', 'row-gap', 'column-gap', 'align-items']);
  await expectMatchesMockup(mockup, page, { mockup: '.item h3', screen: 'briefing', app: '.briefing-event h3' }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.item .meta', screen: 'briefing', app: '.briefing-event .meta' }, ['display', 'align-items', 'column-gap', 'flex-wrap', 'font-size', 'color']);
  await expectMatchesMockup(mockup, page, { mockup: '.item .meta .date', screen: 'briefing', app: '.briefing-when' }, [...TEXT, 'font-variant-numeric']);
  await expectMatchesMockup(mockup, page, { mockup: '.item .note', screen: 'briefing', app: '.briefing-event .note' }, ['background-color', 'border-radius', 'padding', 'font-size', 'color']);
  await expect(eventCard.locator('.note')).toHaveText('Print the agenda.');

  // The overdue job's card is outlined in red (the mockup's soft red line).
  await expect(taskCard).toHaveClass(/\blate\b/);
  await expectMatchesMockup(mockup, page, { mockup: '.item.late', screen: 'briefing', app: '.briefing-task.late' }, ['border-top-color', 'border-top-width', 'background-color']);
  await expect(taskCard.locator('.stamp-overdue')).toHaveText('OVERDUE');
  await expect(taskCard.locator('.briefing-when')).toHaveText(/^Was due /);

  // Title first, then date and people; stamps on the right, YOURS to the left of the date stamp.
  const title = (await taskCard.locator('h3').boundingBox())!;
  const meta = (await taskCard.locator('.meta').boundingBox())!;
  expect(title.y + title.height).toBeLessThanOrEqual(meta.y + 2);
  const card = (await taskCard.boundingBox())!;
  const stamps = (await taskCard.locator('.stamps').boundingBox())!;
  expect(stamps.x).toBeGreaterThan(card.x + card.width / 2);
  const yours = (await taskCard.locator('.stamp-yours').boundingBox())!;
  const dateStamp = (await taskCard.locator('.stamp-overdue').boundingBox())!;
  expect(yours.x + yours.width).toBeLessThanOrEqual(dateStamp.x + 2);
  // People are circles, and mine is the accent one.
  await expect(taskCard.locator('.briefing-people .av')).toHaveAttribute('title', me);
  await expect(taskCard.locator('.briefing-people .av')).toHaveClass(/\bme\b/);
  await axeCheck(page);
  await expectNoSideScroll(page);
});

test('gate 4.47: the name picker is a full ink-navy screen with the large wordmark, the question in condensed capitals, and a grid of large names that turn accent when pointed at', async ({
  page,
  server,
  mockup,
}) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  await page.goto(server.baseURL + '/who');
  await expect(page.locator('.person-button').first()).toBeVisible();

  await expectMatchesMockup(mockup, page, { mockup: '.picker', screen: 'picker', app: '.picker' }, ['background-color', 'display', 'padding']);
  await expect(page.locator('body')).toHaveCSS('background-color', 'rgb(20, 27, 45)');
  await expectMatchesMockup(mockup, page, { mockup: '.picker-inner', screen: 'picker', app: '.picker-inner' }, ['display', 'row-gap', 'justify-items', 'text-align', 'max-width']);
  await expectMatchesMockup(mockup, page, { mockup: '.picker .mark small', screen: 'picker', app: '.picker .wordmark small' }, [...TEXT, 'display']);
  await expectMatchesMockup(mockup, page, { mockup: '.picker .mark strong', screen: 'picker', app: '.picker .wordmark strong' }, TEXT);
  await expectMatchesMockup(mockup, page, { mockup: '.picker h1', screen: 'picker', app: '.picker h1' }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.names', screen: 'picker', app: '.people-grid' }, ['display', 'row-gap', 'column-gap']);
  const columns = await page.locator('.people-grid').evaluate((el) => getComputedStyle(el).gridTemplateColumns.split(' ').length);
  expect(columns).toBe(4);
  await expectMatchesMockup(mockup, page, { mockup: '.names button:not(.hover)', screen: 'picker', app: '.person-button' }, [...TEXT, ...BOX]);
  await expectMatchesMockup(mockup, page, { mockup: '.picker a', screen: 'picker', app: '.picker .link-button' }, ['color', 'font-size', 'text-decoration-line', 'text-decoration-color', 'text-underline-offset']);

  // Pointed at, a name turns accent with ink text.
  await mockup.page.hover('.names button:not(.hover)');
  await page.locator('.person-button').first().hover();
  const hovered = diffStyles(
    await readStyles(mockup.page, '.names button:not(.hover)', ['background-color', 'border-top-color', 'color']),
    await readStyles(page, '.person-button', ['background-color', 'border-top-color', 'color']),
  );
  expect(hovered, 'hovered name').toEqual([]);
  await expect(page.locator('.person-button').first()).toHaveCSS('background-color', 'rgb(255, 107, 26)');
  await expect(page.locator('.person-button').first()).toHaveCSS('color', 'rgb(20, 27, 45)');
});

test('the picker passes the accessibility and window-size checks', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  await page.goto(server.baseURL + '/who');
  await axeCheck(page);
  await expectNoSideScroll(page);
});
