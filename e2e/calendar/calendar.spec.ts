import { expect, test } from '../helpers/fixtures';
import { test as testToday } from '../helpers/fixtures-today';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

async function addEvent(page: Page, baseURL: string, fields: Record<string, string>): Promise<void> {
  await page.goto(baseURL + '/calendar/new');
  await ready(page);
  for (const [name, value] of Object.entries(fields)) {
    await page.locator(`[name="${name}"]`).fill(value);
  }
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);
}

// SPEC gate 2.12: weeks start Monday, Saturdays are highlighted, today
// is marked, and Previous/Next/Today navigate. COMPHQ_TEST_TODAY is
// PLAN.md's own fixed reference date for this gate.
testToday.describe('gate 2.12: month view', () => {
  testToday.use({ today: '2026-09-16' });

  testToday('@fresh Saturdays highlighted, today marked, and navigation works', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar');
    await ready(page);

    await expect(page.locator('h2')).toHaveText('September 2026');
    await expect(page.locator('.calendar-day-today')).toHaveAttribute('data-date', '2026-09-16');

    // Every Saturday in the visible grid is highlighted, and no other
    // day is (a real check, not just "at least one").
    const saturdayCells = page.locator('.calendar-day-saturday');
    const saturdayDates = await saturdayCells.evaluateAll((els) => els.map((el) => el.getAttribute('data-date')));
    for (const date of saturdayDates) {
      expect(new Date(date + 'T00:00:00Z').getUTCDay()).toBe(6);
    }
    expect(saturdayDates.length).toBeGreaterThanOrEqual(4);

    await page.click('text=Previous');
    await ready(page);
    await expect(page.locator('h2')).toHaveText('August 2026');

    await page.click('text=Next');
    await ready(page);
    await page.click('text=Next');
    await ready(page);
    await expect(page.locator('h2')).toHaveText('October 2026');

    await page.click('text=Today');
    await ready(page);
    await expect(page.locator('h2')).toHaveText('September 2026');
  });

  testToday('@fresh clicking a day starts a new event on that date', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar');
    await ready(page);

    await page.locator('.calendar-day[data-date="2026-09-24"] .calendar-day-number').click();
    await ready(page);
    await expect(page.locator('#event-start-date')).toHaveValue('24/09/2026');
  });
});

// SPEC gate 2.12: "The List view shows the next 12 weeks in date order."
testToday.describe('gate 2.12: list view', () => {
  testToday.use({ today: '2026-09-16' });

  testToday('@fresh shows events in date order across 12 weeks, excluding anything further out', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar');
    await ready(page);

    const later = uniqueName('Later this quarter');
    const soon = uniqueName('Very soon');
    const middle = uniqueName('Mid-window');
    const tooFar = uniqueName('Too far out');
    await addEvent(page, server.baseURL, { title: soon, start_date: '2026-09-18' });
    await addEvent(page, server.baseURL, { title: middle, start_date: '2026-10-15' });
    await addEvent(page, server.baseURL, { title: later, start_date: '2026-12-05' }); // within 12 weeks of 2026-09-16
    await addEvent(page, server.baseURL, { title: tooFar, start_date: '2027-06-01' }); // well beyond 12 weeks

    await page.goto(server.baseURL + '/calendar/list');
    await ready(page);

    const titles = await page.locator('.calendar-list-title').allTextContents();
    const indices = [soon, middle, later].map((t) => titles.findIndex((got) => got.includes(t)));
    expect(indices.every((i) => i >= 0)).toBe(true);
    expect(indices).toEqual([...indices].sort((a, b) => a - b));
    expect(titles.some((t) => t.includes(tooFar))).toBe(false);
  });
});

// SPEC gate 2.13: every field can be added, and is saved and redisplayed.
test('gate 2.13: every field is saved and redisplayed', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar');
  await ready(page);

  const title = uniqueName('Christmas concert');
  await addEvent(page, server.baseURL, {
    title,
    notes: 'Bring programmes',
    start_date: '2026-12-12',
    end_date: '2026-12-13',
    start_time: '18:00',
    end_time: '20:00',
    until_date: '2030-12-31',
  });
  await page.selectOption('#event-recurrence', 'yearly');
  await page.fill('#event-notice-amount', '2');
  await page.selectOption('#event-notice-unit', 'weeks');
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);

  await expect(page.locator('h1')).toHaveText(title);
  await expect(page.locator('#event-title')).toHaveValue(title);
  await expect(page.locator('#event-notes')).toHaveValue('Bring programmes');
  await expect(page.locator('#event-start-date')).toHaveValue('12/12/2026');
  await expect(page.locator('#event-end-date')).toHaveValue('13/12/2026');
  await expect(page.locator('#event-start-time')).toHaveValue('18:00');
  await expect(page.locator('#event-end-time')).toHaveValue('20:00');
  await expect(page.locator('#event-recurrence')).toHaveValue('yearly');
  await expect(page.locator('#event-until-date')).toHaveValue('31/12/2030');
  await expect(page.locator('#event-notice-amount')).toHaveValue('2');
  await expect(page.locator('#event-notice-unit')).toHaveValue('weeks');
});

