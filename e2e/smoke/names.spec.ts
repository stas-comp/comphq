import { expect, test } from '../helpers/fixtures';
import { ready } from '../helpers/ready';

// Restart sub-test only (PLAN.md P1-38): proves a person's name survives
// a container restart/recreate (SPEC gate 1.39's "data intact" — People
// is the one piece of Phase 1 content e2e/smoke/basic.spec.ts's
// @seed/@verify pair doesn't already cover). A fixed name is safe here,
// unlike basic.spec.ts's deliberately-random one: these two tests are
// each run exactly once per container-test.sh execution, with no repeat
// @seed against the same data folder the way the upgrade/rollback
// sub-test replays @seed twice. Tagged @names-seed/@names-verify rather
// than reusing @seed/@verify so container-test.sh's other grep-filtered
// smoke runs (the upgrade/rollback sub-test) never happen to pick these
// up too — neither literal substring "@seed" nor "@verify" appears in
// either tag.
const PERSON_NAME = 'Restart Test Person';

test('@names-seed adds a fixed-name person', async ({ page, server }) => {
  await page.goto(server.baseURL + '/who');
  await page.click('#add-name-link');
  await page.fill('#add-name-input', PERSON_NAME);
  await page.click('#add-name-form button[type="submit"]');
  await ready(page);
});

test('@names-verify the person is still in the picker', async ({ page, server }) => {
  await page.goto(server.baseURL + '/who');
  await ready(page);
  await expect(page.locator('.person-button', { hasText: PERSON_NAME })).toHaveCount(1);
});
