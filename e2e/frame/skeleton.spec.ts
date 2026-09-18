import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

test('root shows the Briefing and is ready', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/');
  await ready(page);
  await expect(page.locator('body[data-ready]')).toHaveCount(1);
  expect(new URL(page.url()).pathname).toBe('/'); // "/" is the Briefing (SPEC gate 3.01, P3-02)
  await expect(page.locator('.briefing-headline')).toBeVisible();
  await expect(page).toHaveTitle(/Comp HQ/);
});
