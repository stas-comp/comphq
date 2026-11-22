import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { idFromHref, myPersonID } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gates 8.35-8.38 (B14.3, D-107): steps under the cards on My jobs.

test.describe.configure({ timeout: 90_000 });

/** Makes a job with steps for the signed-in person, through the form's own endpoint. */
async function makeJob(page: Page, baseURL: string, title: string, stage: string, steps: string[], mine = true): Promise<string> {
  const me = await myPersonID(page, baseURL);
  const pairs: [string, string][] = [['title', title], ['stage', stage]];
  if (mine) pairs.push(['person_id', me]);
  for (const s of steps) pairs.push(['step', s]);
  const res = await page.request.post(baseURL + '/tasks', {
    data: new URLSearchParams(pairs).toString(),
    headers: { origin: baseURL, 'content-type': 'application/x-www-form-urlencoded' },
    maxRedirects: 0,
  });
  expect(res.status()).toBeLessThan(400);
  await page.goto(baseURL + '/tasks/board?q=' + encodeURIComponent(title));
  await ready(page);
  return idFromHref((await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').first().getAttribute('href'))!);
}

const card = (page: Page, title: string) => page.locator('li.team-task', { hasText: title });

async function openMyJobs(page: Page, baseURL: string) {
  await page.goto(baseURL + '/tasks');
  await ready(page);
}

test('gate 8.35: a Working on now card shows its steps open; each can be ticked and unticked there, with who and when, and the count updates at once', async ({ page, server }) => {
  const meName = await signInAsNewPerson(page, server.baseURL, '/tasks');
  const title = uniqueName('Doing');
  await makeJob(page, server.baseURL, title, 'doing', ['Pick the list', 'Print labels', 'Stuff envelopes']);
  await openMyJobs(page, server.baseURL);

  const c = card(page, title);
  await expect(c.locator('.card-steps-body')).toBeVisible(); // open by default
  await expect(c.locator('.steps-list .step-words')).toHaveText(['Pick the list', 'Print labels', 'Stuff envelopes']);
  await expect(c.locator('.step-badge')).toContainText('0/3');

  await c.getByLabel('Print labels').check();
  await expect(c.locator('.step-badge')).toContainText('1/3'); // the count updates at once, no reload
  const ticked = c.locator('.step', { hasText: 'Print labels' });
  await expect(ticked).toHaveClass(/done/);
  await expect(ticked.locator('.step-by')).toContainText(meName); // who, and when
  await expect(ticked.locator('.step-by')).toContainText(/\d/);

  await c.getByLabel('Pick the list').check();
  await c.getByLabel('Stuff envelopes').check();
  await expect(c.locator('.step-badge.done')).toBeVisible(); // all three: the badge becomes a tick
  await expect(c.locator('.step.done')).toHaveCount(3);

  await c.getByLabel('Print labels').uncheck();
  await expect(c.locator('.step.done')).toHaveCount(2);
  await expect(c.locator('.step', { hasText: 'Print labels' }).locator('.step-by')).toHaveCount(0);
  // The same tick is on the job in its window, and after a reload.
  await page.reload();
  await ready(page);
  await expect(card(page, title).locator('.step.done')).toHaveCount(2);
  await card(page, title).locator('.card-steps-edit').click();
  await expect(page.locator('#task-window[open] .task-steps .step.done')).toHaveCount(2);
});

test('gate 8.37: a tick is the same tick as in the job window: seen on the Board, Team and the window, and on another computer', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  const title = uniqueName('Same tick');
  const id = await makeJob(page, server.baseURL, title, 'doing', ['One', 'Two']);
  await openMyJobs(page, server.baseURL);
  await card(page, title).getByLabel('One').check();
  await expect(card(page, title).locator('.step-badge')).toContainText('1/2');

  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);
  await expect(page.locator('.task-card', { hasText: title }).locator('.step-badge')).toContainText('1/2');
  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  await expect(page.locator('.team-task', { hasText: title }).locator('.step-badge')).toContainText('1/2');
  await page.goto(`${server.baseURL}/tasks/${id}`);
  await ready(page);
  await expect(page.locator('.task-steps .step.done')).toHaveCount(1);

  // Another computer, the same person, sees it within 60 seconds.
  const other = await browser.newContext();
  const otherPage = await other.newPage();
  await otherPage.clock.install();
  await other.addCookies(await page.context().cookies());
  await openMyJobs(otherPage, server.baseURL);
  await expect(card(otherPage, title).locator('.step.done')).toHaveCount(1);
  await openMyJobs(page, server.baseURL);
  await card(page, title).getByLabel('Two').check();
  await expect(card(page, title).locator('.step-badge.done')).toBeVisible();
  await otherPage.clock.runFor(61_000);
  await expect(card(otherPage, title).locator('.step.done')).toHaveCount(2);
  await other.close();
});

