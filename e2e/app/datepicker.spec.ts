import { expect, test } from '../helpers/fixtures';
import { test as testToday } from '../helpers/fixtures-today';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask, openTaskPage } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// data-ready (SPEC B2) marks the server-rendered page, not that every
// script tag has finished loading and run; under load, a datepicker
// interaction can otherwise land before datepicker.js has registered its
// listeners. Every test in this file needs it, so it replaces the bare
// ready() import everywhere below.
async function readyForDatepicker(page: Page): Promise<void> {
  await ready(page);
  await page.waitForSelector('body.has-datepicker', { state: 'attached' });
}

async function todayDayFirst(page: Page): Promise<string> {
  const iso = await page.evaluate(() => document.body.dataset.today);
  const [y, m, d] = iso!.split('-');
  return `${d}/${m}/${y}`;
}

async function openPicker(page: Page, fieldId: string): Promise<void> {
  await page.locator('#' + fieldId).click();
  await expect(page.locator('.datepicker')).toBeVisible();
}

async function addEvent(page: Page, baseURL: string, fields: Record<string, string>): Promise<void> {
  await page.goto(baseURL + '/calendar/new');
  await readyForDatepicker(page);
  for (const [name, value] of Object.entries(fields)) {
    await page.locator(`[name="${name}"]`).fill(value);
  }
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await readyForDatepicker(page);
}

// SPEC gate 6.10: every date box in the app opens a calendar. One test per
// box named in the gate, each: open, pick Today, assert the value.
test.describe('gate 6.10: every date box opens a calendar', () => {
  test('the add-a-task window\'s due date', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    await readyForDatepicker(page);
    await page.click('#add-task-link');
    await expect(page.locator('#task-window')).toBeVisible();
    const today = await todayDayFirst(page);
    await openPicker(page, 'new-task-due-date');
    await page.locator('.datepicker-foot .datepicker-today').click();
    await expect(page.locator('#new-task-due-date')).toHaveValue(today);
  });

  test('the add-a-task page\'s due date, with no window', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    await readyForDatepicker(page);
    await openNewTask(page);
    const today = await todayDayFirst(page);
    await openPicker(page, 'new-task-due-date');
    await page.locator('.datepicker-foot .datepicker-today').click();
    await expect(page.locator('#new-task-due-date')).toHaveValue(today);
  });

  test('editing a job\'s due date, on its own page', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    await readyForDatepicker(page);
    const title = uniqueName('Dated job');
    await openNewTask(page);
    await page.fill('#new-task-title', title);
    await page.click('.add-task-form button[type="submit"]');
    await readyForDatepicker(page);
    await openTaskPage(page, title);
    const today = await todayDayFirst(page);
    await openPicker(page, 'details-due-date');
    await page.locator('.datepicker-foot .datepicker-today').click();
    await expect(page.locator('#details-due-date')).toHaveValue(today);
  });

  test('editing a job\'s due date, in the task window', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    await readyForDatepicker(page);
    const title = uniqueName('Dated window job');
    await openNewTask(page);
    await page.fill('#new-task-title', title);
    await page.click('.add-task-form button[type="submit"]');
    await readyForDatepicker(page);
    await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').first().click();
    await expect(page.locator('#task-window')).toBeVisible();
    await page.click('[data-hook="window-edit"]');
    const today = await todayDayFirst(page);
    await openPicker(page, 'details-due-date');
    await page.locator('.datepicker-foot .datepicker-today').click();
    await expect(page.locator('#details-due-date')).toHaveValue(today);
  });

  test('an event\'s date, end date and until date', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar/new');
    await readyForDatepicker(page);
    const today = await todayDayFirst(page);
    for (const id of ['event-start-date', 'event-end-date', 'event-until-date']) {
      await openPicker(page, id);
      await page.locator('.datepicker-foot .datepicker-today').click();
      await expect(page.locator('#' + id)).toHaveValue(today);
    }
  });

  test('a "just this one" change\'s date and end date', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar');
    await readyForDatepicker(page);
    const title = uniqueName('Repeating concert');
    await addEvent(page, server.baseURL, { title, start_date: '12/12/2026' });
    await page.selectOption('#event-recurrence', 'yearly');
    await page.locator('.calendar-event-form button[type="submit"]').click();
    await readyForDatepicker(page);
    await page.goto(server.baseURL + '/calendar?month=2026-12');
    await readyForDatepicker(page);
    await page.locator('.calendar-chip', { hasText: title }).click();
    await readyForDatepicker(page);
    await page.click('text=Change just this one');
    await readyForDatepicker(page);

    const today = await todayDayFirst(page);
    for (const id of ['occurrence-start-date', 'occurrence-end-date']) {
      await openPicker(page, id);
      await page.locator('.datepicker-foot .datepicker-today').click();
      await expect(page.locator('#' + id)).toHaveValue(today);
    }
  });
});

