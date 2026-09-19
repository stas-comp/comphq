import { expect, test } from '../helpers/fixtures-today';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';

type Page = import('@playwright/test').Page;

// Every test here is @fresh: it needs its own server with its own
// COMPHQ_TEST_TODAY (SPEC B4), which also means each starts with an
// empty database, so plain titles never collide with another test's.

async function addTask(
  page: Page,
  baseURL: string,
  task: { title: string; due?: string; stage?: string; people?: string[] },
): Promise<void> {
  await page.goto(baseURL + '/tasks/board');
  await ready(page);
  await openNewTask(page);
  await page.fill('#new-task-title', task.title);
  await page.selectOption('#new-task-stage', task.stage ?? 'todo');
  if (task.due) await page.fill('#new-task-due-date', task.due);
  if (task.people) await page.selectOption('#new-task-people', task.people.map((label) => ({ label })));
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

async function addEvent(page: Page, baseURL: string, fields: Record<string, string>): Promise<void> {
  await page.goto(baseURL + '/calendar/new');
  await ready(page);
  const { recurrence, notice_amount, notice_unit, ...rest } = fields;
  for (const [name, value] of Object.entries(rest)) {
    await page.locator(`[name="${name}"]`).fill(value);
  }
  if (recurrence) await page.selectOption('#event-recurrence', recurrence);
  if (notice_amount) await page.fill('#event-notice-amount', notice_amount);
  if (notice_unit) await page.selectOption('#event-notice-unit', notice_unit);
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);
}

async function openBriefing(page: Page, baseURL: string, query = ''): Promise<void> {
  await page.goto(baseURL + '/briefing' + query);
  await ready(page);
}

function section(page: Page, heading: string) {
  return page.locator('.briefing-section', { has: page.getByRole('heading', { name: heading, exact: true }) });
}

// The card in a section whose title link reads `title`.
function card(page: Page, heading: string, title: string) {
  return section(page, heading).locator('.briefing-card', { has: page.getByRole('link', { name: title, exact: true }) });
}

