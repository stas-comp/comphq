import { test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { pages } from '../helpers/page-registry';
import { ready } from '../helpers/ready';

for (const p of pages) {
  test(`axe: ${p.name}`, async ({ page, server }) => {
    await page.goto(server.baseURL + p.path);
    await ready(page);
    await axeCheck(page);
  });
}