// SPEC gate 6.11, 6.12: one month, Sunday first, picking a day fills the
// box as dd/mm/yyyy and closes the calendar.
test('gate 6.11, 6.12: the grid is Sunday first, and picking a specific day fills and closes', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar/new');
  await readyForDatepicker(page);
  await openPicker(page, 'event-start-date');

  await expect(page.locator('.datepicker-grid thead th')).toHaveText(['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']);
  await expect(page.locator('.datepicker-day.datepicker-saturday-col').first()).toBeVisible();

  const iso = await page.evaluate(() => document.body.dataset.today);
  await page.locator(`.datepicker-day[data-date="${iso}"]`).click();
  const [y, m, d] = iso!.split('-');
  await expect(page.locator('#event-start-date')).toHaveValue(`${d}/${m}/${y}`);
  await expect(page.locator('.datepicker')).toHaveCount(0);
});

// SPEC gate 6.13: Today and This Saturday, pinned to a Saturday and a
// Wednesday so "the coming Saturday" is unambiguous either way.
testToday.describe('gate 6.13: Today and This Saturday', () => {
  testToday.describe('on a Saturday (19 Sep 2026)', () => {
    testToday.use({ today: '2026-09-19' });

    testToday('@fresh This Saturday is today itself', async ({ page, server }) => {
      await signInAsNewPerson(page, server.baseURL, '/calendar/new');
      await readyForDatepicker(page);
      await openPicker(page, 'event-start-date');
      await page.locator('.datepicker-saturday').click();
      await expect(page.locator('#event-start-date')).toHaveValue('19/09/2026');
    });
  });

  testToday.describe('on a Wednesday (16 Sep 2026)', () => {
    testToday.use({ today: '2026-09-16' });

    testToday('@fresh This Saturday is the coming one, and Today is today', async ({ page, server }) => {
      await signInAsNewPerson(page, server.baseURL, '/calendar/new');
      await readyForDatepicker(page);
      await openPicker(page, 'event-start-date');
      await page.locator('.datepicker-saturday').click();
      await expect(page.locator('#event-start-date')).toHaveValue('19/09/2026');

      await openPicker(page, 'event-end-date');
      await page.locator('.datepicker-foot .datepicker-today').click();
      await expect(page.locator('#event-end-date')).toHaveValue('16/09/2026');
    });
  });
});

// SPEC gate 6.14: typing moves the calendar to match, and Escape closes
// leaving what was typed alone.
test('gate 6.14: typing moves the calendar, and Escape keeps the typed text', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar/new');
  await readyForDatepicker(page);
  await page.locator('#event-start-date').click();
  await expect(page.locator('.datepicker')).toBeVisible();

  await page.locator('#event-start-date').fill('25/12/2026');
  await expect(page.locator('.datepicker-month')).toHaveText('December 2026');
  await expect(page.locator('.datepicker-day.datepicker-selected')).toHaveAttribute('data-date', '2026-12-25');

  await page.keyboard.press('Escape');
  await expect(page.locator('.datepicker')).toHaveCount(0);
  await expect(page.locator('#event-start-date')).toHaveValue('25/12/2026');
});