// SPEC gate 2.20: an event's details show who last changed it and when.
test('gate 2.20: details show who last changed it and when, updated after an edit', async ({ page, server }) => {
  const name = await signInAsNewPerson(page, server.baseURL, '/calendar');
  await ready(page);

  const title = uniqueName('Tracked event');
  await addEvent(page, server.baseURL, { title, start_date: '2026-10-01' });
  await expect(page.locator('.calendar-last-changed')).toContainText(`Last changed by ${name}`);

  await page.fill('#event-notes', 'Edited notes');
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);
  await expect(page.locator('#event-notes')).toHaveValue('Edited notes');
  await expect(page.locator('.calendar-last-changed')).toContainText(`Last changed by ${name}`);
});

// Keyboard-only: add an event without ever clicking.
test('an event can be added with keyboard only', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar/new');
  await ready(page);

  const title = uniqueName('Keyboard event');
  await page.locator('#event-title').focus();
  await page.keyboard.type(title);
  await page.locator('#event-start-date').focus();
  await page.keyboard.press('Control+A');
  await page.keyboard.type('01/10/2026'); // day first, gate 4.05
  await page.locator('.calendar-event-form button[type="submit"]').focus();
  await page.keyboard.press('Enter');
  await ready(page);

  await expect(page.locator('h1')).toHaveText(title);
  await expect(page.locator('#event-start-date')).toHaveValue('01/10/2026');
});

test('standard page checks for the month view, list view, add form and event details', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar');
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);

  await page.goto(server.baseURL + '/calendar/list');
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);

  await page.goto(server.baseURL + '/calendar/new');
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);

  const title = uniqueName('Checked event');
  await addEvent(page, server.baseURL, { title, start_date: '2026-10-05' });
  await axeCheck(page);
  await expectNoSideScroll(page);
});

// Opens a repeating occurrence's own chip and picks "Change just this
// one" (SPEC A7: "Clicking a repeating event asks...").
async function openJustThisOne(page: Page, title: string): Promise<void> {
  await page.locator('.calendar-chip', { hasText: title }).click();
  await ready(page);
  await page.click('text=Change just this one');
  await ready(page);
}

// SPEC gate 2.15: the exact script — a yearly Christmas concert moved
// just once shows on its new date, and the other year is unaffected.
test('gate 2.15: moving just one occurrence of a repeating event leaves other years unaffected', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar');
  await ready(page);

  const title = uniqueName('Christmas concert');
  await addEvent(page, server.baseURL, { title, start_date: '2026-12-12' });
  await page.selectOption('#event-recurrence', 'yearly');
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);

  await page.goto(server.baseURL + '/calendar?month=2026-12');
  await ready(page);
  await openJustThisOne(page, title);

  await expect(page.locator('h1')).toHaveText('Change just this one');
  await expect(page.locator('#occurrence-start-date')).toHaveValue('12/12/2026');
  await page.fill('#occurrence-start-date', '2026-12-19');
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);

  await page.goto(server.baseURL + '/calendar?month=2026-12');
  await ready(page);
  await expect(page.locator('.calendar-day[data-date="2026-12-12"] .calendar-chip', { hasText: title })).toHaveCount(0);
  await expect(page.locator('.calendar-day[data-date="2026-12-19"] .calendar-chip', { hasText: title })).toHaveCount(1);

  await page.goto(server.baseURL + '/calendar?month=2027-12');
  await ready(page);
  await expect(page.locator('.calendar-day[data-date="2027-12-12"] .calendar-chip', { hasText: title })).toHaveCount(1);
});

// SPEC gate 2.16: "Cancel just this one" hides only that occurrence.
test('gate 2.16: cancelling just one occurrence leaves the others', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar');
  await ready(page);

  const title = uniqueName('Weekly sync');
  await addEvent(page, server.baseURL, { title, start_date: '2026-03-02' });
  await page.selectOption('#event-recurrence', 'weekly');
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);

  await page.goto(server.baseURL + '/calendar?month=2026-03');
  await ready(page);
  await page.locator('.calendar-day[data-date="2026-03-09"] .calendar-chip', { hasText: title }).click();
  await ready(page);
  await page.click('text=Change just this one');
  await ready(page);
  await page.click('text=Cancel just this one');
  await ready(page);

  await page.goto(server.baseURL + '/calendar?month=2026-03');
  await ready(page);
  await expect(page.locator('.calendar-day[data-date="2026-03-09"] .calendar-chip', { hasText: title })).toHaveCount(0);
  await expect(page.locator('.calendar-day[data-date="2026-03-02"] .calendar-chip', { hasText: title })).toHaveCount(1);
  await expect(page.locator('.calendar-day[data-date="2026-03-16"] .calendar-chip', { hasText: title })).toHaveCount(1);
});

