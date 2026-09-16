import { expect, test } from '../helpers/fixtures';

// Only meaningful under deploy/test/offline.override.yaml, which is the
// only place that sets COMPHQ_TEST_MODE=1 on a running container (so
// GET /__test/egress exists at all) and puts the app on an internal-only
// network (so the request it makes actually fails). Not tagged @seed,
// @verify or @verify-prev, so the upgrade/rollback sub-test's grep-filtered
// runs — against a normally-networked container, where this would either
// 404 or succeed — never pick it up.
test('the app answers healthz but cannot reach the internet', async ({ page, server }) => {
  const health = await page.request.get(server.baseURL + '/healthz');
  expect(health.ok()).toBe(true);

  const egress = await page.request.get(server.baseURL + '/__test/egress');
  expect(egress.status()).not.toBe(200);
});
