import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { idFromHref, myPersonID, seedJob } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gate 8.18 (B14.2, D-105): an article's page lists the jobs that use it,
// and linking is not an edit of the article.

test.describe.configure({ timeout: 60_000 });

async function jobID(page: Page, baseURL: string, title: string): Promise<string> {
  await page.goto(baseURL + '/tasks/board?q=' + encodeURIComponent(title));
  await ready(page);
  return idFromHref((await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').first().getAttribute('href'))!);
}

test('gate 8.18: Jobs using this article lists each job with its column and Finished, leaves out removed jobs, and linking edits nothing', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const me = await myPersonID(page, server.baseURL);
  const base = server.baseURL;
  const tag = uniqueName('use').replace(/[^a-z0-9]/gi, '').toLowerCase();

  await page.request.post(base + '/kb/categories', { form: { name: `Cat ${tag}` }, headers: { origin: base } });
  await page.goto(base + '/kb/new');
  const cat = (await page.locator('select[name="category_id"] option', { hasText: `Cat ${tag}` }).getAttribute('value'))!;
  const title = `Mailing SOP ${tag}`;
  const saved = await page.request.post(base + '/kb/articles', { form: { title, category_id: cat, body_html: '<p>Steps.</p>' }, headers: { origin: base } });
  const articleID = /\/kb\/articles\/(\d+)/.exec(saved.url())![1];

  // With no job linking it, the panel isn't there.
  await page.goto(`${base}/kb/articles/${articleID}`);
  await ready(page);
  await expect(page.locator('.kb-jobs-using')).toHaveCount(0);

  const todo = `Send the mailing ${tag}`;
  const done = `Last year's mailing ${tag}`;
  const gone = `Cancelled mailing ${tag}`;
  await seedJob(page, base, { title: todo, stage: 'todo', personID: me });
  await seedJob(page, base, { title: done, stage: 'done', personID: me });
  await seedJob(page, base, { title: gone, stage: 'todo', personID: me });
  for (const t of [todo, done, gone]) {
    const id = await jobID(page, base, t);
    const linked = await page.request.post(`${base}/tasks/${id}/articles`, { form: { article_id: articleID }, headers: { origin: base } });
    expect(linked.ok()).toBe(true);
  }
  await page.request.post(`${base}/tasks/${await jobID(page, base, gone)}/remove`, { headers: { origin: base } });

  await page.goto(`${base}/kb/articles/${articleID}`);
  await ready(page);
  const panel = page.locator('.kb-jobs-using');
  await expect(panel.getByRole('heading', { name: 'Jobs using this article' })).toBeVisible();
  const rows = panel.locator('.kb-job');
  await expect(rows).toHaveCount(2); // the removed job isn't listed
  await expect(rows.nth(0)).toContainText(todo);
  await expect(rows.nth(0)).toContainText('To do');
  await expect(rows.nth(0)).not.toContainText('Finished');
  await expect(rows.nth(1)).toContainText(done);
  await expect(rows.nth(1)).toContainText('Done');
  await expect(rows.nth(1)).toContainText('Finished');
  await expect(rows.nth(0).getByRole('link', { name: todo })).toHaveAttribute('href', /\/tasks\/\d+$/);
  await axeCheck(page);

  // Linking is not an edit: no new version, and the byline and History are unchanged.
  await page.goto(`${base}/kb/articles/${articleID}/history`);
  await ready(page);
  await expect(page.locator('.kb-history-list li')).toHaveCount(1); // still just the one version
  const updated = await page.request.get(`${base}/kb/articles/${articleID}`);
  expect(await updated.text()).toContain('History (1)');
});

test('gate 8.18: linking leaves Recently updated alone', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const me = await myPersonID(page, server.baseURL);
  const base = server.baseURL;
  const tag = uniqueName('rec').replace(/[^a-z0-9]/gi, '').toLowerCase();
  await page.request.post(base + '/kb/categories', { form: { name: `Cat ${tag}` }, headers: { origin: base } });
  await page.goto(base + '/kb/new');
  const cat = (await page.locator('select[name="category_id"] option', { hasText: `Cat ${tag}` }).getAttribute('value'))!;
  const older = `Older article ${tag}`;
  const newer = `Newer article ${tag}`;
  const a = await page.request.post(base + '/kb/articles', { form: { title: older, category_id: cat, body_html: '<p>x</p>' }, headers: { origin: base } });
  const olderID = /\/kb\/articles\/(\d+)/.exec(a.url())![1];
  await page.request.post(base + '/kb/articles', { form: { title: newer, category_id: cat, body_html: '<p>y</p>' }, headers: { origin: base } });

  const recent = async () => {
    await page.goto(base + '/kb');
    await ready(page);
    const all = await page.locator('.kb-recent-list li').allInnerTexts();
    return all.filter((t) => t.includes(tag));
  };
  const before = await recent();
  expect(before.length).toBe(2);
  expect(before[0]).toContain(newer);

  await seedJob(page, base, { title: `Job ${tag}`, stage: 'todo', personID: me });
  const id = await jobID(page, base, `Job ${tag}`);
  await page.request.post(`${base}/tasks/${id}/articles`, { form: { article_id: olderID }, headers: { origin: base } });
  const after = await recent();
  expect(after).toEqual(before); // the older article did not jump to the top
});