test.describe('on a Saturday (19 Sep 2026)', () => {
  test.use({ today: '2026-09-19' });

  // SPEC gate 3.01: opening Comp HQ shows the Briefing, the frame link
  // leads there too, and "Coming soon" is gone from the app.
  test('@fresh gate 3.01: opening Comp HQ shows the Briefing, and nothing says Coming soon', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/');
    await ready(page);

    expect(new URL(page.url()).pathname).toBe('/');
    await expect(page.locator('.briefing-headline')).toBeVisible();
    await expect(page.locator('.sidebar nav a[aria-current="page"]')).toHaveText('Briefing');

    await page.click('.sidebar nav a[href="/kb"]');
    await ready(page);
    await page.click('.sidebar nav a[href="/briefing"]');
    await ready(page);
    expect(new URL(page.url()).pathname).toBe('/briefing');
    await expect(page.locator('.briefing-headline')).toBeVisible();

    const routes: string[] = await (await page.request.get(server.baseURL + '/__test/routes')).json();
    for (const route of ['/', ...routes]) {
      await page.goto(server.baseURL + route);
      await ready(page);
      await expect(page.locator('main')).not.toContainText('Coming soon');
    }
  });

  // SPEC gate 3.02 (Saturday), 3.03, 3.05, 3.06, 3.07, 3.09, 3.10, 3.11.
  test('@fresh gates 3.02, 3.03, 3.05, 3.06, 3.07, 3.09, 3.11: the Saturday briefing', async ({ page, server }) => {
    const me = await signInAsNewPerson(page, server.baseURL, '/');
    await ready(page);

    await addTask(page, server.baseURL, { title: 'Wednesday job', due: '2026-09-23', people: [me] });
    await addTask(page, server.baseURL, { title: 'Next Saturday job', due: '2026-09-26' });
    await addTask(page, server.baseURL, { title: 'Old job', due: '2026-09-10', stage: 'idea' });
    await addTask(page, server.baseURL, { title: 'Friday job', due: '2026-09-25' });
    await addTask(page, server.baseURL, { title: 'Finished job', due: '2026-09-21', stage: 'done' });
    await addTask(page, server.baseURL, { title: 'Undated job' });

    await addEvent(page, server.baseURL, { title: 'Exams', start_date: '2026-09-21', notes: 'Print exam papers today' });
    await addEvent(page, server.baseURL, {
      title: 'Camp', start_date: '2026-09-17', end_date: '2026-09-22',
    });
    await addEvent(page, server.baseURL, {
      title: 'Card campaign', start_date: '2026-10-31', notice_amount: '6', notice_unit: 'weeks',
    });
    await addEvent(page, server.baseURL, { title: 'Far away', start_date: '2026-12-05', notice_amount: '2', notice_unit: 'weeks' });

    await openBriefing(page, server.baseURL);

    // 3.02
    await expect(page.locator('h1')).toHaveText("Saturday 19 September — today's briefing");

    // 3.03: due on or before Fri 25 Sep, overdue included and stamped,
    // earliest first, each with its people; Sat 26 Sep and finished
    // and undated tasks are not listed.
    const mustDo = section(page, 'Must be done today');
    await expect(mustDo.locator('.briefing-card-title')).toHaveText(['Old job', 'Wednesday job', 'Friday job']);
    await expect(card(page, 'Must be done today', 'Old job').locator('.stamp').first()).toHaveText('OVERDUE');
    await expect(card(page, 'Must be done today', 'Wednesday job').locator('.stamp').first()).toHaveText('WED 23 SEP');
    await expect(card(page, 'Must be done today', 'Wednesday job').locator('.briefing-people .av')).toHaveAttribute('title', me); // people are circles named by their title (gates 4.08, 4.46)

    // 3.05: YOURS on the task assigned to me, and Just mine shows only it.
    await expect(card(page, 'Must be done today', 'Wednesday job').locator('.stamp-yours')).toHaveText('YOURS');
    await expect(mustDo.locator('.stamp-yours')).toHaveCount(1);
    await page.click('text=Just mine');
    await ready(page);
    await expect(section(page, 'Must be done today').locator('.briefing-card-title')).toHaveText(['Wednesday job']);
    await page.click('text=Everyone');
    await ready(page);
    await expect(section(page, 'Must be done today').locator('.briefing-card')).toHaveCount(3);

    // 3.06: This week, in date order, with notes visible.
    const week = section(page, 'This week');
    await expect(week.locator('.briefing-card-title')).toHaveText(['Camp', 'Exams']);
    await expect(card(page, 'This week', 'Exams').locator('.stamp').first()).toHaveText('MON 21 SEP');
    await expect(card(page, 'This week', 'Exams')).toContainText('Print exam papers today');

    // 3.09: a multi-day event already under way shows "until <date>".
    await expect(card(page, 'This week', 'Camp')).toContainText('until Tue 22 Sep');

    // 3.07: the campaign's 6 weeks' notice has started; a later event's hasn't.
    const coming = section(page, 'Coming up');
    await expect(coming.locator('.briefing-card-title')).toHaveText(['Card campaign']);
    await expect(card(page, 'Coming up', 'Card campaign').locator('.stamp')).toHaveText('IN 6 WEEKS');
    await expect(card(page, 'Coming up', 'Card campaign')).toContainText('Sat 31 Oct 2026');

    // 3.11: every item links to its task or event.
    await card(page, 'Must be done today', 'Friday job').getByRole('link').click();
    await ready(page);
    expect(new URL(page.url()).pathname).toMatch(/^\/tasks\/\d+$/);
    await expect(page.locator('h1')).toHaveText('Friday job');

    await openBriefing(page, server.baseURL);
    await card(page, 'This week', 'Exams').getByRole('link').click();
    await ready(page);
    expect(new URL(page.url()).pathname).toMatch(/^\/calendar\/events\/\d+$/);
    await expect(page.locator('h1')).toHaveText('Exams');

    await openBriefing(page, server.baseURL);
    await card(page, 'Coming up', 'Card campaign').getByRole('link').click();
    await ready(page);
    expect(new URL(page.url()).pathname).toMatch(/^\/calendar\/events\/\d+$/);
    await expect(page.locator('h1')).toHaveText('Card campaign');
  });

  // SPEC gate 3.10: empty sections say something friendly, not nothing.
  test('@fresh gate 3.10: empty sections show a friendly message', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/');
    await openBriefing(page, server.baseURL);

    await expect(section(page, 'Must be done today')).toContainText('Nothing due before next Saturday.');
    await expect(section(page, 'This week')).toContainText('Nothing on the calendar this week.');
    await expect(section(page, 'Coming up')).toContainText('Nothing coming up yet.');
    await expect(page.locator('.briefing-card')).toHaveCount(0);

    // Just mine with tasks due but none of mine also says so kindly.
    await addTask(page, server.baseURL, { title: 'Somebody else', due: '2026-09-20' });
    await openBriefing(page, server.baseURL, '?mine=1');
    await expect(section(page, 'Must be done today')).toContainText('Nothing of yours is due before next Saturday.');
  });

  // Layers 5 and 6 (SPEC B7): populated as well as empty.
  test('@fresh gate 3.13: a populated Briefing passes axe and has no sideways scroll', async ({ page, server }) => {
    const me = await signInAsNewPerson(page, server.baseURL, '/');
    await addTask(page, server.baseURL, { title: 'Wednesday job', due: '2026-09-23', people: [me], stage: 'doing' });
    await addTask(page, server.baseURL, { title: 'Old job', due: '2026-09-10' });
    await addEvent(page, server.baseURL, { title: 'Exams', start_date: '2026-09-21', notes: 'Print exam papers today' });
    await addEvent(page, server.baseURL, { title: 'Card campaign', start_date: '2026-10-31', notice_amount: '6', notice_unit: 'weeks' });
    await openBriefing(page, server.baseURL);
    await expect(page.locator('.briefing-card')).toHaveCount(4);

    await axeCheck(page);
    for (const [width, height] of [[1024, 700], [1920, 1080]] as const) {
      await page.setViewportSize({ width, height });
      await expectNoSideScroll(page);
    }
  });
});

