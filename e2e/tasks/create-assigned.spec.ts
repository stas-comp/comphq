import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// The one place this file knows how a task is created from the Board, so
// it changes in one line when the add-a-task form becomes the task window.
async function createTaskWithPerson(page: Page, title: string, stage: string, personName: string): Promise<void> {
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', stage);
  await page.selectOption('#new-task-people', { label: personName });
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

// SPEC gates 4.48–4.51 (the reported fault): a task created with a person
// on it, in any starting column, is on the Board and — unless it's already
// finished — in that person's My jobs and their Team lane straight away.
// An idea sits in its own group and never counts towards workload.
const cases = [
  { stage: 'idea', myJobsList: '.myjobs-ideas-list', teamHeading: 'Ideas', blocks: 0 },
  { stage: 'todo', myJobsList: '.myjobs-upnext-list', teamHeading: 'Up next', blocks: 2 },
  { stage: 'doing', myJobsList: '.myjobs-working-list', teamHeading: 'Working on now', blocks: 2 },
] as const;

for (const c of cases) {
  test(`gate 4.51: a task created with someone on it in ${c.stage} shows on the Board, My jobs and Team`, async ({
    page,
    server,
  }) => {
    const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    await ready(page);
    const title = uniqueName(`Assigned ${c.stage}`);
    await createTaskWithPerson(page, title, c.stage, me);

    await expect(page.locator(`.task-column[data-stage="${c.stage}"] .task-card`, { hasText: title })).toHaveCount(1);

    await page.goto(server.baseURL + '/tasks');
    await ready(page);
    await expect(page.locator(`.myjobs-mine ${c.myJobsList} .team-task`, { hasText: title })).toHaveCount(1);
    await expect(page.locator('.myjobs-grabs-list .team-task', { hasText: title })).toHaveCount(0);

    await page.goto(server.baseURL + '/tasks/team');
    await ready(page);
    const lane = page.locator('.team-lane', { has: page.locator('h2', { hasText: me }) });
    const group = lane.locator('h3', { hasText: c.teamHeading }).locator('xpath=following-sibling::ul[1]');
    await expect(group.locator('li', { hasText: title })).toHaveCount(1);
    await expect(page.locator('.team-lane[data-person-id="0"] li', { hasText: title })).toHaveCount(0);
    expect(await lane.locator('.workload-block').count()).toBe(c.blocks);
  });
}

// A task created already finished is on the Board only: finished jobs
// stay out of My jobs and Team (gates 2.25, 2.33) — see D-59.
test('gate 4.51: a task created finished with someone on it is on the Board, not in My jobs or Team', async ({
  page,
  server,
}) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('Assigned done');
  await createTaskWithPerson(page, title, 'done', me);
  await expect(page.locator('.task-column[data-stage="done"] .task-card', { hasText: title })).toHaveCount(1);

  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await expect(page.locator('.myjobs-page .team-task', { hasText: title })).toHaveCount(0);
  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  await expect(page.locator('.team-lane li', { hasText: title })).toHaveCount(0);
});

// SPEC gate 4.50: an idea nobody is on is still in Up for grabs; taking it
// moves it to To do.
test('gate 4.50: an unassigned idea is in Up for grabs, and taking it moves it to To do', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('Loose idea');
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', 'idea');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  const row = page.locator('.myjobs-grabs-list .team-task', { hasText: title });
  await expect(row).toHaveCount(1);
  await row.locator('button', { hasText: 'Take it' }).click();
  await ready(page);
  await expect(page.locator('.myjobs-upnext-list .team-task', { hasText: title })).toHaveCount(1);
  await expect(page.locator('.myjobs-ideas-list .team-task', { hasText: title })).toHaveCount(0);
});

test('standard page checks for My jobs and Team with an idea in the new group', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  await createTaskWithPerson(page, uniqueName('Checked idea'), 'idea', me);
  for (const path of ['/tasks', '/tasks/team']) {
    await page.goto(server.baseURL + path);
    await ready(page);
    await axeCheck(page);
    await expectNoSideScroll(page);
  }
});
