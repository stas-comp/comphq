import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

async function createCategory(page: import('@playwright/test').Page, name: string) {
  await page.fill('#new-category-name', name);
  await page.click('.create-category-form button[type="submit"]');
  await ready(page);
}

function rowFor(page: import('@playwright/test').Page, name: string) {
  return page.locator('.kb-category-row', { has: page.locator(`input[value="${name}"]`) });
}

// SPEC gate 1.12: create, rename, reorder (buttons, no drag needed here),
// and delete (refused with articles, allowed when empty).
test('categories: create, rename, reorder, and delete', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb/categories');
  await ready(page);

  const suffix = uniqueName('').slice(1);
  const nameA = `AAA-${suffix}`;
  const nameB = `BBB-${suffix}`;

  await createCategory(page, nameA);
  await createCategory(page, nameB);

  // Reorder: move B up, so it now comes before A.
  await rowFor(page, nameB).locator('button', { hasText: 'Move up' }).click();
  await ready(page);
  const rows = page.locator('.kb-category-row');
  const allNames = await rows.locator('input[name="name"]').evaluateAll(
    (inputs) => inputs.map((i) => (i as HTMLInputElement).value),
  );
  const indexB = allNames.indexOf(nameB);
  const indexA = allNames.indexOf(nameA);
  expect(indexB).toBeGreaterThanOrEqual(0);
  expect(indexB).toBeLessThan(indexA);

  // Rename.
  const renamed = `${nameA}-renamed`;
  const rowA = rowFor(page, nameA);
  await rowA.locator('input[name="name"]').fill(renamed);
  await rowA.locator('button', { hasText: 'Rename' }).click();
  await ready(page);
  await expect(page.locator(`input[value="${renamed}"]`)).toHaveCount(1);

  // Delete an empty category: succeeds.
  await rowFor(page, renamed).locator('button', { hasText: 'Delete' }).click();
  await ready(page);
  await expect(page.locator(`input[value="${renamed}"]`)).toHaveCount(0);
});

test('deleting a category with articles is refused with a plain explanation', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb/categories');
  await ready(page);

  const name = uniqueName('WithArticle');
  await createCategory(page, name);

  // No article-creation UI exists yet (P1-19); seed one directly through
  // the test-mode-only route so this gate is provable now, the same
  // pattern used for people (P1-13) and container tests (P1-06).
  const row = rowFor(page, name);
  const categoryId = await row.locator('form.rename-form').getAttribute('action');
  const id = categoryId?.match(/\/kb\/categories\/(\d+)\//)?.[1];
  expect(id).toBeTruthy();

  const res = await page.request.post(server.baseURL + '/__test/kb/seed-article', {
    form: { category_id: id! },
    headers: { origin: server.baseURL },
  });
  expect(res.ok()).toBe(true);

  await page.goto(server.baseURL + '/kb/categories');
  await ready(page);
  await rowFor(page, name).locator('button', { hasText: 'Delete' }).click();
  await ready(page);

  await expect(page.locator('.message')).toContainText('This category still has 1 article. Move or archive them first.');
  await expect(page.locator(`input[value="${name}"]`)).toHaveCount(1); // still there
});

// SPEC gate 1.13: the home page shows tiles in sort_order with published
// counts. (The "recently updated" list itself is asserted once P1-19 adds
// real articles, per PLAN.md's own instruction not to write a test marked
// "completed later.")
test('home page shows category tiles with article counts', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb/categories');
  await ready(page);

  const suffix = uniqueName('').slice(1);
  const nameX = `XXX-${suffix}`;
  const nameY = `YYY-${suffix}`;
  await createCategory(page, nameX);
  await createCategory(page, nameY);

  const row = rowFor(page, nameX);
  const action = await row.locator('form.rename-form').getAttribute('action');
  const id = action?.match(/\/kb\/categories\/(\d+)\//)?.[1];
  const res = await page.request.post(server.baseURL + '/__test/kb/seed-article', {
    form: { category_id: id! },
    headers: { origin: server.baseURL },
  });
  expect(res.ok()).toBe(true);

  await page.goto(server.baseURL + '/kb');
  await ready(page);

  const tiles = page.locator('.kb-tile');
  const tileX = tiles.filter({ hasText: nameX });
  const tileY = tiles.filter({ hasText: nameY });
  await expect(tileX).toContainText('1 article');
  await expect(tileY).toContainText('0 articles');

  const allTileText = await tiles.allTextContents();
  expect(allTileText.findIndex((t) => t.includes(nameX))).toBeLessThan(
    allTileText.findIndex((t) => t.includes(nameY)),
  );
});
