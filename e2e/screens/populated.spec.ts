import path from 'node:path';
import { expect, test } from '../helpers/fixtures-today';
import { ready } from '../helpers/ready';

test.use({ viewport: { width: 1366, height: 768 }, today: '2026-09-19' });

type Page = import('@playwright/test').Page;

// The registry's own screenshots are of empty screens. These are the same
// screens as they look in use, with the mockup's own example content (a
// Saturday, 19 September 2026), so the owner can hold each one beside
// docs/design/mockup.html (owner check O4.1): the Board, My jobs, Team, the
// Calendar, an article, search results, the editor after an import, and the
// task window in each of its states.

async function signInAs(page: Page, baseURL: string, name: string): Promise<void> {
  await page.goto(baseURL + '/who?next=' + encodeURIComponent('/tasks/board'));
  await page.click('#add-name-link');
  await page.fill('#add-name-input', name);
  await page.click('#add-name-form button[type="submit"]');
  await ready(page);
}

type Job = { title: string; stage: string; size?: string; due?: string; people?: string[]; notes?: string };

async function addJob(page: Page, baseURL: string, job: Job): Promise<void> {
  await page.goto(baseURL + '/tasks/new');
  await ready(page);
  await page.fill('#new-task-title', job.title);
  await page.selectOption('#new-task-stage', job.stage);
  if (job.size) await page.selectOption('#new-task-size', job.size);
  if (job.due) await page.fill('#new-task-due-date', job.due);
  if (job.notes) await page.fill('#new-task-notes', job.notes);
  if (job.people?.length) await page.selectOption('#new-task-people', job.people.map((label) => ({ label })));
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

async function addEvent(page: Page, baseURL: string, e: { title: string; start: string; end?: string; time?: string; endTime?: string; notes?: string }): Promise<void> {
  await page.goto(baseURL + '/calendar/new');
  await ready(page);
  await page.fill('#event-title', e.title);
  await page.fill('#event-start-date', e.start);
  if (e.end) await page.fill('#event-end-date', e.end);
  if (e.time) await page.fill('#event-start-time', e.time);
  if (e.endTime) await page.fill('#event-end-time', e.endTime);
  if (e.notes) await page.fill('#event-notes', e.notes);
  await page.click('.calendar-event-form button[type="submit"]');
  await ready(page);
}

const shot = (page: Page, name: string) => page.screenshot({ path: `reports/screens/${name}.png`, fullPage: true });

test('@fresh screenshots: the screens in use, and the task window in each state', async ({ page, server, browser }) => {
  test.setTimeout(240_000);
  const base = server.baseURL;

  // Sam, on their own computer, so the Team view has more than one person.
  const samCtx = await browser.newContext();
  const samPage = await samCtx.newPage();
  await signInAs(samPage, base, 'Sam');
  await samCtx.close();
  await signInAs(page, base, 'Priya');

  for (const job of [
    { title: 'Volunteer thank-you lunch', stage: 'idea', size: 'M' },
    { title: 'Label the storage shelves', stage: 'idea', size: 'S', people: ['Sam'] },
    { title: 'Print exam papers', stage: 'todo', size: 'M', due: '23/09/2026', people: ['Priya', 'Sam'], notes: 'Three sets per room, stapled.\nCheck the copier has paper before you start.' },
    { title: 'Order printer toner (HP 305A)', stage: 'todo', size: 'S', due: '12/09/2026', people: ['Sam'] },
    { title: 'Book piano tuner for concert', stage: 'todo', size: 'M', due: '31/10/2026' },
    { title: 'Post September newsletter', stage: 'doing', size: 'M', due: '25/09/2026', people: ['Sam'] },
    { title: 'Update Christmas card address list', stage: 'doing', size: 'L', due: '17/10/2026', people: ['Priya'] },
    { title: 'Photocopy concert programmes', stage: 'todo', size: 'L', due: '05/12/2026' },
    { title: 'Move HelpScout articles', stage: 'done', size: 'M' },
    { title: 'Fix the kitchen tap sign', stage: 'done', size: 'S' },
  ] as Job[]) {
    await addJob(page, base, job);
  }

  // The Board, and the task window over it.
  await page.goto(base + '/tasks/board');
  await ready(page);
  await shot(page, 'tasks-board-populated');
  await page.getByRole('link', { name: '+ Add task' }).click();
  await expect(page.locator('#task-window')).toBeVisible();
  await page.locator('#new-task-title').fill('Order more staples');
  await page.locator('#new-task-notes').fill('The small ones, two boxes.');
  await page.screenshot({ path: 'reports/screens/task-window-new.png' });
  page.once('dialog', (d) => d.accept());
  await page.keyboard.press('Escape');
  await expect(page.locator('#task-window')).toBeHidden();

  await page.locator('.task-card-title a', { hasText: 'Print exam papers' }).click();
  await expect(page.locator('#task-window').locator('.task-read')).toBeVisible();
  await page.screenshot({ path: 'reports/screens/task-window-reading.png' });
  await page.locator('#task-window details.task-history summary').click();
  await page.screenshot({ path: 'reports/screens/task-window-reading-history.png' });
  await page.getByRole('button', { name: 'Edit' }).click();
  await expect(page.locator('#details-title')).toBeVisible();
  await page.screenshot({ path: 'reports/screens/task-window-editing.png' });
  await page.getByRole('button', { name: 'Cancel' }).click();
  await page.getByRole('button', { name: 'Close' }).click();
  await expect(page.locator('#task-window')).toBeHidden();

  // The people menu on a card.
  await page.locator('.task-card', { hasText: 'Book piano tuner' }).locator('.people-menu-trigger').click();
  await expect(page.locator('.people-menu-panel')).toBeVisible();
  await page.screenshot({ path: 'reports/screens/tasks-board-people-menu.png' });
  await page.keyboard.press('Escape');

  // My jobs and Team.
  await page.goto(base + '/tasks');
  await ready(page);
  await shot(page, 'tasks-populated');
  await page.goto(base + '/tasks/team');
  await ready(page);
  await shot(page, 'tasks-team-populated');

  // The Calendar, September 2026.
  for (const e of [
    { title: 'Board meeting', start: '08/09/2026', time: '19:00' },
    { title: 'Office day', start: '19/09/2026' },
    { title: 'Exams', start: '21/09/2026', end: '25/09/2026' },
    { title: 'Rota meeting', start: '23/09/2026', time: '18:30' },
    { title: 'Office day', start: '26/09/2026' },
    { title: 'Harvest fair set-up', start: '03/10/2026' },
  ]) {
    await addEvent(page, base, e);
  }
  await page.goto(base + '/calendar?month=2026-09');
  await ready(page);
  await shot(page, 'calendar-populated');

  // The Knowledge Base: an article, its search results, and the editor after an import.
  await page.goto(base + '/kb/categories');
  await page.fill('#new-category-name', 'Office facts');
  await page.click('.create-category-form button[type="submit"]');
  await ready(page);
  const categoryID = (await page.locator('input[name="name"][value="Office facts"]').getAttribute('id'))!.replace('category-name-', '');
  const res = await page.request.post(base + '/kb/articles', {
    form: {
      category_id: categoryID,
      title: 'Printers: which is where',
      body_html:
        '<p>We have three printers. The big office copier handles anything over 50 pages; the other two are for quick jobs.</p>' +
        '<table><thead><tr><th>Printer</th><th>Where</th><th>Takes</th></tr></thead><tbody>' +
        '<tr><td>Office copier</td><td>Main office, by the window</td><td>Kyocera TK-5240</td></tr>' +
        '<tr><td>Front desk</td><td>Reception</td><td>HP 305A</td></tr>' +
        '<tr><td>Label printer</td><td>Mail room shelf</td><td>Dymo 99010 labels</td></tr></tbody></table>' +
        '<h2>Spares</h2><p>Spare toner lives in the grey cabinet by the kitchen door, top shelf. If you take the last one, add a task to reorder it.</p>',
    },
    headers: { origin: base },
    maxRedirects: 0,
  });
  await page.goto(base + res.headers()['location']);
  await ready(page);
  await shot(page, 'kb-article');
  await page.goto(base + '/kb');
  await ready(page);
  await page.fill('#search-box', 'toner');
  await expect(page.locator('.search-panel .search-result')).toHaveCount(1);
  await page.keyboard.press('ArrowDown');
  await page.screenshot({ path: 'reports/screens/kb-search-open.png', fullPage: true });
  await page.locator('.search-panel .search-result').first().click();
  await ready(page);
  await expect(page.locator('.kb-highlighted').first()).toBeVisible();
  await shot(page, 'kb-article-highlighted');

  await page.goto(base + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
  await page.selectOption('#article-category', { label: 'Office facts' });
  const chooser = page.waitForEvent('filechooser');
  await page.click('#btn-import-word');
  await (await chooser).setFiles(path.join(__dirname, '..', 'fixtures', 'docx', 'sample.docx'));
  await expect(page.locator('#editor-message.imported')).toBeVisible();
  await shot(page, 'kb-editor-imported');

  // The name picker with several names.
  await page.goto(base + '/who');
  await shot(page, 'who-populated');
});
