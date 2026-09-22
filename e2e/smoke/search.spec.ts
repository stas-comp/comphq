import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// SPEC gate 6.34, B12.5, D-85: kb_search_words is rebuilt from kb_search
// at every start-up, so an article published or edited while rolled
// back to v1.2.0 (which writes kb_search, but has never heard of
// kb_search_words) is still found by a half-typed word the moment the
// newer build starts again — no separate migration step, no manual
// rebuild. @search-seed-rolledback runs in container-test.sh's phase 3
// (rolled back to the previous release, SPEC P1-12's harness), using
// only the plain publish flow that has existed since Phase 1;
// @search-verify-upgraded runs in phase 4, after upgrading again.
//
// CATEGORY_NAME and ARTICLE_TITLE are fixed, not uniqueName()'d
// (e2e/smoke/basic.spec.ts's own reasoning): @search-verify-upgraded, a
// separate container-test.sh phase, has to know in advance what to look
// for. "payment"/"pay" is D-85's own probed example, not a made-up
// word: kb_search's porter stemmer turns the typed "pay" into "pai" (a
// real Porter step-1c rule, an "AY" ending after a consonant becomes
// "AI"), which is not a prefix of "payment" — so unlike a coincidental
// pick, this pair only finds the article at all when kb_search_words'
// unstemmed lookup is actually working, which is the one thing this
// test needs to prove.
const CATEGORY_NAME = 'Smoke Test Search Category';
const ARTICLE_TITLE = 'Smoke Test Search Article';
const WHOLE_WORD = 'payment';
const TYPED_PREFIX = 'pay';

test('@search-seed-rolledback publish an article while rolled back to the previous release', async ({ page, server }) => {
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
  await page.keyboard.type(`A note about the ${WHOLE_WORD} schedule, published while rolled back.`);
  await page.click('#btn-publish');
  await ready(page);

  await expect(page.locator('h1')).toHaveText(ARTICLE_TITLE);
});

test('@search-verify-upgraded a half-typed word finds the article published while rolled back', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await ready(page);
  await page.fill('#search-box', TYPED_PREFIX);
  const firstResult = page.locator('.search-result').first();
  await expect(firstResult).toContainText(ARTICLE_TITLE);
  await expect(firstResult.locator('mark')).toHaveText(WHOLE_WORD);
});
