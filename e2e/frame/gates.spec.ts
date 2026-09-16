import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// SPEC gate 1.07: every page shows the same frame (wordmark, the four
// sections, Settings, search box, current name), with the current
// section marked. The route list comes from a test-mode-only endpoint,
// so this test covers every real route without hard-coding it twice.
test('every registered route shows the frame, with the current section marked', async ({ page, server }) => {
  const name = await signInAsNewPerson(page, server.baseURL, '/');

  const res = await page.request.get(server.baseURL + '/__test/routes');
  const routes: string[] = await res.json();
  expect(routes.length).toBeGreaterThan(0);

  for (const route of routes) {
    await page.goto(server.baseURL + route);
    await ready(page);

    await expect(page.locator('.wordmark')).toHaveText('Comp HQ');
    for (const section of ['Briefing', 'Knowledge Base', 'Tasks', 'Calendar', 'Settings']) {
      await expect(page.locator('.sidebar nav a', { hasText: section })).toHaveCount(1);
    }
    await expect(page.locator(`.sidebar nav a[href="${route}"]`)).toHaveAttribute('aria-current', 'page');
    await expect(page.locator('.topbar')).toContainText(`You: ${name}`);
    await expect(page.locator('#search-box')).toHaveCount(1);
  }
});

// SPEC gate 1.08: each placeholder section shows its own "Coming soon"
// heading with status 200, never an error.
test('placeholder sections show Coming soon with status 200', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/');

  for (const [path, title] of [
    ['/briefing', 'Briefing'],
    ['/tasks', 'Tasks'],
    ['/calendar', 'Calendar'],
  ] as const) {
    const res = await page.request.get(server.baseURL + path);
    expect(res.status()).toBe(200);
    await page.goto(server.baseURL + path);
    await ready(page);
    await expect(page.locator('main h1')).toHaveText(title);
    await expect(page.locator('main')).toContainText('Coming soon');
  }
});

// SPEC gate 1.11: an unknown address shows a friendly "Page not found"
// inside the normal frame, with a working link to the Knowledge Base.
test('an unknown address shows Page not found inside the frame', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/');

  const res = await page.request.get(server.baseURL + '/this-page-does-not-exist');
  expect(res.status()).toBe(404);

  await page.goto(server.baseURL + '/this-page-does-not-exist');
  await ready(page);
  await expect(page.locator('.wordmark')).toHaveText('Comp HQ'); // still inside the frame
  await expect(page.locator('main h1')).toHaveText('Page not found');

  await page.click('main a[href="/kb"]');
  await ready(page);
  expect(new URL(page.url()).pathname).toBe('/kb');
});
