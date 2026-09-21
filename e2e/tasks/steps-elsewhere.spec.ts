import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';

type Page = import('@playwright/test').Page;

// Steps show up where they should and nowhere else (PLAN P5-04, gates 5.10,
// 5.11, 5.14): the small 3/7 on a card, History, and steps.csv.

/** Makes a job that is on the signed-in person and in To do, so it shows on all three screens. */
async function makeJob(page: Page, baseURL: string): Promise<{ name: string; title: string; id: string }> {
  const name = await signInAsNewPerson(page, baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('Concert');
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', 'todo');
  await page.selectOption('#new-task-people', [{ label: name }]);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  const href = await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').first().getAttribute('href');
  return { name, title, id: href!.split('/').pop()! };
}

/** Does what the forms do, quickly: posts to a step route as the signed-in person. */
async function post(page: Page, baseURL: string, urlPath: string, form: Record<string, string>): Promise<void> {
  const res = await page.request.post(baseURL + urlPath, { headers: { origin: baseURL }, form, maxRedirects: 0 });
  expect([200, 302]).toContain(res.status());
}

async function addSteps(page: Page, baseURL: string, id: string, texts: string[]): Promise<void> {
  for (const text of texts) await post(page, baseURL, `/tasks/${id}/steps`, { text });
}

async function stepIds(page: Page, baseURL: string, id: string): Promise<string[]> {
  await page.goto(`${baseURL}/tasks/${id}`);
  await ready(page);
  return page.locator('.task-steps .step').evaluateAll((els) => els.map((el) => (el as HTMLElement).dataset.stepId!));
}

const cardOn = {
  board: (page: Page, id: string) => page.locator(`.task-card[data-task-id="${id}"]`),
  team: (page: Page, id: string) => page.locator(`.team-task[data-task-id="${id}"]`).first(),
  myjobs: (page: Page, id: string) => page.locator(`.team-task[data-task-id="${id}"]`).first(),
};
const screens = { board: '/tasks/board', team: '/tasks/team', myjobs: '/tasks' } as const;
type Screen = keyof typeof screens;
const eachScreen = Object.entries(screens) as [Screen, string][];

test('gate 5.10: a card carries 3/7 when the job has steps, a tick when all are done, and nothing when it has none', async ({ page, server }) => {
  const { id } = await makeJob(page, server.baseURL);

  for (const [screen, url] of eachScreen) {
    await page.goto(server.baseURL + url);
    await ready(page);
    await expect(cardOn[screen](page, id), `${screen}: the job is on the screen`).toBeVisible();
    await expect(cardOn[screen](page, id).locator('.step-badge'), `${screen}: no steps, no badge`).toHaveCount(0);
  }

  await addSteps(page, server.baseURL, id, ['One', 'Two', 'Three']);
  const ids = await stepIds(page, server.baseURL, id);
  await post(page, server.baseURL, `/tasks/${id}/steps/${ids[0]}/tick`, { done: '1' });

  for (const [screen, url] of eachScreen) {
    await page.goto(server.baseURL + url);
    await ready(page);
    const badge = cardOn[screen](page, id).locator('.step-badge');
    await expect(badge, `${screen}: the count`).toHaveCount(1);
    await expect(badge.locator('[aria-hidden="true"]')).toHaveText('1/3');
    await expect(badge).toHaveAttribute('title', '1 of 3 steps done');
    await expect(badge.locator('.visually-hidden')).toHaveText('1 of 3 steps done');
  }

  await post(page, server.baseURL, `/tasks/${id}/steps/${ids[1]}/tick`, { done: '1' });
  await post(page, server.baseURL, `/tasks/${id}/steps/${ids[2]}/tick`, { done: '1' });
  for (const [screen, url] of eachScreen) {
    await page.goto(server.baseURL + url);
    await ready(page);
    const badge = cardOn[screen](page, id).locator('.step-badge');
    await expect(badge, `${screen}: all done`).toHaveClass(/\bdone\b/);
    await expect(badge.locator('.icon-check')).toBeVisible(); // a tick, not 3/3
    await expect(badge).not.toContainText('/');
    await expect(badge.locator('.visually-hidden')).toHaveText('All 3 steps done');
  }

  // Removing a step takes it out of the count; removing them all takes the badge away.
  await post(page, server.baseURL, `/tasks/${id}/steps/${ids[2]}/remove`, {});
  await page.goto(server.baseURL + '/tasks/board');
  await expect(cardOn.board(page, id).locator('.step-badge .icon-check')).toBeVisible(); // 2 of 2
  await post(page, server.baseURL, `/tasks/${id}/steps/${ids[0]}/remove`, {});
  await post(page, server.baseURL, `/tasks/${id}/steps/${ids[1]}/remove`, {});
  await page.goto(server.baseURL + '/tasks/board');
  await expect(cardOn.board(page, id).locator('.step-badge')).toHaveCount(0);
});

test('gate 5.10: a step ticked in the window changes the card behind it without a reload', async ({ page, server }) => {
  const { id, title } = await makeJob(page, server.baseURL);
  await addSteps(page, server.baseURL, id, ['One', 'Two']);
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await page.evaluate(() => ((window as unknown as { __kept: boolean }).__kept = true));
  await expect(cardOn.board(page, id).locator('.step-badge [aria-hidden="true"]')).toHaveText('0/2');

  await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').first().click();
  await page.getByRole('checkbox', { name: 'One', exact: true }).check();
  await expect(cardOn.board(page, id).locator('.step-badge [aria-hidden="true"]')).toHaveText('1/2');
  expect(await page.evaluate(() => (window as unknown as { __kept?: boolean }).__kept)).toBe(true);
});

// D-71, and the test that is the point of it: ticking never goes into History.
test('gate 5.11: History records adding, renaming, removing and restoring a step, and never a tick', async ({ page, server }) => {
  const { name, id } = await makeJob(page, server.baseURL);
  await page.goto(`${server.baseURL}/tasks/${id}`);
  const history = page.locator('.task-activity-list li');
  await expect(history).toHaveCount(1); // "created this"

  const texts = Array.from({ length: 10 }, (_, i) => `Step number ${i + 1}`);
  await addSteps(page, server.baseURL, id, texts);
  const ids = await stepIds(page, server.baseURL, id);
  const before = await history.allTextContents();
  expect(before.length).toBe(11); // created + ten adds
  expect(before[0]).toContain(`${name} added the step “Step number 10”`);

  // Tick all ten, untick some, move one: History is exactly what it was.
  for (const sid of ids) await post(page, server.baseURL, `/tasks/${id}/steps/${sid}/tick`, { done: '1' });
  for (const sid of ids.slice(0, 5)) await post(page, server.baseURL, `/tasks/${id}/steps/${sid}/tick`, { done: '0' });
  await post(page, server.baseURL, `/tasks/${id}/steps/${ids[9]}/move`, { direction: 'up' });
  await page.reload();
  await expect(history).toHaveCount(11);
  expect(await history.allTextContents()).toEqual(before);

  // The three that are recorded after that (adding was the first).
  await post(page, server.baseURL, `/tasks/${id}/steps/${ids[0]}`, { text: 'Step number one, renamed' });
  await post(page, server.baseURL, `/tasks/${id}/steps/${ids[1]}/remove`, {});
  await post(page, server.baseURL, `/tasks/${id}/steps/${ids[1]}/restore`, {});
  await page.reload();
  const after = (await history.allTextContents()).map((t) => t.split(' · ')[0]);
  expect(after.slice(0, 3)).toEqual([
    `${name} restored the step “Step number 2”`,
    `${name} removed the step “Step number 2”`,
    `${name} renamed the step “Step number 1” to “Step number one, renamed”`,
  ]);
  await expect(history).toHaveCount(14);
});

test('gate 5.11: the same History shows in the window, without the tick', async ({ page, server }) => {
  const { id, title, name } = await makeJob(page, server.baseURL);
  await addSteps(page, server.baseURL, id, ['Book the hall']);
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').first().click();
  await page.getByRole('checkbox', { name: 'Book the hall', exact: true }).check();
  await expect(page.locator('#task-window .steps-count')).toHaveText('All 1 done');
  await page.locator('#task-window .task-history summary').click();
  await expect(page.locator('#task-window .task-history')).toContainText(`${name} added the step “Book the hall”`);
  await expect(page.locator('#task-window .task-history li')).toHaveCount(2); // created + added; the tick is not there
});

test('gate 5.14: Export everything includes steps.csv, opening in Excel, with removed steps marked', async ({ page, server }) => {
  const { name, title, id } = await makeJob(page, server.baseURL);
  await addSteps(page, server.baseURL, id, ['Print exam papers', 'Book the hall', 'Email the parents']);
  const ids = await stepIds(page, server.baseURL, id);
  await post(page, server.baseURL, `/tasks/${id}/steps/${ids[0]}/tick`, { done: '1' });
  await post(page, server.baseURL, `/tasks/${id}/steps/${ids[1]}/remove`, {});

  const res = await page.request.get(server.baseURL + '/settings/export');
  expect(res.ok()).toBeTruthy();
  const workDir = fs.mkdtempSync(path.join(os.tmpdir(), 'comphq-export-steps-'));
  const zipPath = path.join(workDir, 'export.zip');
  const extractDir = path.join(workDir, 'extracted');
  fs.writeFileSync(zipPath, await res.body());
  const unzipBin = process.env.COMPHQ_UNZIP_BINARY;
  if (!unzipBin) throw new Error('COMPHQ_UNZIP_BINARY is not set; global setup should have built it');
  execFileSync(unzipBin, [zipPath, extractDir]);

  const raw = fs.readFileSync(path.join(extractDir, 'steps.csv'));
  expect(raw.subarray(0, 3).equals(Buffer.from([0xef, 0xbb, 0xbf]))).toBe(true); // Excel reads it as UTF-8
  const text = raw.subarray(3).toString('utf8');
  expect(text.startsWith('Job,Step,Done,Ticked by,Ticked,Removed\r\n')).toBe(true);
  const mine = text.split('\r\n').filter((l) => l.startsWith(title + ','));
  expect(mine).toHaveLength(3);
  // A date with a time has a comma in it, so a spreadsheet quotes it.
  const when = '"[A-Z][a-z]{2} \\d{1,2} [A-Z][a-z]{2} \\d{4}, \\d\\d:\\d\\d"';
  expect(mine[0]).toMatch(new RegExp(`^${title},Print exam papers,Yes,${name},${when},$`));
  expect(mine[1]).toBe(`${title},Email the parents,No,,,`);
  // The removed step is still there, and says when it was removed.
  expect(mine[2]).toMatch(new RegExp(`^${title},Book the hall,No,,,${when}$`));
});

test('a removed step leaves the list but not the database: the restore route still finds it', async ({ page, server }) => {
  const { id } = await makeJob(page, server.baseURL);
  await addSteps(page, server.baseURL, id, ['Keep me']);
  const [sid] = await stepIds(page, server.baseURL, id);
  await post(page, server.baseURL, `/tasks/${id}/steps/${sid}/remove`, {});
  await page.goto(`${server.baseURL}/tasks/${id}`);
  await expect(page.locator('.task-steps .step')).toHaveCount(0);
  await post(page, server.baseURL, `/tasks/${id}/steps/${sid}/restore`, {});
  await page.reload();
  await expect(page.locator('.task-steps .step-words')).toHaveText(['Keep me']);
});
