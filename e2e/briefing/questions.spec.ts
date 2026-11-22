import { expect, test } from '../helpers/fixtures-today';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

type Page = import('@playwright/test').Page;

// SPEC gates 8.77 and 8.79 (B14.6, D-110): "Questions waiting for an answer" in the
// Saturday Briefing, and its refresh. Each test is @fresh: it counts the questions
// on the page, so it needs its own empty server.

test.use({ today: '2026-11-21' });

const section = (page: Page) => page.locator('section[aria-labelledby="briefing-questions"]');

async function ask(page: Page, baseURL: string, text: string): Promise<void> {
  const res = await page.request.post(baseURL + '/questions', { form: { text }, headers: { origin: baseURL }, maxRedirects: 0 });
  expect(res.status()).toBeLessThan(400);
}

async function open(page: Page, baseURL: string, path = '/briefing') {
  await page.goto(baseURL + path);
  await ready(page);
}

test('@fresh gate 8.77: with none waiting, the section says so', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/briefing');
  await ready(page);
  await expect(section(page).getByRole('heading', { name: 'Questions waiting for an answer' })).toBeVisible();
  await expect(section(page)).toContainText('No questions waiting for an answer.');
  await expect(section(page).locator('.briefing-question')).toHaveCount(0);
  await axeCheck(page);
});

test('@fresh gate 8.77: it follows Coming up, lists up to 5 oldest first, then "and N more"', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/briefing');
  for (let i = 1; i <= 3; i++) await ask(page, server.baseURL, `Waiting question ${i}`);
  await open(page, server.baseURL);

  // After Coming up, in the page's order.
  const headings = await page.locator('.briefing-section h2').allTextContents();
  expect(headings.map((h) => h.trim())).toEqual(['Must be done today', 'This week', 'Coming up', 'Questions waiting for an answer']);

  // Three: all listed, oldest first, no "more" link, each with who, when and answers.
  await expect(section(page).locator('.briefing-question')).toHaveCount(3);
  await expect(section(page).locator('.briefing-question').first()).toContainText('Waiting question 1');
  await expect(section(page).locator('.briefing-question').first()).toContainText('Asked by');
  await expect(section(page).locator('.briefing-question').first()).toContainText('0 answers');
  await expect(section(page).getByRole('link', { name: /more$/ })).toHaveCount(0);

  // Five: still no "more".
  for (let i = 4; i <= 5; i++) await ask(page, server.baseURL, `Waiting question ${i}`);
  await open(page, server.baseURL);
  await expect(section(page).locator('.briefing-question')).toHaveCount(5);
  await expect(section(page).getByRole('link', { name: /more$/ })).toHaveCount(0);

  // Eight: the five oldest and "and 3 more", which opens Questions.
  for (let i = 6; i <= 8; i++) await ask(page, server.baseURL, `Waiting question ${i}`);
  await open(page, server.baseURL);
  const rows = section(page).locator('.briefing-question');
  await expect(rows).toHaveCount(5);
  await expect(rows.nth(0)).toContainText('Waiting question 1');
  await expect(rows.nth(4)).toContainText('Waiting question 5');
  await expect(section(page)).not.toContainText('Waiting question 6');
  const more = section(page).getByRole('link', { name: 'and 3 more' });
  await expect(more).toHaveAttribute('href', '/questions');

  await axeCheck(page);
  await expectNoSideScroll(page);
  await page.setViewportSize({ width: 390, height: 800 });
  await expectNoSideScroll(page);
  await page.setViewportSize({ width: 1366, height: 768 });

  // Clicking a question opens it; "and 3 more" opens Questions.
  await rows.nth(1).getByRole('link').click();
  await expect(page.locator('.q-title')).toHaveText('Waiting question 2');
  await page.goBack();
  await ready(page);
  await section(page).getByRole('link', { name: 'and 3 more' }).click();
  await expect(page).toHaveURL(/\/questions$/);
});

test('@fresh gate 8.77: answered questions never appear, and Just mine still shows them', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/briefing');
  await ask(page, server.baseURL, 'Still waiting');
  await ask(page, server.baseURL, 'Already answered');
  await page.goto(server.baseURL + '/questions');
  await ready(page);
  await page.locator('.q-row', { hasText: 'Already answered' }).getByRole('link').click();
  await page.getByRole('button', { name: 'Mark answered' }).click();
  await ready(page);

  await open(page, server.baseURL);
  await expect(section(page).locator('.briefing-question')).toHaveCount(1);
  await expect(section(page)).toContainText('Still waiting');
  await expect(section(page)).not.toContainText('Already answered');

  // Questions belong to nobody, so Just mine keeps them.
  await open(page, server.baseURL, '/briefing?mine=1');
  await expect(section(page).locator('.briefing-question')).toHaveCount(1);
  await expect(section(page)).toContainText('Still waiting');

  // Marking it answered takes it off again.
  await page.goto(server.baseURL + '/questions');
  await ready(page);
  await page.locator('.q-row', { hasText: 'Still waiting' }).getByRole('link').click();
  await page.getByRole('button', { name: 'Mark answered' }).click();
  await ready(page);
  await open(page, server.baseURL);
  await expect(section(page)).toContainText('No questions waiting for an answer.');
});

test('@fresh gate 8.79: a question asked on one computer shows in the Briefing and on Questions on another within 60 seconds', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/briefing');
  const context = await browser.newContext();
  const other = await context.newPage();
  await other.clock.install();
  await signInAsNewPerson(other, server.baseURL, '/briefing');
  await ready(other);
  const second = await context.newPage();
  await second.clock.install();
  await second.goto(server.baseURL + '/questions');
  await ready(second);

  await ask(page, server.baseURL, 'Asked on the first computer');
  await expect(section(other)).toContainText('No questions waiting for an answer.');
  await other.clock.runFor(61_000);
  await expect(section(other).locator('.briefing-question')).toHaveCount(1);
  await expect(section(other)).toContainText('Asked on the first computer');
  await second.clock.runFor(61_000);
  await expect(second.locator('.q-row', { hasText: 'Asked on the first computer' })).toHaveCount(1);

  // Marking it answered elsewhere is seen too, and so is clicking back into the window.
  await page.goto(server.baseURL + '/questions');
  await ready(page);
  await page.locator('.q-row', { hasText: 'Asked on the first computer' }).getByRole('link').click();
  await page.getByRole('button', { name: 'Mark answered' }).click();
  await ready(page);
  await other.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(section(other)).toContainText('No questions waiting for an answer.');
  await context.close();
});
