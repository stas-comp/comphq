import { expect, test } from '../helpers/fixtures-today';
import { myPersonID } from '../helpers/tasks';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

type Page = import('@playwright/test').Page;

// Screenshots of what Phase 8 added, for the owner's checks (O8.1-O8.6): the Board's
// Done column, a job's Articles and places, the Add task window with steps, My jobs
// with steps open, Yearly jobs, the Briefing with holidays and questions, and both
// Questions pages. (The December 2027 Calendar is in holidays.spec.ts.) With a fixed
// Saturday, 21 November 2026, so Thanksgiving is in This week.

test.use({ viewport: { width: 1366, height: 768 }, today: '2026-11-21' });
test.describe.configure({ timeout: 120_000 });

async function post(page: Page, baseURL: string, path: string, form: Record<string, string | string[]>): Promise<void> {
  const res = await page.request.post(baseURL + path, { form, headers: { origin: baseURL }, maxRedirects: 0 });
  expect(res.status(), path).toBeLessThan(400);
}

test('@fresh screenshots: Phase 8 screens', async ({ page, server }) => {
  const base = server.baseURL;
  await signInAsNewPerson(page, base, '/tasks/board');
  const me = await myPersonID(page, base);

  // A category and two articles, to link.
  await post(page, base, '/kb/categories', { name: 'Mailings' });
  await page.goto(base + '/kb/new');
  const category = (await page.locator('select[name="category_id"] option', { hasText: 'Mailings' }).getAttribute('value'))!;
  for (const title of ['How to send the quarterly mailing', 'Using the mailing lists']) {
    await post(page, base, '/kb/articles', { title, category_id: category, body_html: `<p>${title}</p>` });
  }

  // The Board: a few jobs, and Done with the most recently finished on top.
  for (const [title, stage] of [
    ['Order stamps', 'todo'],
    ['Check the postage meter', 'doing'],
    ['Print the labels', 'done'],
    ['Stuff the envelopes', 'done'],
    ['Post the quarterly mailing', 'done'],
  ] as const) {
    await post(page, base, '/tasks', { title, stage, person_id: me, due_date: stage === 'todo' ? '27/11/2026' : '' });
  }
  await page.goto(base + '/tasks/board?person=mine');
  await ready(page);
  await page.locator('.task-column[data-stage="done"]').screenshot({ path: 'reports/screens/tasks-done-column.png' });

  // A job with articles and a COSMOS place, with steps, open in its window.
  await post(page, base, '/tasks', {
    title: 'Send the quarterly mailing',
    stage: 'doing',
    person_id: me,
    due_date: '03/12/2026',
    steps_text: 'Pick the list\nPrint the labels\nStuff the envelopes\nPost them',
    place_name: 'Mailing lists 2026',
    place_path: '\\\\NAS\\COSMOS\\Mailings\\2026',
  });
  await page.goto(base + '/tasks/board?person=mine');
  await ready(page);
  const href = await page.locator('.task-card', { hasText: 'Send the quarterly mailing' }).locator('.task-card-title a').first().getAttribute('href');
  const id = /\/tasks\/(\d+)/.exec(href!)![1];
  const articleHref = await page.goto(base + '/kb/search?q=mailing').then(() => page.locator('.kb-search-results a').first().getAttribute('href'));
  await post(page, base, `/tasks/${id}/articles`, { article_id: /\/kb\/articles\/(\d+)/.exec(articleHref!)![1] });
  await page.goto(base + '/tasks/board?person=mine');
  await ready(page);
  await page.locator('.task-card', { hasText: 'Send the quarterly mailing' }).locator('.task-card-title a').click();
  await expect(page.locator('#task-window[open]')).toBeVisible();
  await page.locator('#task-window .task-refs').scrollIntoViewIfNeeded();
  await page.screenshot({ path: 'reports/screens/task-window-articles-places.png' });
  await page.keyboard.press('Escape');

  // The Add task window with steps already added.
  await page.goto(base + '/tasks/board?person=mine');
  await ready(page);
  await page.click('#add-task-link');
  await expect(page.locator('#task-window[open]')).toBeVisible();
  await page.locator('#task-window #new-task-title').fill('Prepare the open house');
  for (const s of ['Book the hall', 'Send the invitations', 'Order the food']) {
    await page.locator('#task-window #new-step-text').fill(s);
    await page.locator('#task-window #new-step-text').press('Enter');
  }
  await page.locator('#task-window .new-steps').scrollIntoViewIfNeeded();
  await page.screenshot({ path: 'reports/screens/add-task-steps.png' });
  await page.keyboard.press('Escape');

  // My jobs with a card's steps open.
  await page.goto(base + '/tasks');
  await ready(page);
  const mine = page.locator('li.team-task', { hasText: 'Send the quarterly mailing' });
  // Steps are open by default on a job being worked on now (gate 8.35).
  if (await mine.locator('.card-steps-body').isHidden()) await mine.locator('.steps-toggle').click();
  await expect(mine.locator('.card-steps-body')).toBeVisible();
  await page.screenshot({ path: 'reports/screens/myjobs-steps-open.png', fullPage: true });

  // Yearly jobs: one waiting, one showing.
  await post(page, base, '/tasks', { title: 'File the annual report', stage: 'todo', due_date: '15/09/2027', repeat: 'yearly', notice_weeks: '6' });
  await post(page, base, '/tasks', { title: 'Renew the postage permit', stage: 'todo', due_date: '18/12/2026', repeat: 'yearly', notice_weeks: '6' });
  await page.goto(base + '/tasks/yearly');
  await ready(page);
  await page.screenshot({ path: 'reports/screens/tasks-yearly.png', fullPage: true });

  // Questions: the list, and one question with answers.
  for (const [text, details] of [
    ['Where do we keep the spare toner?', 'The printer on the first floor is out.'],
    ['Who has the key to the supply cupboard?', ''],
    ['Is the office open on Black Friday?', 'Some of us would like to book the day off.'],
  ]) {
    await post(page, base, '/questions', { text, details });
  }
  await page.goto(base + '/questions');
  await ready(page);
  await page.locator('.q-row', { hasText: 'Where do we keep' }).getByRole('link').click();
  await page.getByLabel('Add an answer').fill('In the bottom drawer of the grey cabinet.');
  await page.getByRole('button', { name: 'Add answer' }).click();
  await ready(page);
  await page.getByLabel('Add an answer').fill('Dana has ordered more; it arrives on Tuesday.');
  await page.getByRole('button', { name: 'Add answer' }).click();
  await ready(page);
  await page.screenshot({ path: 'reports/screens/question.png', fullPage: true });
  await page.getByRole('button', { name: 'Mark answered' }).click();
  await ready(page);
  await page.goto(base + '/questions');
  await ready(page);
  await page.screenshot({ path: 'reports/screens/questions.png', fullPage: true });

  // The Briefing with Thanksgiving this week and the waiting questions.
  await page.goto(base + '/briefing');
  await ready(page);
  await page.screenshot({ path: 'reports/screens/briefing-holidays-questions.png', fullPage: true });
});
