import { expect, test } from '../helpers/fixtures';
import { ready } from '../helpers/ready';

test('frame page loads and is ready', async ({ page, server }) => {
  await page.goto(server.baseURL + '/');
  await ready(page);
  await expect(page.locator('body[data-ready]')).toHaveCount(1);
  await expect(page).toHaveTitle('Comp HQ');
});
