import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { exportCSV } from '../helpers/zip';

// Articles and COSMOS places on a job across an upgrade and a rollback (SPEC
// gate 8.21, B14.2, D-105): a job with a linked article and a place is made
// under the new build; the container is rolled back to 1.5.0, which has never
// heard of them and must carry on as if they weren't there; upgraded again,
// both are still on the job.
//
// What is checked is read through the endpoints, so the same assertions hold
// whatever the screen looks like: the picker leaves out an article that is
// already linked, a second add of the same folder is refused, and Export
// everything lists both.
//
// Tagged @refs-seed / @refs-rolledback / @refs-verify-upgraded (none a
// substring of another) for container-test.sh's phases. Titles are fixed
// because a later phase has to find them; the person is not.
const JOB = 'Smoke References Job';
const CATEGORY = 'Smoke References Category';
const ARTICLE = 'Smoke References Article';
const FOLDER = '\\\\NAS\\COSMOS\\Smoke\\References';

type Page = import('@playwright/test').Page;

async function jobID(page: Page, baseURL: string): Promise<string> {
  await page.goto(baseURL + '/tasks/board?q=' + encodeURIComponent(JOB));
  await ready(page);
  const links = page.locator('.task-card', { hasText: JOB }).locator('.task-card-title a');
  expect(await links.count(), `${JOB} should be on the Board exactly once`).toBe(1);
  return /\/tasks\/(\d+)/.exec((await links.first().getAttribute('href'))!)![1];
}

test('@refs-seed a job with an article and a COSMOS place', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const post = async (path: string, form: Record<string, string>) => {
    const res = await page.request.post(server.baseURL + path, { form, headers: { origin: server.baseURL }, maxRedirects: 0 });
    expect(res.status(), path).toBeLessThan(400);
    return res;
  };
  await post('/kb/categories', { name: CATEGORY });
  await page.goto(server.baseURL + '/kb/new');
  const categoryID = (await page.locator('select[name="category_id"] option', { hasText: CATEGORY }).getAttribute('value'))!;
  const saved = await page.request.post(server.baseURL + '/kb/articles', {
    form: { title: ARTICLE, category_id: categoryID, body_html: '<p>Smoke.</p>' },
    headers: { origin: server.baseURL },
  });
  expect(saved.ok()).toBe(true);
  const articleID = /\/kb\/articles\/(\d+)/.exec(saved.url())![1];

  await post('/tasks', { title: JOB, stage: 'todo' });
  const id = await jobID(page, server.baseURL);
  await post(`/tasks/${id}/articles`, { article_id: articleID });
  await post(`/tasks/${id}/places`, { name: 'Smoke place', path: FOLDER });
});

async function assertBothStillThere(page: Page, baseURL: string) {
  const id = await jobID(page, baseURL);
  // The picker leaves out an article that is already linked to the job.
  const picker = await page.request.get(`${baseURL}/tasks/articles.json?q=${encodeURIComponent(ARTICLE)}&task=${id}`);
  expect((await picker.json()).results).toEqual([]);
  const unfiltered = await page.request.get(`${baseURL}/tasks/articles.json?q=${encodeURIComponent(ARTICLE)}`);
  expect((await unfiltered.json()).results).toHaveLength(1);
  // A second add of the same folder is refused: the first is still on the job.
  const again = await page.request.post(`${baseURL}/tasks/${id}/places`, {
    form: { name: 'Again', path: FOLDER.toLowerCase() },
    headers: { origin: baseURL },
  });
  expect(await again.text()).toContain('That folder is already on this job.');
  // And Export everything lists both.
  const articles = await exportCSV(page, baseURL, 'job-articles.csv');
  expect(articles.find((r) => r[0] === JOB && r[1] === ARTICLE && r[4] === '')).toBeTruthy();
  const places = await exportCSV(page, baseURL, 'job-places.csv');
  expect(places.find((r) => r[0] === JOB && r[1] === 'Smoke place' && r[2] === FOLDER && r[5] === '')).toBeTruthy();
}

// "1.5.0" >= "1.6.0": compares dotted release numbers part by part.
function atLeast(version: string, min: string): boolean {
  const a = version.split('.').map(Number);
  const b = min.split('.').map(Number);
  for (let i = 0; i < Math.max(a.length, b.length); i++) {
    const d = (a[i] ?? 0) - (b[i] ?? 0);
    if (d !== 0) return d > 0;
  }
  return true;
}

// Run against the PREVIOUS release's app. container-test.sh passes its version
// as COMPHQ_PREV_VERSION. A release before 1.6.0 has never heard of articles
// and places on jobs and must open the job as if they weren't there (gate
// 8.21); a later one still has them. Run locally, with no previous release,
// the same server answers, so only the job's page is checked.
test('@refs-rolledback the previous release opens the job without them', async ({ page, server }) => {
  const prev = process.env.COMPHQ_PREV_VERSION ?? '';
  const prevKnowsRefs = prev === '' || atLeast(prev, '1.6.0');
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const id = await jobID(page, server.baseURL);
  await page.goto(`${server.baseURL}/tasks/${id}`);
  await ready(page);
  await expect(page.locator('.task-details-form')).toBeVisible();
  if (!prevKnowsRefs) {
    // It has no endpoint for them either.
    const res = await page.request.post(`${server.baseURL}/tasks/${id}/places`, {
      form: { name: 'x', path: FOLDER },
      headers: { origin: server.baseURL },
    });
    expect([404, 405]).toContain(res.status());
  }
});

test('@refs-verify-upgraded the article and the place are still on the job after the upgrade', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await assertBothStillThere(page, server.baseURL);
});
