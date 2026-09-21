import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';

// Steps across an upgrade and a rollback (SPEC gate 5.15, PLAN P5-09): a job
// with real steps is made under the new build, then the container is rolled
// back to the previous release, which knows nothing about steps and must
// neither break nor lose them, and then upgraded again, which must find every
// one exactly as it was.
//
// Tagged @steps-seed / @steps-verify / @steps-rolledback rather than reusing
// @seed / @verify, so the harness's other grep-filtered runs never pick these
// up (neither "@seed" nor "@verify" is a substring of them). The job's title is
// fixed, because a later run has to find it; the person is not (a fixed name
// would collide with the duplicate-name refusal on the second seeding).
const JOB_TITLE = 'Smoke Test Job With Steps';
const STEPS = ['Print exam papers', 'Book the hall', 'Email the parents'];

async function openJob(page: import('@playwright/test').Page, baseURL: string): Promise<void> {
  await page.goto(baseURL + '/tasks/board');
  await ready(page);
  // .first(): the seeding runs on both builds against one data folder, so an
  // earlier same-titled job may exist; every one of them carries the same steps.
  const href = await page.locator('.task-card', { hasText: JOB_TITLE }).locator('.task-card-title a').first().getAttribute('href');
  await page.goto(baseURL + href!);
  await ready(page);
}

test('@steps-seed a job with steps, one of them ticked and one removed', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await openNewTask(page);
  await page.fill('#new-task-title', JOB_TITLE);
  await page.selectOption('#new-task-stage', 'todo');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  await openJob(page, server.baseURL);

  for (const text of [...STEPS, 'A step that will be removed']) {
    await page.fill('.step-add input[name="text"]', text);
    await page.click('.step-add button[type="submit"]');
    await expect(page.locator('#steps .step-words', { hasText: text })).toHaveCount(1);
  }
  await page.getByRole('button', { name: 'Remove A step that will be removed' }).click();
  await expect(page.locator('#steps .step-words', { hasText: 'A step that will be removed' })).toHaveCount(0);
  await page.getByRole('checkbox', { name: STEPS[1], exact: true }).check();
  await expect(page.locator('#steps .steps-count')).toHaveText('1 of 3 done');
});

test('@steps-verify the steps are all there, with who ticked what', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await openJob(page, server.baseURL);
  await expect(page.locator('#steps .step-words')).toHaveText(STEPS);
  await expect(page.getByRole('checkbox', { name: STEPS[1], exact: true })).toBeChecked();
  await expect(page.locator('#steps .step', { has: page.locator('.step-words', { hasText: STEPS[1] }) }).locator('.step-by')).toHaveText(/^.+, \w{3} \d{1,2} \w{3}/);
  await expect(page.locator('#steps .steps-count')).toHaveText('1 of 3 done');
});

// Run against the PREVIOUS release's app, which has no idea steps exist: its
// own screens must still work with the rows in the database, and the job it
// knew about must still be there.
test('@steps-rolledback the previous release still opens the job, and ignores its steps', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await expect(page.locator('.task-card', { hasText: JOB_TITLE }).first()).toBeVisible();
  const href = await page.locator('.task-card', { hasText: JOB_TITLE }).locator('.task-card-title a').first().getAttribute('href');
  const res = await page.goto(server.baseURL + href!);
  expect(res!.status()).toBe(200);
  await ready(page);
  await expect(page.locator('h1')).toContainText(JOB_TITLE);
});
