import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask, openTaskPage } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// Steps inside a job, with JavaScript switched off (PLAN P5-02, gates 5.08,
// 5.12, 5.13). Every step operation is an ordinary form that reloads the
// page; this is the layer everything else is built on top of, so it is
// tested before any script exists (P5-03).

test.use({ javaScriptEnabled: false });

/** Signs in, makes a job, and lands on its own page (gate 4.27: /tasks/{id}). */
async function startJob(page: Page, baseURL: string): Promise<{ name: string; title: string }> {
  const name = await signInAsNewPerson(page, baseURL, '/tasks/board');
  const title = uniqueName('Concert');
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  await openTaskPage(page, title);
  return { name, title };
}

const steps = (page: Page) => page.locator('#steps');
const rows = (page: Page) => page.locator('#steps .step');
const stepWords = (page: Page) => page.locator('#steps .step-words');

async function addStep(page: Page, text: string): Promise<void> {
  await page.fill('.step-add input[name="text"]', text);
  await page.click('.step-add button[type="submit"]');
}

/** Without script a box is ticked and then saved with its Update button. */
async function setTick(page: Page, text: string, ticked: boolean): Promise<void> {
  const r = row(page, text);
  await r.getByRole('checkbox', { name: text, exact: true }).setChecked(ticked);
  await r.getByRole('button', { name: 'Update' }).click();
}

const row = (page: Page, text: string) => page.locator('#steps .step', { has: page.locator('.step-words', { hasText: text }) });

test('gate 5.01, 5.02: an empty job shows only Add a step; steps are added one after another', async ({ page, server }) => {
  await startJob(page, server.baseURL);

  await expect(steps(page).getByRole('heading', { name: 'Steps' })).toBeVisible();
  await expect(page.getByPlaceholder('Add a step')).toBeVisible();
  await expect(page.locator('#steps .steps-list, #steps .steps-count, #steps progress')).toHaveCount(0);

  for (const words of ['Print exam papers', 'Book the hall', 'Email the parents']) await addStep(page, words);
  await expect(stepWords(page)).toHaveText(['Print exam papers', 'Book the hall', 'Email the parents']);
  await expect(page.getByPlaceholder('Add a step')).toHaveValue(''); // the box comes back empty
  await expect(steps(page).locator('.steps-count')).toHaveText('0 of 3 done');
});

test('gate 5.03, 5.04: ticking shows who and when, and the heading counts', async ({ page, server }) => {
  const { name } = await startJob(page, server.baseURL);
  for (const words of ['One', 'Two']) await addStep(page, words);

  await setTick(page, 'One', true);
  await expect(steps(page).locator('.steps-count')).toHaveText('1 of 2 done');
  await expect(page.getByRole('checkbox', { name: 'One', exact: true })).toBeChecked();
  await expect(row(page, 'One').locator('.step-by')).toHaveText(new RegExp(`^${name}, (Mon|Tue|Wed|Thu|Fri|Sat|Sun) \\d{1,2} [A-Z][a-z]{2}`));
  await expect(row(page, 'Two').locator('.step-by')).toHaveCount(0);

  await setTick(page, 'Two', true);
  await expect(steps(page).locator('.steps-count')).toHaveText('All 2 done');

  await setTick(page, 'One', false);
  await expect(steps(page).locator('.steps-count')).toHaveText('1 of 2 done');
  await expect(row(page, 'One').locator('.step-by')).toHaveCount(0);
});

test('gate 5.05: a step is renamed in its own form and cancelling changes nothing', async ({ page, server }) => {
  await startJob(page, server.baseURL);
  await addStep(page, 'Book the hall');

  await page.getByRole('link', { name: /^Rename/ }).click();
  const box = page.locator('#steps .step-rename input[name="text"]');
  await expect(box).toHaveValue('Book the hall');
  await steps(page).getByRole('link', { name: 'Cancel' }).click();
  await expect(page.locator('#steps .step-rename')).toHaveCount(0);
  await expect(stepWords(page)).toHaveText(['Book the hall']);

  await page.getByRole('link', { name: /^Rename/ }).click();
  await box.fill('Book the big hall');
  await steps(page).getByRole('button', { name: 'Save' }).click();
  await expect(stepWords(page)).toHaveText(['Book the big hall']);
  await expect(page.locator('#steps .step-rename')).toHaveCount(0);
});

test('gate 5.06: removing offers Undo straight away, which puts the step back where it was', async ({ page, server }) => {
  await startJob(page, server.baseURL);
  for (const words of ['One', 'Two', 'Three']) await addStep(page, words);

  await page.getByRole('button', { name: 'Remove Two' }).click();
  await expect(stepWords(page)).toHaveText(['One', 'Three']);
  await expect(steps(page).locator('.steps-undo')).toContainText('Removed “Two”');

  await steps(page).getByRole('button', { name: 'Undo' }).click();
  await expect(stepWords(page)).toHaveText(['One', 'Two', 'Three']);
  await expect(steps(page).locator('.steps-undo')).toHaveCount(0);

  // Remove, then do something else: the Undo is gone but the step is not
  // lost (it stays in the database and in the export).
  await page.getByRole('button', { name: 'Remove Two' }).click();
  await setTick(page, 'Three', true);
  await expect(steps(page).locator('.steps-undo')).toHaveCount(0);
  await expect(stepWords(page)).toHaveText(['One', 'Three']);
});

