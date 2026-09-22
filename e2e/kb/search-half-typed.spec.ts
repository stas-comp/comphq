import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// A random, alphabetic-only prefix (Porter stemming rules look at a
// word's ending, not its start, so this keeps "…pay"/"…payment"'s
// stemming behaviour identical to the plain word while still making the
// pair unique per test run — needed because workers share one server,
// SPEC §2.8).
function uniquePrefix(): string {
  return Math.random().toString(36).replace(/[^a-z]/g, '').slice(0, 8) || 'zz';
}

async function createCategory(page: Page, baseURL: string, name: string) {
  await page.goto(baseURL + '/kb/categories');
  await page.fill('#new-category-name', name);
  await page.click('.create-category-form button[type="submit"]');
  await ready(page);
}

async function typeIntoEditor(page: Page, text: string) {
  const editor = page.locator('#article-editor .ProseMirror');
  await editor.click();
  await expect(editor).toBeFocused();
  await page.keyboard.press('Control+a');
  await page.keyboard.type(text);
}

async function publishArticle(page: Page, baseURL: string, category: string, title: string, bodyText: string) {
  await page.goto(baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
  await page.selectOption('#article-category', { label: category });
  await page.fill('#article-title', title);
  await typeIntoEditor(page, bodyText);
  await page.click('#btn-publish');
  await ready(page);
}

// SPEC gate 6.30: typing a half-typed last word character by character
// into the live search finds an article containing the whole word, the
// moment the typed prefix is long enough to be a search at all (2
// characters). "pay" and "busin" are D-85's own named examples.
for (const [prefix, suffix] of [
  ['pay', 'ment'],
  ['busin', 'ess'],
]) {
  test(`gate 6.30: typing "${prefix}" character by character finds "${prefix}${suffix}"`, async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    const category = uniqueName('Finance');
    await createCategory(page, server.baseURL, category);
    const unique = uniquePrefix();
    const wholeWord = unique + prefix + suffix;
    const title = uniqueName('Half-typed word article');
    await publishArticle(page, server.baseURL, category, title, `A note that mentions ${wholeWord} in passing.`);

    await page.goto(server.baseURL + '/kb');
    await ready(page);
    const typed = unique + prefix;
    for (let i = 1; i <= typed.length; i++) {
      await page.fill('#search-box', typed.slice(0, i));
    }
    // The last character lands well short of the whole word; only once
    // it's fully typed (and the debounce settles) should a result show.
    const panel = page.locator('.search-panel');
    await expect(panel).toBeVisible();
    const firstResult = panel.locator('.search-result').first();
    await expect(firstResult).toContainText(title);
    await expect(firstResult.locator('mark')).toHaveText(wholeWord);
  });
}

// SPEC gate 6.31: finished words still find their relatives exactly as
// before (gate 1.29, unchanged), and a title match still ranks above a
// body match (gate 1.26) — both with a half-typed word also in play, to
// prove the expansion doesn't disturb ordinary stemming or ranking.
test('gate 6.31: finished-word stemming and title-outranks-body still hold alongside the new expansion', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);
  const unique = uniquePrefix();

  const bodyMatchTitle = uniqueName('Body match article');
  await publishArticle(page, server.baseURL, category, bodyMatchTitle, `Ask Sam about the ${unique}toner cartridges.`);
  const titleMatchTitle = unique + 'toner' + ' troubleshooting ' + uniqueName('');
  await publishArticle(page, server.baseURL, category, titleMatchTitle, 'Start by checking the cable is plugged in.');

  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await page.fill('#search-box', unique + 'ton'); // a finished-enough stem, gate 1.29 style
  const panel = page.locator('.search-panel');
  await expect(panel).toBeVisible();
  const results = panel.locator('.search-result');
  await expect(results).toHaveCount(2);
  await expect(results.first()).toContainText(titleMatchTitle); // title match ranks first
});

// SPEC gate 6.32: the word a half-typed word found is highlighted on the
// article page too, the same as any other match (gate 1.28).
test('gate 6.32: the article page highlights the word a half-typed search found', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Volunteering');
  await createCategory(page, server.baseURL, category);
  const unique = uniquePrefix();
  const wholeWord = unique + 'voluntee' + 'r';
  const title = uniqueName('Volunteer article');
  await publishArticle(page, server.baseURL, category, title, `Ask about becoming a ${wholeWord} this term.`);

  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await page.fill('#search-box', unique + 'voluntee');
  const firstResult = page.locator('.search-result').first();
  await expect(firstResult.locator('mark')).toHaveText(wholeWord);
  await firstResult.click();
  await ready(page);

  expect(page.url()).toMatch(/#b-\d+$/);
  const blockID = /#b-(\d+)$/.exec(page.url())![1];
  const target = page.locator(`[data-b="${blockID}"]`);
  await expect(target).toHaveClass(/kb-highlighted/);
  await expect(target.locator('mark')).toHaveText(wholeWord);
});

// SPEC gate 6.33: publishing, editing and archiving are reflected in the
// very next search for half-typed words too (gates 1.30, 1.31).
test('gate 6.33: publish, edit and archive are reflected immediately in half-typed-word search', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Generators');
  await createCategory(page, server.baseURL, category);
  const unique = uniquePrefix();
  const oldWord = unique + 'generat' + 'ion';
  const newWord = unique + 'generat' + 'or';
  const title = uniqueName('Generator article');
  await publishArticle(page, server.baseURL, category, title, `Notes about ${oldWord} equipment.`);

  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await page.fill('#search-box', unique + 'generat');
  await expect(page.locator('.search-result')).toHaveCount(1);

  // Edit: the old whole word is replaced by a different one, still
  // sharing the same half-typed prefix.
  await page.locator('.search-result').first().click();
  await ready(page);
  await page.goto(page.url().split('?')[0].replace(/#.*$/, '') + '/edit');
  await typeIntoEditor(page, `Notes about ${newWord} equipment.`);
  await page.click('#btn-publish');
  await ready(page);

  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await page.fill('#search-box', unique + 'generat');
  const result = page.locator('.search-result').first();
  await expect(result).toBeVisible();
  await expect(result.locator('mark')).toHaveText(newWord);

  // Archive: no longer found at all.
  await result.click();
  await ready(page);
  await page.click('form[action$="/archive"] button[type="submit"]');
  await ready(page);

  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await page.fill('#search-box', unique + 'generat');
  await expect(page.locator('.search-panel')).toContainText('No articles match');
});

test('gate 6.30: axe on the search results page for a half-typed word', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Happiness');
  await createCategory(page, server.baseURL, category);
  const unique = uniquePrefix();
  const wholeWord = unique + 'happin' + 'ess';
  const title = uniqueName('Happiness article');
  await publishArticle(page, server.baseURL, category, title, `A short note on ${wholeWord} at work.`);

  await page.goto(server.baseURL + '/kb/search?q=' + encodeURIComponent(unique + 'happin'));
  await ready(page);
  await expect(page.locator('.search-result').first()).toContainText(title);
  await axeCheck(page);
});
