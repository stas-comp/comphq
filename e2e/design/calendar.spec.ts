import { expect, test } from './fixtures';
import { expectMatchesMockup } from './mockup';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

// The Calendar (PLAN P4-11, gates 4.33-4.38): the month grid as drawn, and
// the Saturday that is actually visible.

type Page = import('@playwright/test').Page;

const TEXT = ['font-family', 'font-weight', 'font-size', 'letter-spacing', 'text-transform', 'color'];
const BOX = ['background-color', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius', 'padding'];

async function addEvent(page: Page, title: string, start: string, end?: string): Promise<void> {
  await page.goto(new URL('/calendar/new', page.url()).href);
  await ready(page);
  await page.fill('#event-title', title);
  await page.fill('#event-start-date', start);
  if (end) await page.fill('#event-end-date', end);
  await page.click('.calendar-event-form button[type="submit"]');
  await ready(page);
}

async function addTask(page: Page, title: string, due: string): Promise<void> {
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', 'todo');
  await page.fill('#new-task-due-date', due);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

// March 2027: the 20th is a Saturday. One single-day event on it, a three-day
// event, and a task due; plus, in March 2020, a task that is long overdue.
async function seeded(page: Page, baseURL: string) {
  await signInAsNewPerson(page, baseURL, '/calendar');
  await ready(page);
  const t = { sat: uniqueName('Saturday meeting'), multi: uniqueName('Exam week'), due: uniqueName('Task due'), late: uniqueName('Long overdue') };
  await addEvent(page, t.sat, '20/03/2027');
  await addEvent(page, t.multi, '22/03/2027', '24/03/2027');
  await addTask(page, t.due, '25/03/2027');
  await addTask(page, t.late, '12/03/2020');
  return t;
}

const MONTH = '/calendar?month=2027-03';

test('gate 4.38: the controls are the mockup ones — Previous, Today and Next secondary buttons, a Month / List switch, and a primary + Add event', async ({
  page,
  server,
  mockup,
}) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar');
  await ready(page);
  await expectMatchesMockup(mockup, page, { mockup: '.tools h1', screen: 'calendar', app: '.tools h1' }, [...TEXT, 'line-height']);
  await expect(page.locator('.tools h1')).toHaveText(/^[A-Z][a-z]+ \d{4}$/); // the month, as the mockup titles the screen
  for (const [name, text] of [['Previous', '‹ Previous'], ['Today', 'Today'], ['Next', 'Next ›']]) {
    const button = page.locator('.tools').getByRole('link', { name });
    await expect(button).toHaveText(text);
    await expect(button).toHaveClass(/\bbtn\b/);
    await expect(button).not.toHaveClass(/\bprimary\b/);
  }
  await expectMatchesMockup(mockup, page, { mockup: '.tools .btn:not(.primary)', screen: 'calendar', app: '.tools a.btn:not(.primary)' }, [...TEXT, ...BOX]);
  await expectMatchesMockup(mockup, page, { mockup: '.tools .btn.primary', screen: 'calendar', app: '.tools .tools-add' }, [...TEXT, ...BOX]);
  await expect(page.locator('.tools .tools-add')).toHaveText('+ Add event');
  await expect(page.locator('.tools .seg')).toHaveClass(/\bseg\b/);
  await expect(page.locator('.tools .seg [aria-current="page"]')).toHaveText('Month');
  await expect(page.locator('.tools .seg a')).toHaveText(['Month', 'List']);
  await expectMatchesMockup(mockup, page, { mockup: '.tools .seg', screen: 'calendar', app: '.tools .seg' }, ['border-top-width', 'border-top-color', 'border-radius']);

  // Today sits between them and goes to the current month; Next goes on one.
  const heading = (await page.locator('.tools h1').textContent())!;
  await page.getByRole('link', { name: 'Next ›' }).click();
  await ready(page);
  await expect(page.locator('.tools h1')).not.toHaveText(heading);
  await page.getByRole('link', { name: 'Today' }).click();
  await ready(page);
  await expect(page.locator('.tools h1')).toHaveText(heading);
});

test('gate 4.33: Saturday is unmistakable — a solid accent heading and an accent-wash column', async ({ page, server, mockup }) => {
  await seeded(page, server.baseURL);
  await page.goto(server.baseURL + MONTH);
  await ready(page);
  await expectMatchesMockup(mockup, page, { mockup: '.cal-grid .dow.sat', screen: 'calendar', app: '.calendar-saturday-heading' }, [...TEXT, 'background-color', 'padding', 'border-right-width', 'border-bottom-width']);
  await expectMatchesMockup(mockup, page, { mockup: '.cal-grid .dow:not(.sat)', screen: 'calendar', app: '.calendar-grid th:not(.calendar-saturday-heading)' }, [...TEXT, 'background-color', 'padding']);
  await expect(page.locator('.calendar-saturday-heading')).toHaveCSS('background-color', 'rgb(255, 107, 26)');
  await expect(page.locator('.calendar-saturday-heading')).toHaveCSS('color', 'rgb(20, 27, 45)');
  await expectMatchesMockup(mockup, page, { mockup: '.cal-grid', screen: 'calendar', app: '.calendar-grid' }, ['border-top-width', 'border-top-style', 'border-top-color', 'border-left-width', 'border-left-color']);

  // Every Saturday cell is the wash; the cells under Saturday are all in the sixth column.
  // (A Saturday that falls outside the month takes the outside shade, as in the mockup.)
  const sats = page.locator('.calendar-day-saturday:not(.calendar-day-outside)');
  const cells = await sats.count();
  expect(cells).toBeGreaterThanOrEqual(4);
  for (let i = 0; i < cells; i++) {
    await expect(sats.nth(i)).toHaveCSS('background-color', 'rgb(255, 243, 234)');
  }
  const col = await sats.first().evaluate((td) => (td as HTMLTableCellElement).cellIndex);
  expect(col).toBe(5); // Monday first: Saturday is the sixth
});

test('gate 4.34: ordinary days, Saturdays and days outside the month are three clearly different shades', async ({ page, server, mockup }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar');
  await page.goto(server.baseURL + MONTH);
  await ready(page);
  const bg = (sel: string) => page.locator(sel).first().evaluate((el) => getComputedStyle(el).backgroundColor);
  const ordinary = await bg('.calendar-day:not(.calendar-day-saturday):not(.calendar-day-outside):not(.calendar-day-today)');
  const saturday = await bg('.calendar-day-saturday:not(.calendar-day-outside)');
  const outside = await bg('.calendar-day-outside:not(.calendar-day-saturday)');
  expect(new Set([ordinary, saturday, outside]).size, `three distinct shades, got ${ordinary} / ${saturday} / ${outside}`).toBe(3);

  const day = ['background-color', 'border-right-width', 'border-right-style', 'border-right-color', 'border-bottom-width', 'border-bottom-color', 'padding'];
  await expectMatchesMockup(mockup, page, { mockup: '.day:not(.sat):not(.out):not(.today)', screen: 'calendar', app: '.calendar-day:not(.calendar-day-saturday):not(.calendar-day-outside):not(.calendar-day-today)' }, day);
  await expectMatchesMockup(mockup, page, { mockup: '.day.sat', screen: 'calendar', app: '.calendar-day-saturday:not(.calendar-day-outside)' }, day);
  await expectMatchesMockup(mockup, page, { mockup: '.day.out:not(.sat)', screen: 'calendar', app: '.calendar-day-outside:not(.calendar-day-saturday)' }, day);
  // The number keeps a colour that passes the readability check on the quieter shade (D-63).
  await expectMatchesMockup(mockup, page, { mockup: '.day:not(.out) .num', screen: 'calendar', app: '.calendar-day:not(.calendar-day-outside):not(.calendar-day-today) .calendar-day-number' }, [...TEXT, 'padding']);
});

test('gate 4.35: today\'s date number sits in a filled ink-navy chip', async ({ page, server, mockup }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar');
  await ready(page);
  const today = page.locator('.calendar-day-today .calendar-day-number');
  await expect(today).toHaveCount(1);
  await expectMatchesMockup(mockup, page, { mockup: '.day.today .num', screen: 'calendar', app: '.calendar-day-today .calendar-day-number' }, [...TEXT, 'background-color', 'border-radius', 'padding']);
  await expect(today).toHaveCSS('background-color', 'rgb(20, 27, 45)');
  await expect(today).toHaveCSS('color', 'rgb(255, 255, 255)');
  // Every other number has no fill.
  await expect(page.locator('.calendar-day:not(.calendar-day-today) .calendar-day-number').first()).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
});

test('gate 4.36: events are solid ink chips, task deadlines white outlined chips with an empty square (red when overdue), and a multi-day event goes lighter after its first day', async ({
  page,
  server,
  mockup,
}) => {
  const t = await seeded(page, server.baseURL);
  await page.goto(server.baseURL + MONTH);
  await ready(page);
  const chip = (title: string, date: string) => page.locator(`.calendar-day[data-date="${date}"] .calendar-chip`, { hasText: title });

  const chipProps = [...TEXT, 'background-color', 'border-radius', 'padding', 'line-height', 'white-space', 'text-overflow', 'overflow'];
  await expectMatchesMockup(mockup, page, { mockup: '.chip.ev:not(.cont)', screen: 'calendar', app: `.calendar-day[data-date="2027-03-20"] .calendar-chip-event` }, chipProps);
  await expect(chip(t.sat, '2027-03-20')).toHaveCSS('background-color', 'rgb(20, 27, 45)');
  await expect(chip(t.sat, '2027-03-20')).toHaveCSS('color', 'rgb(255, 255, 255)');

  // First day solid ink; the later days lighter.
  await expect(chip(t.multi, '2027-03-22')).toHaveCSS('background-color', 'rgb(20, 27, 45)');
  for (const day of ['2027-03-23', '2027-03-24']) {
    await expect(chip(t.multi, day)).toHaveClass(/calendar-chip-continues/);
    await expect(chip(t.multi, day)).toHaveCSS('background-color', 'rgb(44, 54, 86)');
  }
  await expectMatchesMockup(mockup, page, { mockup: '.chip.ev.cont', screen: 'calendar', app: `.calendar-day[data-date="2027-03-23"] .calendar-chip-continues` }, [...chipProps]);

  // A task: white, outlined in ink, with a small empty square.
  await expectMatchesMockup(mockup, page, { mockup: '.chip.task:not(.late)', screen: 'calendar', app: `.calendar-day[data-date="2027-03-25"] .calendar-chip-task` }, [...chipProps, 'display', 'column-gap', 'border-top-width', 'border-top-color']);
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.chip.task:not(.late)', screen: 'calendar', app: `.calendar-day[data-date="2027-03-25"] .calendar-chip-task`, pseudo: '::before' },
    ['width', 'height', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius', 'content'],
  );

  // An overdue task is red, square and all.
  await page.goto(server.baseURL + '/calendar?month=2020-03');
  await ready(page);
  const late = page.locator('.calendar-day[data-date="2020-03-12"] .calendar-chip-task', { hasText: t.late });
  await expect(late).toHaveClass(/calendar-chip-late/);
  await expectMatchesMockup(mockup, page, { mockup: '.chip.task.late', screen: 'calendar', app: `.calendar-day[data-date="2020-03-12"] .calendar-chip-late` }, [...chipProps, 'border-top-color']);
  await expectMatchesMockup(mockup, page, { mockup: '.chip.task.late', screen: 'calendar', app: `.calendar-day[data-date="2020-03-12"] .calendar-chip-late`, pseudo: '::before' }, ['border-top-color']);
  await expect(late).toHaveCSS('color', 'rgb(198, 45, 45)');
});

test('gate 4.37: a legend under the grid explains the two kinds of chip and the Saturday tint', async ({ page, server, mockup }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar');
  await page.goto(server.baseURL + MONTH);
  await ready(page);
  const legend = page.locator('.calendar-legend');
  await expect(legend.locator('.calendar-chip-event')).toHaveText('Event');
  await expect(legend.locator('.calendar-chip-task')).toHaveText('Task due');
  await expect(legend).toContainText('Saturdays are highlighted');
  await expectMatchesMockup(mockup, page, { mockup: '.legend', screen: 'calendar', app: '.calendar-legend' }, ['display', 'align-items', 'column-gap', 'font-size', 'color']);
  await expectMatchesMockup(mockup, page, { mockup: '.legend .chip.ev', screen: 'calendar', app: '.calendar-legend .calendar-chip-event' }, ['background-color', 'color', 'font-size', 'font-weight']);
  await expectMatchesMockup(mockup, page, { mockup: '.legend .chip.task', screen: 'calendar', app: '.calendar-legend .calendar-chip-task' }, ['background-color', 'color', 'border-top-color', 'display']);
  // It is below the grid.
  const grid = (await page.locator('.calendar-grid').boundingBox())!;
  const box = (await legend.boundingBox())!;
  expect(box.y).toBeGreaterThanOrEqual(grid.y + grid.height - 1);
});

test('standard page checks for the month with events, a task and Saturdays, and the list view', async ({ page, server }) => {
  await seeded(page, server.baseURL);
  for (const path of [MONTH, '/calendar?month=2020-03', '/calendar/list']) {
    await page.goto(server.baseURL + path);
    await ready(page);
    await axeCheck(page);
    await expectNoSideScroll(page);
  }
});
