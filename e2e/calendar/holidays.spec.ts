import { expect, test } from '../helpers/fixtures-today';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

type Page = import('@playwright/test').Page;

// SPEC gates 8.60-8.65 (B14.5, D-109): US federal holidays on the Calendar. Every
// test is @fresh: it needs its own server with its own COMPHQ_TEST_TODAY.

const NO_MAIL = 'Federal holiday. Post offices are closed and there is no mail delivery.';
const MAIL = 'Federal offices are closed today, but post offices are open and mail is delivered as normal.';

const chip = (page: Page, date: string) => page.locator(`.calendar-day[data-date="${date}"] .calendar-chip-holiday`);

async function addEvent(page: Page, baseURL: string, title: string, date: string): Promise<void> {
  await page.goto(baseURL + '/calendar/new');
  await ready(page);
  await page.locator('[name="title"]').fill(title);
  await page.locator('[name="start_date"]').fill(date);
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);
}

test.describe('gates 8.60-8.64: the month grid', () => {
  test.use({ today: '2026-11-21' });

  test('@fresh November 2026: Thanksgiving and Veterans Day, with wording, tooltip and a spoken sentence', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2026-11');
    await ready(page);
    const thanks = chip(page, '2026-11-26');
    await expect(thanks).toHaveCount(1);
    await expect(thanks.locator('.calendar-holiday-words')).toHaveText('Thanksgiving Day · No mail');
    await expect(thanks).toHaveAttribute('title', NO_MAIL);
    await expect(thanks).toHaveAttribute('tabindex', '0');
    await expect(thanks.locator('.visually-hidden')).toContainText(NO_MAIL); // a screen reader reads it
    await expect(chip(page, '2026-11-11').locator('.calendar-holiday-words')).toHaveText('Veterans Day · No mail');
    // Not a link and not a button: it cannot open the editor.
    expect(await thanks.evaluate((el) => el.tagName)).toBe('SPAN');
    await expect(thanks.locator('a, button')).toHaveCount(0);
    // It is a third kind of chip: not the solid event, not the outlined job.
    await expect(thanks).not.toHaveClass(/calendar-chip-event|calendar-chip-task/);
    // Keyboard focus reaches it.
    await thanks.focus();
    await expect(thanks).toBeFocused();
    await axeCheck(page);
    await expectNoSideScroll(page);
  });

  test('@fresh July 2027: Sunday 4 July shows the holiday and Monday 5 July the observed no-mail day', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2027-07');
    await ready(page);
    await expect(chip(page, '2027-07-04').locator('.calendar-holiday-words')).toHaveText('Independence Day · No mail');
    await expect(chip(page, '2027-07-04')).toHaveAttribute('title', NO_MAIL);
    await expect(chip(page, '2027-07-05').locator('.calendar-holiday-words')).toHaveText('Independence Day (observed) · No mail');
    // Saturday 19 June 2027 (Juneteenth) is in the June grid; the Friday before is mail as normal.
    await page.goto(server.baseURL + '/calendar?month=2027-06');
    await ready(page);
    await expect(chip(page, '2027-06-19').locator('.calendar-holiday-words')).toHaveText('Juneteenth · No mail');
    await expect(chip(page, '2027-06-18').locator('.calendar-holiday-words')).toHaveText('Juneteenth (observed) · Mail as normal');
    await expect(chip(page, '2027-06-18')).toHaveAttribute('title', MAIL);
    await axeCheck(page);
  });

  test("@fresh December 2027: Christmas on a Saturday, with the observed Friday, and New Year's Day observed on 31 December", async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2027-12');
    await ready(page);
    await expect(chip(page, '2027-12-24').locator('.calendar-holiday-words')).toHaveText('Christmas Day (observed) · Mail as normal');
    await expect(chip(page, '2027-12-24')).toHaveAttribute('title', MAIL);
    await expect(chip(page, '2027-12-25').locator('.calendar-holiday-words')).toHaveText('Christmas Day · No mail');
    await expect(chip(page, '2027-12-31').locator('.calendar-holiday-words')).toHaveText("New Year's Day (observed) · Mail as normal");
    // 1 January 2028 is in the grid's last row too (a Saturday).
    await expect(chip(page, '2028-01-01').locator('.calendar-holiday-words')).toHaveText("New Year's Day · No mail");
    await axeCheck(page);
    await expectNoSideScroll(page);
  });

  test("@fresh gate 8.61: a holiday is listed before that day's events; gate 8.64: a click on it does nothing, a click on blank space still starts an event", async ({
    page,
    server,
  }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2026-11');
    await addEvent(page, server.baseURL, 'Turkey lunch', '26/11/2026');
    await page.goto(server.baseURL + '/calendar?month=2026-11');
    await ready(page);
    const cell = page.locator('.calendar-day[data-date="2026-11-26"]');
    const kinds = await cell.locator('.calendar-chip').evaluateAll((els) => els.map((e) => e.getAttribute('data-kind')));
    expect(kinds).toEqual(['holiday', 'event']);

    const before = page.url();
    await chip(page, '2026-11-26').click();
    await page.waitForTimeout(400);
    expect(page.url()).toBe(before); // nothing opened: no event editor, no new-event page
    await expect(page.locator('#event-start-date')).toHaveCount(0);
    await chip(page, '2026-11-26').press('Enter');
    expect(page.url()).toBe(before);

    // A blank part of the same day still starts a new event (gate 6.21).
    await cell.scrollIntoViewIfNeeded();
    const box = (await cell.boundingBox())!;
    await page.mouse.click(box.x + box.width - 8, box.y + box.height - 6);
    await ready(page);
    await expect(page.locator('#event-start-date')).toHaveValue('26/11/2026');
  });

  test("@fresh gate 8.64, 8.67: holidays can't be edited or removed, and aren't searchable", async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2026-11');
    await ready(page);
    await expect(page.locator('.calendar-chip-holiday button, .calendar-chip-holiday a, .calendar-chip-holiday input')).toHaveCount(0);
    await page.goto(server.baseURL + '/calendar/removed');
    await expect(page.getByText('Thanksgiving')).toHaveCount(0);
    const found = await page.request.get(server.baseURL + '/kb/search.json?q=Thanksgiving');
    expect((await found.json()).results).toHaveLength(0);
    const jobs = await page.request.get(server.baseURL + '/tasks/search.json?q=Thanksgiving');
    expect((await jobs.json()).results).toHaveLength(0);
  });
});

