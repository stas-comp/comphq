import { expect, test } from '@playwright/test';
import { newDataDir, startServerAt, type RunningServer } from '../helpers/server-at';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';
import { exportCSV } from '../helpers/zip';

type Page = import('@playwright/test').Page;

// SPEC gates 8.40-8.50, 8.52 end to end (B14.4, D-108). Each test has a server of
// its own with a fixed "today", and lets the calendar move on by starting the app
// again on the same data with a later date (as weekly.spec.ts does): the yearly
// pass happens the first time Comp HQ is used on or after the day (gate 8.47).
// Tagged @fresh because each needs a private server. 2026-10-01 is a Thursday.

test.describe.configure({ timeout: 240_000 });

const card = (page: Page, title: string) => page.locator('.task-card', { hasText: title });

async function makeYearly(
  page: Page,
  base: string,
  title: string,
  opts: { due: string; weeks?: string; me?: string; stage?: string },
) {
  await page.goto(base + '/tasks/new');
  await ready(page);
  await page.fill('#new-task-title', title);
  if (opts.stage) await page.selectOption('#new-task-stage', opts.stage);
  if (opts.me) await page.selectOption('#new-task-people', { label: opts.me });
  await page.fill('#new-task-due-date', opts.due);
  await page.selectOption('#new-task-repeat', 'yearly');
  if (opts.weeks) await page.fill('#new-task-notice', opts.weeks);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

async function onBoard(page: Page, base: string, title: string): Promise<boolean> {
  await page.goto(base + '/tasks/board?q=' + encodeURIComponent(title));
  await ready(page);
  return (await card(page, title).count()) > 0;
}

async function inMyJobsOrGrabs(page: Page, base: string, title: string): Promise<boolean> {
  await page.goto(base + '/tasks');
  await ready(page);
  return (await page.locator('li.team-task', { hasText: title }).count()) > 0;
}

async function onTeam(page: Page, base: string, title: string): Promise<boolean> {
  await page.goto(base + '/tasks/team');
  await ready(page);
  return (await page.locator('.team-task', { hasText: title }).count()) > 0;
}

test('@fresh gates 8.40, 8.41: Repeats is a choice (No, Every week, Every year); the weeks box shows only for yearly, defaults to 6, and a yearly job needs a due date', async ({ page }) => {
  const dataDir = newDataDir('yearly-form');
  const server = await startServerAt(dataDir, '2026-10-01');
  try {
    const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    void me;
    // In the Add task window.
    await page.goto(server.baseURL + '/tasks/board');
    await ready(page);
    await page.click('#add-task-link');
    const win = page.locator('#task-window');
    await expect(win).toBeVisible();
    await expect(win.getByLabel('Repeats')).toHaveValue('none');
    await expect(win.locator('option')).toContainText(['No', 'Every week (back in To do each Saturday)', 'Every year']);
    await expect(win.locator('[data-yearly-only]')).toBeHidden();
    await win.getByLabel('Repeats').selectOption('yearly');
    await expect(win.locator('[data-yearly-only]')).toBeVisible();
    await expect(win.locator('#new-task-notice')).toHaveValue('6');
    await expect(win.getByText("weeks before it's due")).toBeVisible();
    await win.getByLabel('Repeats').selectOption('weekly');
    await expect(win.locator('[data-yearly-only]')).toBeHidden();

    // No due date: refused in plain words, the form kept as typed.
    await win.getByLabel('Title').fill('Renew the licence');
    await win.getByLabel('Repeats').selectOption('yearly');
    await win.locator('#new-task-notice').fill('9');
    await win.getByRole('button', { name: 'Add task' }).click();
    await expect(win.locator('[role="alert"]').first()).toHaveText('A yearly job needs a due date.');
    await expect(win.getByLabel('Repeats')).toHaveValue('yearly');
    await expect(win.locator('#new-task-notice')).toHaveValue('9');
    await expect(win.locator('[data-yearly-only]')).toBeVisible();
    // A notice outside 1 to 52 weeks: the box itself stops it in a browser, and the
    // server says the same in plain words when the form is edited past that.
    await expect(win.locator('#new-task-notice')).toHaveAttribute('min', '1');
    await expect(win.locator('#new-task-notice')).toHaveAttribute('max', '52');
    const refused = await page.request.post(server.baseURL + '/tasks', {
      data: new URLSearchParams([['title', 'Too long a notice'], ['due_date', '01/12/2026'], ['repeat', 'yearly'], ['notice_weeks', '60'], ['fragment', '1']]).toString(),
      headers: { origin: server.baseURL, 'content-type': 'application/x-www-form-urlencoded' },
    });
    expect(refused.status()).toBe(422);
    expect(await refused.text()).toContain('The notice has to be between 1 and 52 weeks.');
    // The page version of the form has the same choice.
    await page.goto(server.baseURL + '/tasks/new');
    await ready(page);
    await expect(page.getByLabel('Repeats')).toHaveValue('none');
    await axeCheck(page);
    await expectNoSideScroll(page);
  } finally {
    await server.stop();
  }
});

test('@fresh gate 8.40: with no script the weeks box is always there, with its hint', async ({ browser }) => {
  const dataDir = newDataDir('yearly-nojs');
  const server = await startServerAt(dataDir, '2026-10-01');
  const normal = await browser.newContext();
  const p = await normal.newPage();
  try {
    await signInAsNewPerson(p, server.baseURL, '/tasks/board');
    const context = await browser.newContext({ javaScriptEnabled: false });
    await context.addCookies(await normal.cookies());
    const page = await context.newPage();
    await page.goto(server.baseURL + '/tasks/new');
    await expect(page.getByLabel('Repeats')).toHaveValue('none');
    await expect(page.locator('#new-task-notice')).toBeVisible();
    await expect(page.getByText('Only used when Repeats is Every year.')).toBeVisible();
    const title = uniqueName('No script yearly');
    await page.getByLabel('Title').fill(title);
    await page.fill('#new-task-due-date', '10/12/2026');
    await page.selectOption('#new-task-repeat', 'yearly');
    await page.click('.add-task-form button[type="submit"]');
    // It is waiting, so the page says when it will appear instead of showing the Board.
    await expect(page).toHaveURL(/\/tasks\/yearly\?saved=\d+/);
    await expect(page.locator('.message[role="status"]')).toHaveText('Saved. It will appear in To do on Thu 29 Oct 2026.');
    await context.close();
  } finally {
    await normal.close();
    await server.stop();
  }
});

test('@fresh gates 8.42-8.45, 8.47, 8.49: a yearly job waits out of sight, shows on its show date stamped YEARLY, and is put away 15 days after it is finished', async ({ page }) => {
  const dataDir = newDataDir('yearly-life');
  let server: RunningServer = await startServerAt(dataDir, '2026-10-01');
  try {
    const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    const title = uniqueName('Renew the licence');

    // Due in 10 weeks (10 Dec), with 6 weeks' notice: it shows from 29 Oct.
    await makeYearly(page, server.baseURL, title, { due: '10/12/2026', weeks: '6' });
    await expect(page.locator('.message[role="status"]')).toHaveText('Saved. It will appear in To do on Thu 29 Oct 2026.');
    // It is on the Yearly jobs page, with its due date, notice and "Appears Thu 29 Oct", stamped YEARLY.
    const line = page.locator('.yearly-job', { hasText: title });
    await expect(line).toBeVisible();
    await expect(line).toContainText('Due Thu 10 Dec 2026');
    await expect(line).toContainText('6 weeks');
    await expect(line.locator('.yearly-status')).toHaveText('Appears Thu 29 Oct');
    await expect(line.locator('.stamp.yearly')).toHaveText('YEARLY');

    // And nowhere else: not the Board, My jobs, Up for grabs, Team.
    expect(await onBoard(page, server.baseURL, title)).toBe(false);
    expect(await inMyJobsOrGrabs(page, server.baseURL, title)).toBe(false);
    expect(await onTeam(page, server.baseURL, title)).toBe(false);

    // The day before: still waiting. On the show date: in Up for grabs (nobody is on it).
    await server.stop();
    server = await startServerAt(dataDir, '2026-10-28');
    expect(await onBoard(page, server.baseURL, title)).toBe(false);
    await server.stop();
    server = await startServerAt(dataDir, '2026-10-29');
    expect(await inMyJobsOrGrabs(page, server.baseURL, title)).toBe(true);
    await expect(page.locator('.myjobs-grabs-list li', { hasText: title }).locator('.stamp.yearly')).toHaveText('YEARLY');
    expect(await onBoard(page, server.baseURL, title)).toBe(true);
    await expect(card(page, title).locator('.stamp.yearly')).toHaveText('YEARLY');
    await expect(page.locator('.task-column[data-stage="todo"] .task-card', { hasText: title })).toHaveCount(1);
    expect(await onTeam(page, server.baseURL, title)).toBe(true);
    await expect(page.locator('.team-task', { hasText: title }).locator('.stamp.yearly').first()).toHaveText('YEARLY');
    // History says what happened.
    await page.goto(server.baseURL + '/tasks/board?q=' + encodeURIComponent(title));
    const href = (await card(page, title).locator('.task-card-title a').first().getAttribute('href'))!;
    await page.goto(server.baseURL + href);
    await ready(page);
    const history = await page.locator('.task-activity-list li').allTextContents();
    expect(history.join('\n')).toContain('Comp HQ put this in To do, due Thu 10 Dec 2026 (yearly)');
    expect(history.join('\n')).not.toContain('Comp HQ ' + me);

    // Someone takes it, adds a step and ticks it, and finishes it.
    await page.goto(server.baseURL + '/tasks');
    await ready(page);
    await page.locator('.myjobs-grabs-list li', { hasText: title }).getByRole('button', { name: 'Take it' }).click();
    await ready(page);
    await page.goto(server.baseURL + href);
    await ready(page);
    await page.fill('#step-add-' + href.split('/').pop(), 'Send the form');
    await page.click('.step-add button[type="submit"]');
    await ready(page);
    await page.locator('.step input[type="checkbox"]').check();
    await expect(page.locator('.step.done')).toHaveCount(1);
    await page.goto(server.baseURL + '/tasks/board?person=mine');
    await ready(page);
    const c = card(page, title);
    await c.locator('.task-move > summary').click();
    await c.locator('.task-move-to-form select').selectOption('done');
    await c.locator('.task-move-to-form button').click();
    await ready(page);
    await expect(page.locator('.task-column[data-stage="done"] .task-card', { hasText: title })).toHaveCount(1);

    // 14 days later it is still in Done; on the 15th it is put away until next year.
    await server.stop();
    server = await startServerAt(dataDir, '2026-11-12');
    expect(await onBoard(page, server.baseURL, title)).toBe(true);
    await server.stop();
    server = await startServerAt(dataDir, '2026-11-13');
    expect(await onBoard(page, server.baseURL, title)).toBe(false);
    await page.goto(server.baseURL + '/tasks/yearly');
    await ready(page);
    const next = page.locator('.yearly-job', { hasText: title });
    await expect(next).toContainText('Due Fri 10 Dec 2027'); // a year on from the old due date, not from the day it was finished
    await expect(next.locator('.yearly-status')).toHaveText('Appears Fri 29 Oct');
    await expect(next.locator('.stamp.yearly')).toBeVisible();
    // It is never in Finished tasks, and its steps are unticked, and the people stay.
    await page.goto(server.baseURL + '/tasks/finished');
    await ready(page);
    await expect(page.locator('.task-card', { hasText: title })).toHaveCount(0);
    await page.goto(server.baseURL + href);
    await ready(page);
    await expect(page.locator('.step.done')).toHaveCount(0);
    await expect(page.locator('#details-people option:checked')).toHaveText(me);
    const after = (await page.locator('.task-activity-list li').allTextContents()).join('\n');
    expect(after).toContain('Comp HQ put this away until Fri 29 Oct 2027 (yearly)');
  } finally {
    await server.stop();
  }
});

test('@fresh gates 8.43, 8.48, 8.50: the Yearly jobs page lists every yearly job in due-date order and opens each in the window; changing the notice shows a waiting job at once; switching to No makes it ordinary; search marks waiting jobs', async ({ page }) => {
  const dataDir = newDataDir('yearly-edit');
  const server = await startServerAt(dataDir, '2026-10-01');
  try {
    const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    const later = uniqueName('Later job');
    const sooner = uniqueName('Sooner job');
    await makeYearly(page, server.baseURL, later, { due: '10/03/2027', weeks: '4', me });
    await makeYearly(page, server.baseURL, sooner, { due: '10/12/2026', weeks: '6' });

    await page.goto(server.baseURL + '/tasks/yearly');
    await ready(page);
    const titles = await page.locator('.yearly-job .task-card-title').allTextContents();
    expect(titles.filter((t) => t === sooner || t === later)).toEqual([sooner, later]); // order of due date
    await expect(page.locator('.yearly-job', { hasText: later })).toContainText('4 weeks');
    await expect(page.locator('.yearly-job', { hasText: later }).locator('.task-card-assignees')).toBeVisible(); // its people
    await axeCheck(page);
    await expectNoSideScroll(page);
    // The page is linked from the Board and My jobs, beside Finished tasks and Removed tasks.
    for (const path of ['/tasks/board', '/tasks']) {
      await page.goto(server.baseURL + path);
      await ready(page);
      await expect(page.getByRole('link', { name: 'Yearly jobs' })).toHaveAttribute('href', '/tasks/yearly');
    }

    // Job search finds a waiting job, marked.
    await page.goto(server.baseURL + '/tasks/board');
    await ready(page);
    await page.locator('#search-box').fill(sooner);
    await expect(page.locator('.search-panel .search-result', { hasText: sooner })).toContainText('Yearly · appears Thu 29 Oct');
    await page.goto(server.baseURL + '/tasks/search?q=' + encodeURIComponent(later));
    await expect(page.locator('.search-result', { hasText: later })).toContainText('Yearly · appears Wed 10 Feb');

    // Open it in the window from the Yearly jobs page, and give it a long notice: it shows at once.
    await page.goto(server.baseURL + '/tasks/yearly');
    await ready(page);
    await page.locator('.yearly-job', { hasText: sooner }).locator('.task-card-title a').click();
    const win = page.locator('#task-window');
    await expect(win).toBeVisible();
    await win.getByRole('button', { name: 'Edit' }).click();
    await expect(win.getByLabel('Repeats')).toHaveValue('yearly');
    await expect(win.locator('#details-notice')).toHaveValue('6');
    await win.locator('#details-notice').fill('52');
    await win.getByRole('button', { name: 'Save' }).click();
    await expect(win.locator('.task-read')).toBeVisible(); // back to reading
    await page.keyboard.press('Escape');
    expect(await onBoard(page, server.baseURL, sooner)).toBe(true); // shows at once

    // Back to No: an ordinary job, in To do at once (and no longer on the Yearly jobs page).
    await page.goto(server.baseURL + '/tasks/yearly');
    await ready(page);
    await page.locator('.yearly-job', { hasText: later }).locator('.task-card-title a').click();
    await win.getByRole('button', { name: 'Edit' }).click();
    await win.getByLabel('Repeats').selectOption('none');
    await win.getByRole('button', { name: 'Save' }).click();
    await expect(win.locator('.task-read')).toBeVisible();
    await page.keyboard.press('Escape');
    expect(await onBoard(page, server.baseURL, later)).toBe(true);
    await expect(card(page, later).locator('.stamp.yearly')).toHaveCount(0);
    await page.goto(server.baseURL + '/tasks/yearly');
    await ready(page);
    await expect(page.locator('.yearly-job', { hasText: later })).toHaveCount(0);

    // Saving a waiting job from the window says when it will appear (a message at the foot).
    const waiting = uniqueName('Waits in the window');
    await makeYearly(page, server.baseURL, waiting, { due: '15/01/2027', weeks: '5' });
    await page.goto(server.baseURL + '/tasks/yearly');
    await ready(page);
    await page.locator('.yearly-job', { hasText: waiting }).locator('.task-card-title a').click();
    await win.getByRole('button', { name: 'Edit' }).click();
    await win.getByRole('button', { name: 'Save' }).click();
    await expect(page.locator('#toast-region .toast-text')).toHaveText('Saved. It will appear in To do on Fri 11 Dec 2026.');
  } finally {
    await server.stop();
  }
});

test('@fresh gate 8.49: a yearly job that is showing has a YEARLY stamp in the Briefing, and the Calendar shows a waiting job on its due date', async ({ page }) => {
  const dataDir = newDataDir('yearly-briefing');
  const server = await startServerAt(dataDir, '2026-10-03'); // a Saturday
  try {
    const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    const showing = uniqueName('Due this week');
    const waiting = uniqueName('Due in December');
    await makeYearly(page, server.baseURL, showing, { due: '09/10/2026', weeks: '8', me });
    await makeYearly(page, server.baseURL, waiting, { due: '10/12/2026', weeks: '6', me });
    await page.goto(server.baseURL + '/briefing');
    await ready(page);
    await expect(page.locator('.card, li', { hasText: showing }).locator('.stamp.yearly').first()).toHaveText('YEARLY');
    await expect(page.getByText(waiting)).toHaveCount(0); // waiting: not in the Briefing

    // The Calendar shows a waiting job's due date like any job's.
    await page.goto(server.baseURL + '/calendar?month=2026-12');
    await ready(page);
    await expect(page.locator('.calendar-day[data-date="2026-12-10"] .calendar-chip', { hasText: waiting })).toHaveCount(1);
  } finally {
    await server.stop();
  }
});

test('@fresh gate 8.52: Export everything shows yearly in Repeats and a Notice (weeks) column', async ({ page }) => {
  const dataDir = newDataDir('yearly-export');
  const server = await startServerAt(dataDir, '2026-10-01');
  try {
    await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    const title = uniqueName('Exported yearly');
    const plain = uniqueName('Exported plain');
    await makeYearly(page, server.baseURL, title, { due: '10/12/2026', weeks: '8' });
    await page.goto(server.baseURL + '/tasks/new');
    await page.fill('#new-task-title', plain);
    await page.click('.add-task-form button[type="submit"]');
    await ready(page);
    const rows = await exportCSV(page, server.baseURL, 'tasks.csv');
    const header = rows[0];
    expect(header.slice(-2)).toEqual(['Repeats', 'Notice (weeks)']);
    const yearly = rows.find((r) => r[0] === title)!;
    expect(yearly[header.indexOf('Repeats')]).toBe('yearly');
    expect(yearly[header.indexOf('Notice (weeks)')]).toBe('8');
    const ordinary = rows.find((r) => r[0] === plain)!;
    expect(ordinary[header.indexOf('Repeats')]).toBe('');
    expect(ordinary[header.indexOf('Notice (weeks)')]).toBe('');
  } finally {
    await server.stop();
  }
});