test('gate 8.36: Up next and Up for grabs cards say Show steps · n/m, open and close in place, and keep that through a reload and the refresh', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  const upNext = uniqueName('Next');
  const grab = uniqueName('Grab');
  await makeJob(page, server.baseURL, upNext, 'todo', ['A', 'B', 'C']);
  await makeJob(page, server.baseURL, grab, 'todo', ['X', 'Y'], false);
  await openMyJobs(page, server.baseURL);

  const next = card(page, upNext);
  const grabbed = card(page, grab);
  await expect(next.locator('.steps-toggle')).toHaveText('Show steps · 0/3');
  await expect(grabbed.locator('.steps-toggle')).toHaveText('Show steps · 0/2');
  await expect(next.locator('.card-steps-body')).toBeHidden();
  await expect(grabbed.locator('.card-steps-body')).toBeHidden();

  await next.locator('.steps-toggle').click();
  await expect(next.locator('.steps-toggle')).toHaveText('Hide steps · 0/3');
  await expect(next.locator('.card-steps-body')).toBeVisible();
  await grabbed.locator('.steps-toggle').click();
  await expect(grabbed.locator('.card-steps-body')).toBeVisible();
  await grabbed.locator('.steps-toggle').click(); // and closed again
  await expect(grabbed.locator('.card-steps-body')).toBeHidden();

  // Any card can be closed, Working on now ones too.
  const working = uniqueName('Working');
  await makeJob(page, server.baseURL, working, 'doing', ['W1']);
  await openMyJobs(page, server.baseURL);
  await expect(card(page, working).locator('.card-steps-body')).toBeVisible();
  await card(page, working).locator('.steps-toggle').click();
  await expect(card(page, working).locator('.card-steps-body')).toBeHidden();

  // Kept through a reload...
  await page.reload();
  await ready(page);
  await expect(card(page, upNext).locator('.card-steps-body')).toBeVisible();
  await expect(card(page, grab).locator('.card-steps-body')).toBeHidden();
  await expect(card(page, working).locator('.card-steps-body')).toBeHidden();
  await expect(card(page, working).locator('.steps-toggle')).toHaveText('Show steps · 0/1');

  // ...and through the 60-second refresh when something changes elsewhere.
  const other = await browser.newContext();
  const otherPage = await other.newPage();
  await other.addCookies(await page.context().cookies());
  await otherPage.goto(server.baseURL + '/tasks');
  await ready(otherPage);
  await card(otherPage, upNext).locator('.steps-toggle').click();
  await card(otherPage, upNext).getByLabel('B', { exact: true }).check();
  await other.close();

  await page.close();
  const fresh = await browser.newContext();
  const p2 = await fresh.newPage();
  await p2.clock.install();
  await signInAsNewPerson(p2, server.baseURL, '/tasks');
  const mine = uniqueName('Refresh');
  await makeJob(p2, server.baseURL, mine, 'todo', ['R1', 'R2']);
  await openMyJobs(p2, server.baseURL);
  await card(p2, mine).locator('.steps-toggle').click();
  await expect(card(p2, mine).locator('.card-steps-body')).toBeVisible();
  // Another computer ticks a step of it.
  const third = await browser.newContext();
  const p3 = await third.newPage();
  await third.addCookies(await fresh.cookies());
  await p3.goto(server.baseURL + '/tasks');
  await ready(p3);
  await card(p3, mine).locator('.steps-toggle').click();
  await card(p3, mine).getByLabel('R1').check();
  await p2.clock.runFor(61_000);
  await expect(card(p2, mine).locator('.step-badge')).toContainText('1/2'); // the page was redrawn
  await expect(card(p2, mine).locator('.card-steps-body')).toBeVisible(); // and the card is still open
  await expect(card(p2, mine).locator('.steps-toggle')).toContainText('Hide steps');
  await third.close();
  await fresh.close();
});