test('gate 5.07: steps are put in order with the up and down buttons', async ({ page, server }) => {
  await startJob(page, server.baseURL);
  for (const words of ['One', 'Two', 'Three']) await addStep(page, words);

  await expect(page.getByRole('button', { name: 'Move One up' })).toBeDisabled();
  await expect(page.getByRole('button', { name: 'Move Three down' })).toBeDisabled();

  await page.getByRole('button', { name: 'Move Three up' }).click();
  await expect(stepWords(page)).toHaveText(['One', 'Three', 'Two']);
  await page.getByRole('button', { name: 'Move One down' }).click();
  await expect(stepWords(page)).toHaveText(['Three', 'One', 'Two']);
});

test('gate 5.13: the two limits say so in plain words and keep what was typed', async ({ page, server }) => {
  await startJob(page, server.baseURL);
  const box = page.locator('.step-add input[name="text"]');
  const jobUrl = page.url();

  await page.click('.step-add button[type="submit"]'); // nothing typed
  await expect(page.getByRole('alert')).toHaveText('Type what the step is first.');

  const long = 'x'.repeat(201);
  await addStep(page, long);
  await expect(page.getByRole('alert')).toHaveText('A step can be up to 200 characters.');
  await expect(box).toHaveValue(long);
  await expect(rows(page)).toHaveCount(0);

  // Fill the list to 50 quickly, the same way the form does.
  const jobPath = new URL(jobUrl).pathname;
  for (let i = 1; i <= 50; i++) {
    const res = await page.request.post(server.baseURL + jobPath + '/steps', {
      headers: { origin: server.baseURL },
      form: { text: `Step ${i}` },
      maxRedirects: 0,
    });
    expect(res.status()).toBe(302);
  }
  await page.goto(server.baseURL + jobPath);
  await expect(rows(page)).toHaveCount(50);
  await addStep(page, 'The fifty-first');
  await expect(page.getByRole('alert')).toHaveText('A job can have up to 50 steps.');
  await expect(page.locator('.step-add input[name="text"]')).toHaveValue('The fifty-first');
  await expect(rows(page)).toHaveCount(50);
});

test('gate 5.12: two people on two computers meet no error page', async ({ page, server, browser }) => {
  await startJob(page, server.baseURL);
  await addStep(page, 'Book the hall');
  await addStep(page, 'Email the parents');
  const jobUrl = page.url();

  // A second person, on a second computer, looking at the same job.
  const other = await browser.newContext({ javaScriptEnabled: false });
  const otherPage = await other.newPage();
  const otherName = await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await otherPage.goto(jobUrl);
  await expect(stepWords(otherPage)).toHaveText(['Book the hall', 'Email the parents']);

  // First person ticks; the second, still looking at the old page, ticks the
  // same step: no error, and the first person's name stays.
  await setTick(page, 'Book the hall', true);
  await setTick(otherPage, 'Book the hall', true);
  await expect(otherPage.getByRole('alert')).toHaveCount(0);
  await expect(row(otherPage, 'Book the hall').locator('.step-by')).not.toContainText(otherName);
  await expect(row(otherPage, 'Book the hall').getByRole('checkbox')).toBeChecked();

  // First person removes a step; the second, on the stale page, tries to
  // tick it: a plain sentence, on the page, and nothing changes.
  await page.getByRole('button', { name: 'Remove Email the parents' }).click();
  await setTick(otherPage, 'Email the parents', true);
  await expect(otherPage.getByRole('alert')).toHaveText('Somebody else removed that step.');
  await expect(stepWords(otherPage)).toHaveText(['Book the hall']);
  await other.close();
});

// The accessibility check needs script to run, so this one test runs with it
// on; the section is the same plain markup either way.
test.describe('with JavaScript on', () => {
  test.use({ javaScriptEnabled: true });

  test('gate 5.16: the Steps section on the job page passes the accessibility check, and so does a step being renamed', async ({ page, server }) => {
    await startJob(page, server.baseURL);
    for (const words of ['One', 'Two']) await addStep(page, words);
    // Tick one the way the form does (the script that ticks in place is P5-03's).
    const stepId = await row(page, 'One').getAttribute('data-step-id');
    const res = await page.request.post(`${page.url().split('#')[0]}/steps/${stepId}/tick`, {
      headers: { origin: server.baseURL },
      form: { done: '1' },
    });
    expect(res.ok()).toBe(true);
    await page.reload();
    await expect(steps(page).locator('.steps-count')).toHaveText('1 of 2 done');
    await axeCheck(page);

    await page.getByRole('link', { name: /^Rename/ }).first().click();
    await expect(page.locator('#steps .step-rename input[name="text"]')).toBeVisible();
    await axeCheck(page);
  });
});