test.describe('on a Wednesday (16 Sep 2026)', () => {
  test.use({ today: '2026-09-16' });

  // SPEC gates 3.02 (other days) and 3.04.
  test('@fresh gates 3.02 and 3.04: the coming Saturday, and "Must be done this Saturday"', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/');
    await addTask(page, server.baseURL, { title: 'Due Friday', due: '2026-09-25' });
    await addTask(page, server.baseURL, { title: 'Due Saturday', due: '2026-09-26' });
    await addTask(page, server.baseURL, { title: 'Due today', due: '2026-09-16' });
    await openBriefing(page, server.baseURL);

    await expect(page.locator('h1')).toHaveText('Briefing for Saturday 19 September');
    const mustDo = section(page, 'Must be done this Saturday');
    await expect(mustDo.locator('.briefing-card-title')).toHaveText(['Due today', 'Due Friday']);
    await expect(card(page, 'Must be done this Saturday', 'Due today').locator('.stamp').first()).toHaveText('TODAY');
  });

  // SPEC gate 3.08, driven the way a person would: click through from
  // the Briefing, change just one occurrence, and come back.
  test('@fresh gate 3.08: a cancelled occurrence never appears and a moved one appears at its new date', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/');
    await addEvent(page, server.baseURL, { title: 'Staff meeting', start_date: '2026-09-14', recurrence: 'weekly' });
    await addEvent(page, server.baseURL, { title: 'Concert', start_date: '2026-09-24', recurrence: 'yearly' });

    await openBriefing(page, server.baseURL);
    await expect(section(page, 'This week').locator('.briefing-card-title')).toHaveText(['Staff meeting', 'Concert']);
    await expect(card(page, 'This week', 'Staff meeting').locator('.stamp')).toHaveText('MON 21 SEP');

    // Cancel just Monday 21 Sep's meeting.
    await card(page, 'This week', 'Staff meeting').getByRole('link').click();
    await ready(page);
    await page.click('text=Change just this one');
    await ready(page);
    await page.click('text=Cancel just this one');
    await ready(page);

    // Move just this year's concert from Thu 24 Sep to Fri 25 Sep.
    await openBriefing(page, server.baseURL);
    await card(page, 'This week', 'Concert').getByRole('link').click();
    await ready(page);
    await page.click('text=Change just this one');
    await ready(page);
    await page.fill('#occurrence-start-date', '2026-09-25');
    await page.locator('.calendar-event-form button[type="submit"]').click();
    await ready(page);

    await openBriefing(page, server.baseURL);
    await expect(section(page, 'This week').locator('.briefing-card-title')).toHaveText(['Concert']);
    await expect(card(page, 'This week', 'Concert').locator('.stamp')).toHaveText('FRI 25 SEP');
  });
});

test.describe('the Saturday before (12 Sep 2026)', () => {
  test.use({ today: '2026-09-12' });

  // SPEC gate 3.07's other half: the same campaign isn't shown yet.
  test('@fresh gate 3.07: the card campaign is not on the 12 September briefing', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/');
    await addEvent(page, server.baseURL, { title: 'Card campaign', start_date: '2026-10-31', notice_amount: '6', notice_unit: 'weeks' });
    await openBriefing(page, server.baseURL);

    await expect(page.locator('h1')).toHaveText("Saturday 12 September — today's briefing");
    await expect(section(page, 'Coming up')).toContainText('Nothing coming up yet.');
    await expect(page.locator('.briefing-card', { hasText: 'Card campaign' })).toHaveCount(0);
  });
});
