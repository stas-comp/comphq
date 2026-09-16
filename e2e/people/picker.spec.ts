import { expect, test } from '../helpers/fixtures';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

async function addName(page: import('@playwright/test').Page, baseURL: string, name: string) {
  await page.goto(baseURL + '/who?next=%2F');
  await ready(page);
  await page.click('#add-name-link');
  await page.fill('#add-name-input', name);
  await page.click('#add-name-form button[type="submit"]');
}

// SPEC gate 1.01: alphabetical buttons.
test('names appear as buttons in alphabetical order', async ({ page, server }) => {
  const suffix = uniqueName('').slice(1); // a shared random tail so these three sort only against each other
  const nameA = `AAA-${suffix}`;
  const nameM = `MMM-${suffix}`;
  const nameZ = `ZZZ-${suffix}`;

  await addName(page, server.baseURL, nameZ);
  await addName(page, server.baseURL, nameA);
  await addName(page, server.baseURL, nameM);

  await page.goto(server.baseURL + '/who');
  await ready(page);

  const buttons = await page.locator('.person-button').allTextContents();
  const indexA = buttons.indexOf(nameA);
  const indexM = buttons.indexOf(nameM);
  const indexZ = buttons.indexOf(nameZ);
  expect(indexA).toBeGreaterThanOrEqual(0);
  expect(indexM).toBeGreaterThan(indexA);
  expect(indexZ).toBeGreaterThan(indexM);
});

// SPEC gate 1.02: picking a name persists across a fresh page load in the
// same browser context, and the top bar shows "You: name".
test('a picked name persists across a new page in the same context', async ({ page, server }) => {
  const name = uniqueName('Test');
  await addName(page, server.baseURL, name);
  expect(new URL(page.url()).pathname).toBe('/kb'); // "/" redirects to "/kb" (P1-14)
  await expect(page.locator('.topbar')).toContainText(`You: ${name}`);

  await page.goto(server.baseURL + '/');
  await ready(page);
  expect(new URL(page.url()).pathname).toBe('/kb'); // no bounce back to /who
});

// SPEC gate 1.03: "Change" (simulated by visiting /who directly, since the
// frame's own link lands in P1-14) shows the picker again, and choosing a
// different name replaces the current one.
test('visiting the picker again allows switching to a different name', async ({ page, server }) => {
  const first = uniqueName('First');
  const second = uniqueName('Second');

  await addName(page, server.baseURL, first);
  const cookiesAfterFirst = await page.context().cookies();
  const personCookieAfterFirst = cookiesAfterFirst.find((c) => c.name === 'comphq_person');

  await addName(page, server.baseURL, second);
  const cookiesAfterSecond = await page.context().cookies();
  const personCookieAfterSecond = cookiesAfterSecond.find((c) => c.name === 'comphq_person');

  expect(personCookieAfterSecond?.value).toBeTruthy();
  expect(personCookieAfterSecond?.value).not.toBe(personCookieAfterFirst?.value);
});

// SPEC gate 1.04: a name added on one computer (browser context) shows up
// in the picker on another.
test("a name added in one context appears in another context's picker", async ({ page, server, browser }) => {
  const name = uniqueName('CrossContext');
  await addName(page, server.baseURL, name);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await otherPage.goto(server.baseURL + '/who');
  await ready(otherPage);
  await expect(otherPage.locator('.person-button', { hasText: name })).toHaveCount(1);
  await otherContext.close();
});

// SPEC gate 1.06: removing (deactivating) the current person shows the
// picker on the next page. Uses the same store helper Settings → People
// (P1-15) will call, via a test-mode-only route (docs/decisions.md).
test('deactivating the current person shows the picker next time', async ({ page, server }) => {
  const name = uniqueName('Removed');
  await addName(page, server.baseURL, name);

  const cookies = await page.context().cookies();
  const personID = cookies.find((c) => c.name === 'comphq_person')?.value;
  expect(personID).toBeTruthy();

  const res = await page.request.post(server.baseURL + '/__test/people/deactivate', {
    form: { person_id: personID! },
    headers: { origin: server.baseURL },
  });
  expect(res.ok()).toBe(true);

  await page.goto(server.baseURL + '/');
  await ready(page);
  expect(new URL(page.url()).pathname).toBe('/who');
});

test('adding a duplicate name (case-insensitive) shows a plain message', async ({ page, server }) => {
  const name = uniqueName('Dup');
  await addName(page, server.baseURL, name);

  await page.goto(server.baseURL + '/who');
  await ready(page);
  await page.click('#add-name-link');
  await page.fill('#add-name-input', name.toUpperCase());
  await page.click('#add-name-form button[type="submit"]');

  await expect(page.locator('.message')).toContainText('already');
});

test('the picker works with keyboard only', async ({ page, server }) => {
  await page.goto(server.baseURL + '/who');
  await ready(page);

  // Tab to the "My name isn't here" control and activate it with the
  // keyboard, then fill and submit the add-name form without a mouse.
  const link = page.locator('#add-name-link');
  await link.focus();
  await page.keyboard.press('Enter');

  const name = uniqueName('Keyboard');
  await page.locator('#add-name-input').fill(name);
  await page.keyboard.press('Enter');

  await page.waitForURL('**/kb');
  expect(new URL(page.url()).pathname).toBe('/kb'); // "/" (the default next) redirects to "/kb" (P1-14)
});
