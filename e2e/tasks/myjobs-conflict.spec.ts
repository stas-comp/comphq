import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

async function addTask(page: Page, title: string, stage: string, personName?: string): Promise<void> {
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', stage);
  if (personName) {
    await page.selectOption('#new-task-people', { label: personName });
  }
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

async function taskIDFromLink(page: Page, title: string): Promise<string> {
  const href = await page.locator('.team-task', { hasText: title }).locator('a').first().getAttribute('href');
  const match = href?.match(/\/tasks\/(\d+)/);
  if (!match) throw new Error(`couldn't find a task id for ${title}`);
  return match[1];
}

// Backdates done_at without waiting on the real 14-day window (SPEC
// gate 2.08) — the same test-only-route pattern already used for
// backups' set-last-backup-at and Finished tasks' own tests.
async function setTaskDoneDaysAgo(server: { baseURL: string }, page: Page, taskID: string, daysAgo: number): Promise<void> {
  const res = await page.request.post(server.baseURL + '/__test/tasks/set-done-at', {
    form: { task_id: taskID, days_ago: String(daysAgo) },
    headers: { origin: server.baseURL },
  });
  expect(res.ok()).toBeTruthy();
}

function takeItButton(page: Page, title: string) {
  return page.locator('.myjobs-grabs-list .team-task', { hasText: title }).locator('button', { hasText: 'Take it' });
}

// SPEC gate 2.36: the second person to take an already-taken job sees
// the exact message, and nothing is added to their list.
test('gate 2.36: two people taking the same job — the first succeeds, the second sees the exact message', async ({ page, server, browser }) => {
  const nameA = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks');
  await ready(otherPage);

  const title = uniqueName('Contested job');
  await addTask(page, title, 'todo');

  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await otherPage.reload();
  await ready(otherPage);

  // nameA takes it first.
  await takeItButton(page, title).click();
  await ready(page);
  await expect(page.locator('.myjobs-upnext-list .team-task', { hasText: title })).toHaveCount(1);

  // otherPage's stale render still shows the Take it button for the
  // same job — clicking it is exactly the "almost the same moment"
  // race SPEC gate 2.36 describes.
  const resp = otherPage.waitForResponse((res) => /\/tasks\/\d+\/take$/.test(new URL(res.url()).pathname));
  await takeItButton(otherPage, title).click();
  const takeResponse = await resp;
  expect(takeResponse.status()).toBe(409);
  await expect(otherPage.locator('.message', { hasText: `${nameA} has just taken this job.` })).toHaveCount(1);
  await expect(otherPage.locator('.myjobs-upnext-list .team-task', { hasText: title })).toHaveCount(0);

  await otherContext.close();
});

// SPEC gate 2.36: "A taken job leaves Up for grabs on every computer
// within 60 seconds, or straight away when that window is clicked back
// into." Reuses refresh.spec.ts's own page.clock technique.
test('gate 2.36: a taken job leaves Up for grabs on another computer within 60 seconds', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await otherPage.clock.install();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks');
  await ready(otherPage);

  const title = uniqueName('Seen leaving after 60s');
  await addTask(page, title, 'todo');
  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await otherPage.reload();
  await ready(otherPage);

  await expect(otherPage.locator('.myjobs-grabs-list .team-task', { hasText: title })).toHaveCount(1);
  await takeItButton(page, title).click();
  await ready(page);

  await otherPage.clock.runFor(61_000);
  await expect(otherPage.locator('.myjobs-grabs-list .team-task', { hasText: title })).toHaveCount(0);

  await otherContext.close();
});

test('gate 2.36: a taken job leaves Up for grabs on another computer immediately on focus', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks');
  await ready(otherPage);

  const title = uniqueName('Seen leaving on focus');
  await addTask(page, title, 'todo');
  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await otherPage.reload();
  await ready(otherPage);

  await expect(otherPage.locator('.myjobs-grabs-list .team-task', { hasText: title })).toHaveCount(1);
  await takeItButton(page, title).click();
  await ready(page);

  await otherPage.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(otherPage.locator('.myjobs-grabs-list .team-task', { hasText: title })).toHaveCount(0);

  await otherContext.close();
});

// SPEC gate 2.37: Done leaves My jobs, and — after the existing 14-day
// rule (gate 2.08) — the job reaches Finished tasks the same way any
// other Done task does.
test('gate 2.37: Done leaves My jobs, and reaches Finished tasks after 14 days', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const title = uniqueName('Finish me');
  await addTask(page, title, 'todo', me);

  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  const id = await taskIDFromLink(page, title);

  await page.locator('.myjobs-upnext-list .team-task', { hasText: title }).locator('button', { hasText: 'Done' }).click();
  await ready(page);
  await expect(page.locator('.myjobs-mine .team-task', { hasText: title })).toHaveCount(0);

  await setTaskDoneDaysAgo(server, page, id, 15);

  await page.goto(server.baseURL + '/tasks/finished');
  await ready(page);
  await expect(page.locator('.task-card', { hasText: title })).toHaveCount(1);
});

// SPEC gate 2.37: Give back takes the person off a job; it returns to
// Up for grabs only once nobody else is on it.
test('gate 2.37: Give back returns a job to Up for grabs, but not while someone else is still on it', async ({ page, server, browser }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  const other = await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);
  await otherContext.close();

  await page.reload();
  await ready(page);

  const solo = uniqueName('Give back solo');
  await addTask(page, solo, 'todo', me);
  const shared = uniqueName('Give back shared');
  await page.fill('#new-task-title', shared);
  await page.selectOption('#new-task-stage', 'todo');
  await page.selectOption('#new-task-people', [{ label: me }, { label: other }]);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  await page.goto(server.baseURL + '/tasks');
  await ready(page);

  await page.locator('.myjobs-upnext-list .team-task', { hasText: solo }).locator('button', { hasText: 'Give back' }).click();
  await ready(page);
  await expect(page.locator('.myjobs-grabs-list .team-task', { hasText: solo })).toHaveCount(1);

  await page.locator('.myjobs-upnext-list .team-task', { hasText: shared }).locator('button', { hasText: 'Give back' }).click();
  await ready(page);
  await expect(page.locator('.myjobs-grabs-list .team-task', { hasText: shared })).toHaveCount(0);
  await expect(page.locator('.myjobs-mine .team-task', { hasText: shared })).toHaveCount(0);
});

test('standard page checks for My jobs with a conflict message showing', async ({ page, server, browser }) => {
  const nameA = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks');
  await ready(otherPage);

  const title = uniqueName('Contested for axe');
  await addTask(page, title, 'todo');
  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await otherPage.reload();
  await ready(otherPage);

  await takeItButton(page, title).click();
  await ready(page);
  await takeItButton(otherPage, title).click();
  await ready(otherPage);
  await expect(otherPage.locator('.message')).toHaveCount(1);

  await axeCheck(otherPage);
  await otherContext.close();
});
