import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

async function addTask(page: Page, title: string, personName?: string): Promise<void> {
  await page.fill('#new-task-title', title);
  if (personName) {
    await page.selectOption('#new-task-people', { label: personName });
  }
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

function cardTitles(page: Page) {
  return page.locator('.task-card-title').allTextContents();
}

// SPEC gate 2.07: "My tasks" shows only cards assigned to the current
// person; choosing a person shows only theirs; typing a word shows only
// cards with that word in the title.
test('gate 2.07: My tasks, a chosen person, and a word each narrow the board', async ({ page, server, browser }) => {
  const nameA = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  const nameB = await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);
  await otherContext.close();

  // Reload so the people multi-select picks up the just-created nameB.
  await page.reload();
  await ready(page);

  const titleA = uniqueName('Task for A');
  const titleB = uniqueName('Task for B');
  const titleUnassigned = uniqueName('Unassigned task');
  await addTask(page, titleA, nameA);
  await addTask(page, titleB, nameB);
  await addTask(page, titleUnassigned);

  const all = await cardTitles(page);
  expect(all).toEqual(expect.arrayContaining([titleA, titleB, titleUnassigned]));

  // "My tasks".
  await page.check('#filter-mine');
  await page.click('.task-filter-form button[type="submit"]');
  await ready(page);
  let titles = await cardTitles(page);
  expect(titles).toContain(titleA);
  expect(titles).not.toContain(titleB);
  expect(titles).not.toContain(titleUnassigned);

  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);

  // A chosen person.
  await page.selectOption('#filter-person', { label: nameB });
  await page.click('.task-filter-form button[type="submit"]');
  await ready(page);
  titles = await cardTitles(page);
  expect(titles).toContain(titleB);
  expect(titles).not.toContain(titleA);
  expect(titles).not.toContain(titleUnassigned);

  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);

  // A word in the title.
  await page.fill('#filter-q', titleUnassigned);
  await page.click('.task-filter-form button[type="submit"]');
  await ready(page);
  titles = await cardTitles(page);
  expect(titles).toContain(titleUnassigned);
  expect(titles).not.toContain(titleA);
  expect(titles).not.toContain(titleB);

  // Clear filter shows everything again.
  await page.locator('.task-filter-form a', { hasText: 'Clear filter' }).click();
  await ready(page);
  titles = await cardTitles(page);
  expect(titles).toEqual(expect.arrayContaining([titleA, titleB, titleUnassigned]));
});
