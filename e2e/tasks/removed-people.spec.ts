import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

// SPEC gate 2.11: a removed person no longer appears when assigning,
// but still shows on their old tasks, marked "(removed)" — on the
// board card's avatar and on the task details page's people select.
test('gate 2.11: a removed person no longer appears when assigning, but still shows on their old tasks marked (removed)', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  const nameB = await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);
  const cookies = await otherPage.context().cookies();
  const personBID = cookies.find((c) => c.name === 'comphq_person')?.value;
  expect(personBID).toBeTruthy();
  await otherContext.close();

  // The people multi-select is only refreshed from a fresh page load.
  await page.reload();
  await ready(page);

  const title = uniqueName('Assigned to B');
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-people', { label: nameB });
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  const avatar = page.locator('.task-card', { hasText: title }).locator('.av');
  await expect(avatar).toHaveAttribute('title', nameB);

  const res = await page.request.post(server.baseURL + '/__test/people/deactivate', {
    form: { person_id: personBID! },
    headers: { origin: server.baseURL },
  });
  expect(res.ok()).toBe(true);

  await page.reload();
  await ready(page);

  const peopleOptions = await page.locator('#new-task-people option').allTextContents();
  expect(peopleOptions).not.toContain(nameB);

  await expect(avatar).toHaveAttribute('title', `${nameB} (removed)`);

  await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').click();
  await ready(page);
  const removedOption = page.locator('#details-people option', { hasText: `${nameB} (removed)` });
  await expect(removedOption).toHaveCount(1);
  const isSelected = await removedOption.evaluate((el) => (el as HTMLOptionElement).selected);
  expect(isSelected).toBe(true);
});
