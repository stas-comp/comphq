import { expect, test } from '../helpers/fixtures-today';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

type Page = import('@playwright/test').Page;

// SPEC gate 8.65 (B14.5, D-109): holidays in the Saturday Briefing, on the five test
// Saturdays B14.7 names. Each is @fresh: it needs its own COMPHQ_TEST_TODAY.

const thisWeek = (page: Page) => page.locator('section[aria-labelledby="briefing-week"]');
const comingUp = (page: Page) => page.locator('section[aria-labelledby="briefing-coming"]');

async function open(page: Page, baseURL: string) {
  await signInAsNewPerson(page, baseURL, '/briefing');
  await ready(page);
}

test.describe('Sat 7 Nov 2026', () => {
  test.use({ today: '2026-11-07' });
  test('@fresh Thanksgiving is not on the page yet', async ({ page, server }) => {
    await open(page, server.baseURL);
    await expect(page.getByText('Thanksgiving')).toHaveCount(0);
    await expect(thisWeek(page).locator('.briefing-holiday')).toHaveCount(1); // Veterans Day, Wed 11 Nov
    await expect(thisWeek(page)).toContainText('Veterans Day · No mail');
  });
});

test.describe('Sat 14 Nov 2026', () => {
  test.use({ today: '2026-11-14' });
  test('@fresh Thanksgiving is in Coming up, in 12 days, labelled like an event', async ({ page, server }) => {
    await open(page, server.baseURL);
    const card = comingUp(page).locator('.briefing-holiday', { hasText: 'Thanksgiving Day' });
    await expect(card).toHaveCount(1);
    await expect(card).toContainText('Thanksgiving Day · No mail');
    await expect(card.locator('.stamp')).toHaveText('IN 12 DAYS');
    await expect(card.locator('.briefing-holiday-title')).toHaveAttribute('title', /Post offices are closed/);
    await expect(thisWeek(page)).not.toContainText('Thanksgiving');
    await axeCheck(page);
    await expectNoSideScroll(page);
  });
});

test.describe('Sat 21 Nov 2026', () => {
  test.use({ today: '2026-11-21' });
  test('@fresh Thanksgiving is in This week with a HOLIDAY stamp, and Just mine keeps it', async ({ page, server }) => {
    await open(page, server.baseURL);
    const card = thisWeek(page).locator('.briefing-holiday', { hasText: 'Thanksgiving Day' });
    await expect(card).toHaveCount(1);
    await expect(card.locator('.stamp')).toHaveText('HOLIDAY');
    await expect(card).toContainText('Thu 26 Nov 2026');
    await expect(card).not.toContainText('No mail today'); // it is not the Saturday itself
    // Holidays belong to everyone, like events: Just mine doesn't hide them.
    await page.goto(server.baseURL + '/briefing?mine=1');
    await ready(page);
    await expect(thisWeek(page).locator('.briefing-holiday', { hasText: 'Thanksgiving Day' })).toHaveCount(1);
    // It is not a link: there is nothing to open.
    await expect(card.locator('a')).toHaveCount(0);
    await axeCheck(page);
  });
});

test.describe('Sat 3 Jul 2027', () => {
  test.use({ today: '2027-07-03' });
  test('@fresh Sunday 4 July and the no-mail Monday 5 July are both in This week', async ({ page, server }) => {
    await open(page, server.baseURL);
    const cards = thisWeek(page).locator('.briefing-holiday');
    await expect(cards).toHaveCount(2);
    await expect(cards.nth(0)).toContainText('Independence Day · No mail');
    await expect(cards.nth(1)).toContainText('Independence Day (observed) · No mail');
    await expect(cards.nth(0).locator('.stamp')).toHaveText('HOLIDAY');
  });
});

test.describe('Sat 25 Dec 2027', () => {
  test.use({ today: '2027-12-25' });
  test('@fresh the Saturday itself is a no-mail day, and This week says so', async ({ page, server }) => {
    await open(page, server.baseURL);
    const cards = thisWeek(page).locator('.briefing-holiday');
    await expect(cards.first()).toContainText('Christmas Day · No mail today');
    // New Year's Day observed on Fri 31 Dec: mail as normal.
    await expect(thisWeek(page)).toContainText("New Year's Day (observed) · Mail as normal");
    await expect(page.locator('.briefing-holiday-title').first()).toHaveAttribute('title', /Post offices are closed/);
    await axeCheck(page);
  });
});