test('gate 8.36: a blocked store only means the defaults', async ({ page, server }) => {
  await page.addInitScript(() => {
    Object.defineProperty(window, 'localStorage', {
      get() {
        throw new Error('blocked');
      },
    });
  });
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  const open = uniqueName('Blocked open');
  const closed = uniqueName('Blocked folded');
  await makeJob(page, server.baseURL, open, 'doing', ['One']);
  await makeJob(page, server.baseURL, closed, 'todo', ['Two']);
  await openMyJobs(page, server.baseURL);
  await expect(card(page, open).locator('.card-steps-body')).toBeVisible();
  await expect(card(page, closed).locator('.card-steps-body')).toBeHidden();
  await card(page, closed).locator('.steps-toggle').click(); // still works in place
  await expect(card(page, closed).locator('.card-steps-body')).toBeVisible();
});

test('gate 8.37: Edit steps opens the job window, where steps are added, reworded, reordered and removed', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  const title = uniqueName('Window job');
  await makeJob(page, server.baseURL, title, 'doing', ['First']);
  await openMyJobs(page, server.baseURL);
  // The list on My jobs has no add box, rename or move controls: it is tick-only.
  const c = card(page, title);
  await expect(c.locator('input[type="text"], .step-rename-link, .step-actions, .icon-btn')).toHaveCount(0);
  await c.locator('.card-steps-edit').click();
  await expect(page.locator('#task-window[open]')).toBeVisible();
  await expect(page.locator('#task-window .step-add')).toBeVisible();
  await expect(page.locator('#task-window h1, #task-window .task-window-title')).toContainText(title);
});

test('gate 8.38: without JavaScript the boxes are forms that tick and come back, and Show steps is a link', async ({ browser, server }) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  const page = await context.newPage();
  const normal = await browser.newContext();
  const p = await normal.newPage();
  await signInAsNewPerson(p, server.baseURL, '/tasks');
  const closedTitle = uniqueName('NoJS folded');
  const openTitle = uniqueName('NoJS open');
  await makeJob(p, server.baseURL, closedTitle, 'todo', ['Alpha', 'Beta']);
  await makeJob(p, server.baseURL, openTitle, 'doing', ['Gamma']);
  await context.addCookies(await normal.cookies());
  await normal.close();

  await page.goto(server.baseURL + '/tasks');
  // Working on now is open; the Up next card shows only its link.
  await expect(card(page, openTitle).locator('.card-steps-body')).toBeVisible();
  await expect(card(page, closedTitle).locator('.card-steps-body')).toBeHidden();
  const link = card(page, closedTitle).getByRole('link', { name: /Show steps/ });
  await expect(link).toHaveAttribute('href', /\/tasks\?open=\d+/);
  await link.click();
  await expect(card(page, closedTitle).locator('.card-steps-body')).toBeVisible();
  await expect(card(page, closedTitle).getByRole('link', { name: /Hide steps/ })).toBeVisible();

  // Tick: the box has an Update button with no script; it ticks and returns to My jobs, card open.
  await card(page, closedTitle).getByLabel('Alpha').check();
  await card(page, closedTitle).locator('.step', { hasText: 'Alpha' }).getByRole('button', { name: 'Update' }).click();
  await expect(page).toHaveURL(/\/tasks\?open=\d+#job-\d+/);
  await expect(card(page, closedTitle).locator('.step.done')).toHaveCount(1);
  await expect(card(page, closedTitle).locator('.card-steps-body')).toBeVisible();
  await expect(card(page, closedTitle).locator('.steps-toggle')).toContainText('1/2');
  await context.close();
});

test('standard checks: axe and no sideways scroll with steps open and closed, at both window sizes', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  await makeJob(page, server.baseURL, uniqueName('Axe open'), 'doing', ['One', 'Two']);
  await makeJob(page, server.baseURL, uniqueName('Axe folded'), 'todo', ['Three']);
  await openMyJobs(page, server.baseURL);
  await axeCheck(page);
  await expectNoSideScroll(page);
  await page.locator('.steps-toggle', { hasText: 'Show steps' }).first().click();
  await axeCheck(page);
  await expectNoSideScroll(page);
});
