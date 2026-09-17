import { expect, test } from '../helpers/fixtures';
import { loadFixtureFile, pasteFile } from '../helpers/paste';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// The smoke suite runs against a real, already-running server (inside a
// container, over BASE_URL — see e2e/helpers/fixtures.ts) for the offline
// and upgrade/rollback harnesses (P1-12). @seed/@verify/@verify-prev prove
// gate 1.42 for real: an upgrade (and a rollback) must not lose or corrupt
// Knowledge Base content (P1-27's test library is deliberately not used
// here — it's disposable speed-test fixture data, regenerated on demand,
// not the kind of real, once-created content an upgrade must protect).
//
// ARTICLE_TITLE is fixed rather than uniqueName()'d: @verify-prev, run from
// a later release's own smoke suite against a data folder this exact
// @seed test seeded, has to know in advance what to look for. The person
// name is deliberately NOT fixed (signInAsNewPerson), since @seed runs
// again on the upgraded build against the same data folder in the same
// container-test.sh pass (SPEC gate 1.42's phase 2) — a fixed, already-used
// person name would collide with gate 1.06's duplicate-name refusal, where
// a fixed article title or category name doesn't collide with anything.
const CATEGORY_NAME = 'Smoke Test Category';
const ARTICLE_TITLE = 'Smoke Test Article';
const ARTICLE_BODY_V1 = 'Original smoke-test content, seeded before an upgrade.';
const ARTICLE_BODY_V2 = 'A second version, added after the article already existed.';

test('@seed the test library', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');

  await page.goto(server.baseURL + '/kb/categories');
  await page.fill('#new-category-name', CATEGORY_NAME);
  await page.click('.create-category-form button[type="submit"]');
  await ready(page);

  await page.goto(server.baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
  await page.selectOption('#article-category', { label: CATEGORY_NAME });
  await page.fill('#article-title', ARTICLE_TITLE);

  const editor = page.locator('#article-editor .ProseMirror');
  await editor.click();
  await page.keyboard.type(ARTICLE_BODY_V1);

  const png = loadFixtureFile('images/tiny.png', 'image/png');
  await pasteFile(page, '#article-editor .ProseMirror', png);
  await expect(page.locator('#article-editor img')).toHaveCount(1, { timeout: 3000 });

  await page.click('#btn-publish');
  await ready(page);

  // An edit, appending rather than replacing, so the pasted image (and
  // the first version's text) survives into the current, latest version —
  // "the article opens with its image" checks the article as it stands
  // today, not an old version of it.
  await page.locator('a', { hasText: 'Edit' }).click();
  await page.waitForSelector('body[data-editor-ready]');
  const editEditor = page.locator('#article-editor .ProseMirror');
  await editEditor.click();
  await page.keyboard.press('Control+End');
  await page.keyboard.press('Enter');
  await page.keyboard.type(ARTICLE_BODY_V2);
  await page.click('#btn-publish');
  await ready(page);
});

test('@verify the seeded data is present', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');

  await page.goto(server.baseURL + '/kb');
  await ready(page);
  // .first(): a rerun of @seed against the same data folder (gate 1.42's
  // phase 2, current build) leaves an earlier same-titled article behind
  // too; the most recently updated one — this run's own — sorts first.
  await page.locator('.kb-recent-list a', { hasText: ARTICLE_TITLE }).first().click();
  await ready(page);

  await expect(page.locator('h1')).toHaveText(ARTICLE_TITLE);
  await expect(page.locator('.kb-article-body')).toContainText(ARTICLE_BODY_V1);
  await expect(page.locator('.kb-article-body')).toContainText(ARTICLE_BODY_V2);
  const img = page.locator('.kb-article-body img');
  await expect(img).toHaveCount(1);
  await expect(img).toHaveAttribute('src', /^\/images\/[0-9a-f]+\.png$/);

  await page.locator('a', { hasText: 'History' }).click();
  await ready(page);
  await expect(page.locator('.kb-history-list li')).toHaveCount(2);
});

// Run against the *current* release's smoke suite while pointed at data
// seeded by the *previous* release, to prove an upgrade didn't lose or
// break anything old (SPEC gate 1.42). Read-only: it never creates
// anything, only confirms what an earlier @seed already put there is
// still intact and still opens correctly under the new build.
test('@verify-prev reads data seeded by the previous release', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');

  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await page.locator('.kb-recent-list a', { hasText: ARTICLE_TITLE }).first().click();
  await ready(page);

  await expect(page.locator('h1')).toHaveText(ARTICLE_TITLE);
  await expect(page.locator('.kb-article-body')).toContainText(ARTICLE_BODY_V1);
  await expect(page.locator('.kb-article-body')).toContainText(ARTICLE_BODY_V2);
  const img = page.locator('.kb-article-body img');
  await expect(img).toHaveCount(1);
  await expect(img).toHaveAttribute('src', /^\/images\/[0-9a-f]+\.png$/);

  await page.locator('a', { hasText: 'History' }).click();
  await ready(page);
  await expect(page.locator('.kb-history-list li')).toHaveCount(2);
});