test.describe('gate 8.60: the 12-week list', () => {
  test.use({ today: '2026-11-21' });

  test("@fresh the list shows the holidays in the next 12 weeks, before that day's events", async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2026-11');
    await addEvent(page, server.baseURL, 'Christmas party', '25/12/2026');
    await page.goto(server.baseURL + '/calendar/list');
    await ready(page);
    const rows = await page.locator('.calendar-list-row').allTextContents();
    const text = rows.map((r) => r.replace(/\s+/g, ' ').trim());
    const thanks = text.findIndex((r) => r.includes('Thanksgiving Day · No mail'));
    const xmas = text.findIndex((r) => r.includes('Christmas Day · No mail'));
    const party = text.findIndex((r) => r.includes('Christmas party'));
    expect(thanks).toBeGreaterThanOrEqual(0);
    expect(xmas).toBeGreaterThanOrEqual(0);
    expect(xmas).toBeLessThan(party); // the holiday first
    expect(text.some((r) => r.includes("New Year's Day · No mail"))).toBe(true); // 1 Jan 2027 (a Friday)
    // The same chip, with the same tooltip.
    await expect(page.locator('.calendar-list-holiday .calendar-chip-holiday').first()).toHaveAttribute('title', NO_MAIL);
    await axeCheck(page);
  });
});
