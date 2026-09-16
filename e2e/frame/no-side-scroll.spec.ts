import { test } from '../helpers/fixtures';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { pages } from '../helpers/page-registry';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// SPEC gate 1.09: no page needs sideways scrolling from 1024×700 to
// 1920×1080. Every registered page type, per SPEC §2.8.
for (const p of pages) {
  test(`no sideways scroll: ${p.name}`, async ({ page, server }) => {
    if (p.needsPerson) {
      await signInAsNewPerson(page, server.baseURL, p.path);
    } else {
      await page.goto(server.baseURL + p.path);
    }
    await ready(page);
    await expectNoSideScroll(page);
  });
}