// SPEC gate 2.17: "Change all" shows on every occurrence, and a
// previously moved occurrence keeps its own date but shows the new title.
test('gate 2.17: Change all updates every occurrence, and a moved one keeps its date with the new title', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar');
  await ready(page);

  const originalTitle = uniqueName('Annual gala');
  await addEvent(page, server.baseURL, { title: originalTitle, start_date: '2026-06-01' });
  await page.selectOption('#event-recurrence', 'yearly');
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);

  // Move the 2027 occurrence just this once.
  await page.goto(server.baseURL + '/calendar?month=2027-06');
  await ready(page);
  await openJustThisOne(page, originalTitle);
  await page.fill('#occurrence-start-date', '2027-06-08');
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);

  // Change all: edit the series' own title from its details page.
  const newTitle = uniqueName('Annual gala renamed');
  await page.fill('#event-title', newTitle);
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);

  await page.goto(server.baseURL + '/calendar?month=2026-06');
  await ready(page);
  await expect(page.locator('.calendar-day[data-date="2026-06-01"] .calendar-chip', { hasText: newTitle })).toHaveCount(1);

  await page.goto(server.baseURL + '/calendar?month=2027-06');
  await ready(page);
  await expect(page.locator('.calendar-day[data-date="2027-06-01"] .calendar-chip', { hasText: newTitle })).toHaveCount(0);
  await expect(page.locator('.calendar-day[data-date="2027-06-08"] .calendar-chip', { hasText: newTitle })).toHaveCount(1);
});

// SPEC gate 2.18: Remove hides the whole event; Restore brings it back.
test('gate 2.18: Remove hides an event and lists it in Removed events, and Restore brings it back', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar');
  await ready(page);

  const title = uniqueName('Removable event');
  await addEvent(page, server.baseURL, { title, start_date: '2026-10-10' });
  await page.locator('.calendar-remove-form button[type="submit"]').click();
  await ready(page);

  await page.goto(server.baseURL + '/calendar?month=2026-10');
  await ready(page);
  await expect(page.locator('.calendar-day[data-date="2026-10-10"] .calendar-chip', { hasText: title })).toHaveCount(0);

  await page.goto(server.baseURL + '/calendar/removed');
  await ready(page);
  await expect(page.locator('.calendar-list-title', { hasText: title })).toHaveCount(1);

  await page.locator('.calendar-list-row', { hasText: title }).locator('button', { hasText: 'Restore' }).click();
  await ready(page);
  await expect(page.locator('.calendar-list-title', { hasText: title })).toHaveCount(0);

  await page.goto(server.baseURL + '/calendar?month=2026-10');
  await ready(page);
  await expect(page.locator('.calendar-day[data-date="2026-10-10"] .calendar-chip', { hasText: title })).toHaveCount(1);
});

test('standard page checks for Removed events', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar/removed');
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);
});

async function addTask(page: Page, title: string, stage: string, dueDate: string): Promise<void> {
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', stage);
  await page.fill('#new-task-due-date', dueDate);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

// SPEC gate 2.19: unfinished tasks with due dates show on their date,
// looking different from events (outlined, data-kind="task"), and
// open the task; finished and removed tasks don't show at all.
test('gate 2.19: a due task shows as an outlined chip and opens the task, but finished and removed tasks don\'t show', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const dueTitle = uniqueName('Due task');
  await addTask(page, dueTitle, 'todo', '2026-09-25');

  const doneTitle = uniqueName('Finished task');
  await addTask(page, doneTitle, 'done', '2026-09-25');

  const removedTitle = uniqueName('Removed task');
  await addTask(page, removedTitle, 'todo', '2026-09-25');
  await page.locator('.task-card', { hasText: removedTitle }).locator('.task-card-title a').click();
  await ready(page);
  await page.locator('.task-remove-form button[type="submit"]').click();
  await ready(page);

  await page.goto(server.baseURL + '/calendar?month=2026-09');
  await ready(page);

  const cell = page.locator('.calendar-day[data-date="2026-09-25"]');
  const dueChip = cell.locator('.calendar-chip', { hasText: dueTitle });
  await expect(dueChip).toHaveAttribute('data-kind', 'task');
  await expect(cell.locator('.calendar-chip', { hasText: doneTitle })).toHaveCount(0);
  await expect(cell.locator('.calendar-chip', { hasText: removedTitle })).toHaveCount(0);

  await dueChip.click();
  await ready(page);
  await expect(page).toHaveURL(/\/tasks\/\d+$/);
  await expect(page.locator('h1')).toHaveText(dueTitle);
});
