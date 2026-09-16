import { expect, test } from '../helpers/fixtures';
import { ready } from '../helpers/ready';

// Real speed measurements (navigation-to-ready, search latency) are added
// in P1-27 once the test-library seeder exists. This placeholder keeps
// `npx playwright test --project=speed` from failing with "No tests
// found" on a full CI dispatch or a release tag before then.
test('placeholder: frame page loads', async ({ page, server }) => {
  await page.goto(server.baseURL + '/');
  await ready(page);
  await expect(page.locator('body[data-ready]')).toHaveCount(1);
});
