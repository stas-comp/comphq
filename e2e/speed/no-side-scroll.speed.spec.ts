import { test } from './fixtures';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { pages } from '../helpers/page-registry';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// SPEC gate 7.50: with the test library loaded (2,000 jobs, 10 people, 500
// articles), no page scrolls sideways at 1024 × 700 or 1920 × 1080,
// measured on the whole page (D-97).
for (const p of pages) {
  test(`7.50 no sideways scroll with the test library loaded: ${p.name}`, async ({ page, server }) => {
    if (p.needsPerson) {
      await signInAsNewPerson(page, server.baseURL, p.path);
    } else {
      await page.goto(server.baseURL + p.path);
    }
    await ready(page);
    await expectNoSideScroll(page);
  });
}
