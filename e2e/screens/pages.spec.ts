import { test } from '../helpers/fixtures';
import { pages } from '../helpers/page-registry';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

test.use({ viewport: { width: 1366, height: 768 } });

for (const p of pages) {
  test(`screenshot: ${p.name}`, async ({ page, server }) => {
    if (p.needsPerson) {
      await signInAsNewPerson(page, server.baseURL, p.path);
    } else {
      await page.goto(server.baseURL + p.path);
    }
    await ready(page);
    await page.screenshot({ path: `reports/screens/${p.name}.png`, fullPage: true });
  });
}
