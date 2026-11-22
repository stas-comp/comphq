import { test } from '../helpers/fixtures-today';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

test.use({ viewport: { width: 1366, height: 768 } });

// Screenshots of the holiday chips for the owner's wording check (O8.5, gates
// 8.60-8.65). Each is @fresh for its own fixed date.

test.describe('November 2026', () => {
  test.use({ today: '2026-11-14' });
  test('@fresh screenshot: calendar-holidays-nov-2026 and briefing-holidays', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2026-11');
    await ready(page);
    await page.screenshot({ path: 'reports/screens/calendar-holidays-nov-2026.png', fullPage: true });
    await page.goto(server.baseURL + '/briefing');
    await ready(page);
    await page.screenshot({ path: 'reports/screens/briefing-holidays.png', fullPage: true });
  });
});

test.describe('December 2027', () => {
  test.use({ today: '2027-12-11' });
  test('@fresh screenshot: calendar-holidays-dec-2027', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2027-12');
    await ready(page);
    await page.screenshot({ path: 'reports/screens/calendar-holidays-dec-2027.png', fullPage: true });
  });
});