// SPEC gate 6.15: an empty end date opens on the start date's month.
test('gate 6.15: an empty end date opens on the start date\'s month', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar/new');
  await readyForDatepicker(page);
  await page.fill('#event-start-date', '5/3/2027');
  await page.locator('#event-end-date').click();
  await expect(page.locator('.datepicker-month')).toHaveText('March 2027');
});

// SPEC gate 6.16: the whole flow works from the keyboard alone: Down
// arrow from the box into the grid, the arrow keys, Page Up/Down for a
// month, Home for the week's start, and Enter to pick.
test('gate 6.16: the whole flow works from the keyboard alone', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar/new');
  await readyForDatepicker(page);

  await page.locator('#event-start-date').focus();
  await page.keyboard.press('ArrowDown');
  await expect(page.locator('.datepicker-day[tabindex="0"]')).toBeFocused();
  const startDate = await page.locator('.datepicker-day[tabindex="0"]').getAttribute('data-date');

  await page.keyboard.press('ArrowRight');
  const afterRight = await page.locator('.datepicker-day[tabindex="0"]').getAttribute('data-date');
  const expectedNext = new Date(startDate! + 'T00:00:00');
  expectedNext.setDate(expectedNext.getDate() + 1);
  expect(afterRight).toBe(expectedNext.toISOString().slice(0, 10));

  const monthBefore = await page.locator('.datepicker-month').textContent();
  await page.keyboard.press('PageUp');
  await expect(page.locator('.datepicker-month')).not.toHaveText(monthBefore ?? '');

  // Home moves to the start of the active day's week, then Enter picks it.
  await page.keyboard.press('Home');
  const homeCellDate = await page.locator('.datepicker-day[tabindex="0"]').getAttribute('data-date');
  await page.keyboard.press('Enter');
  const [y, m, d] = homeCellDate!.split('-');
  await expect(page.locator('#event-start-date')).toHaveValue(`${d}/${m}/${y}`);
  await expect(page.locator('.datepicker')).toHaveCount(0);
  await expect(page.locator('#event-start-date')).toBeFocused();
});

// SPEC gate 6.16: a screen reader hears the month's name and each day's
// full date, and the picked day carries aria-selected.
test('gate 6.16: accessible names for the month and each day', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar/new');
  await readyForDatepicker(page);
  await page.fill('#event-start-date', '25/9');
  await page.locator('#event-start-date').click();
  await expect(page.locator('.datepicker-month')).toHaveAttribute('aria-live', 'polite');
  const iso = await page.evaluate(() => document.body.dataset.today);
  const selectedCell = page.locator('.datepicker-day.datepicker-selected');
  await expect(selectedCell).toHaveAttribute('aria-selected', 'true');
  expect(iso).toBeTruthy();
});

// SPEC gate 6.16: the accessibility check passes with the calendar open,
// both in the task window and on the event form.
test('gate 6.16: axe passes with the calendar open, on the event form', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar/new');
  await readyForDatepicker(page);
  await openPicker(page, 'event-start-date');
  await axeCheck(page);
});

test('gate 6.16: axe passes with the calendar open, in the task window', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await readyForDatepicker(page);
  await page.click('#add-task-link');
  await expect(page.locator('#task-window')).toBeVisible();
  await openPicker(page, 'new-task-due-date');
  await axeCheck(page);
});

// SPEC gate 6.41 (this task's own share of it): the calendar never makes
// the page scroll sideways.
test('no sideways scroll with the calendar open', async ({ page, server }) => {
  await page.setViewportSize({ width: 1024, height: 700 });
  await signInAsNewPerson(page, server.baseURL, '/calendar/new');
  await readyForDatepicker(page);
  await openPicker(page, 'event-start-date');
  await expectNoSideScroll(page);
});
