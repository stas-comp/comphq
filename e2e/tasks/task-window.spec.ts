import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openTaskPage } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// The task window: the dialog, and creating a task (PLAN P4-08, gates
// 4.22, 4.25, 4.26, 4.28).

const addLink = (page: Page) => page.getByRole('link', { name: '+ Add task' });
const win = (page: Page) => page.locator('#task-window');

async function openWindow(page: Page): Promise<void> {
  await addLink(page).click();
  await expect(win(page)).toBeVisible();
}

test.beforeEach(async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
});

test('gate 4.22: + Add task opens the window, empty and ready, with every field and a primary Add task', async ({ page }) => {
  await openWindow(page);
  await expect(page).toHaveURL(/\/tasks\/board$/); // the window is over the Board, not a new page
  await expect(win(page).getByRole('heading', { name: 'Add task' })).toBeVisible();
  for (const id of ['new-task-title', 'new-task-notes', 'new-task-size', 'new-task-stage', 'new-task-due-date', 'new-task-people']) {
    await expect(win(page).locator(`#${id}`), id).toBeVisible();
  }
  await expect(win(page).locator('#new-task-title')).toHaveValue('');
  await expect(win(page).locator('#new-task-title')).toBeFocused(); // the keyboard moves in (gate 4.26)
  const add = win(page).getByRole('button', { name: 'Add task' });
  await expect(add).toHaveClass(/\bprimary\b/);
  await expect(win(page).getByRole('button', { name: 'Cancel' })).toBeVisible();
});

test('gate 4.22: a job made in the window keeps its description and size, and appears on the board without a reload', async ({ page, server }) => {
  const title = uniqueName('Window job');
  await page.evaluate(() => ((window as unknown as { __kept: number }).__kept = 1));
  await openWindow(page);
  await win(page).locator('#new-task-title').fill(title);
  await win(page).locator('#new-task-notes').fill('Three sets per room, stapled.');
  await win(page).locator('#new-task-size').selectOption('L');
  await win(page).locator('#new-task-stage').selectOption('todo');
  await win(page).getByRole('button', { name: 'Add task' }).click();

  await expect(win(page)).toBeHidden();
  const card = page.locator('.task-column[data-stage="todo"] .task-card', { hasText: title });
  await expect(card).toHaveCount(1);
  await expect(card.locator('.size')).toHaveText('Large');
  expect(await page.evaluate(() => (window as unknown as { __kept?: number }).__kept), 'no reload').toBe(1);
  await expect(page).toHaveURL(/\/tasks\/board$/);

  // Stored: the task's own page shows both.
  await openTaskPage(page, title);
  await expect(page.locator('#details-notes')).toHaveValue('Three sets per room, stapled.');
  await expect(page.locator('#details-size')).toHaveValue('L');
  await page.goto(server.baseURL + '/tasks/board');
});

test('gate 4.25: the window closes with Escape, the close icon, Cancel, and a click on the page behind it', async ({ page }) => {
  for (const how of ['escape', 'icon', 'cancel', 'backdrop'] as const) {
    await openWindow(page);
    if (how === 'escape') await page.keyboard.press('Escape');
    if (how === 'icon') await win(page).getByRole('button', { name: 'Close' }).click();
    if (how === 'cancel') await win(page).getByRole('button', { name: 'Cancel' }).click();
    if (how === 'backdrop') await page.mouse.click(8, 8);
    await expect(win(page), how).toBeHidden();
    // Back where you were, with the keyboard on what opened it (gate 4.26).
    await expect(addLink(page), how).toBeFocused();
    await expect(page).toHaveURL(/\/tasks\/board$/);
  }
});

test('gate 4.25: closing with something typed asks first; staying keeps it, leaving closes', async ({ page }) => {
  // Nothing typed: no question.
  let asked = 0;
  page.on('dialog', async (d) => {
    asked++;
    await d.dismiss();
  });
  await openWindow(page);
  await page.keyboard.press('Escape');
  await expect(win(page)).toBeHidden();
  expect(asked, 'no question when nothing was typed').toBe(0);

  // Something typed, and I choose to stay.
  await openWindow(page);
  await win(page).locator('#new-task-title').fill('Half written');
  await page.keyboard.press('Escape');
  await expect.poll(() => asked).toBe(1);
  await expect(win(page)).toBeVisible();
  await expect(win(page).locator('#new-task-title')).toHaveValue('Half written');
  await win(page).getByRole('button', { name: 'Close' }).click();
  await expect.poll(() => asked).toBe(2);
  await expect(win(page)).toBeVisible();

  // ...and this time I choose to leave.
  page.removeAllListeners('dialog');
  page.on('dialog', (d) => d.accept());
  await page.mouse.click(8, 8);
  await expect(win(page)).toBeHidden();
  await expect(addLink(page)).toBeFocused();
  // The next time it opens, the old text is gone.
  await openWindow(page);
  await expect(win(page).locator('#new-task-title')).toHaveValue('');
});

test('gate 4.26: Tab moves around inside the window only, and never reaches the board behind', async ({ page }) => {
  await openWindow(page);
  // Where the keyboard is: in the window, or off the page altogether (the
  // browser's own bar, which is how a native dialog lets Tab wrap round).
  // Never on anything of the board, sidebar or top bar.
  const where = () =>
    page.evaluate(() => {
      const el = document.activeElement;
      if (!el || el === document.body) return 'off the page';
      return el.closest('#task-window') ? 'inside' : 'behind: ' + (el.id || el.tagName + '.' + el.className);
    });
  const seen = new Set<string>();
  for (let i = 0; i < 30; i++) {
    await page.keyboard.press('Tab');
    const w = await where();
    seen.add(w);
    expect(w, `Tab #${i + 1}`).not.toMatch(/^behind/);
  }
  for (let i = 0; i < 30; i++) {
    await page.keyboard.press('Shift+Tab');
    const w = await where();
    seen.add(w);
    expect(w, `Shift+Tab #${i + 1}`).not.toMatch(/^behind/);
  }
  expect(seen.has('inside')).toBe(true);
});

test('gate 4.22: a refused save keeps the window open with a plain message and everything typed', async ({ page }) => {
  await openWindow(page);
  await win(page).locator('#new-task-title').fill('Needs a real date');
  await win(page).locator('#new-task-notes').fill('Kept as typed');
  await win(page).locator('#new-task-due-date').fill('31/02/2026');
  await win(page).getByRole('button', { name: 'Add task' }).click();
  await expect(win(page).getByRole('alert')).toContainText('day/month/year');
  await expect(win(page)).toBeVisible();
  await expect(win(page).locator('#new-task-title')).toHaveValue('Needs a real date');
  await expect(win(page).locator('#new-task-notes')).toHaveValue('Kept as typed');
  await expect(win(page).locator('#new-task-due-date')).toHaveValue('31/02/2026');
  // Fix it, and it saves.
  await win(page).locator('#new-task-due-date').fill('30/03/2026');
  await win(page).getByRole('button', { name: 'Add task' }).click();
  await expect(win(page)).toBeHidden();
  await expect(page.locator('.task-card', { hasText: 'Needs a real date' })).toHaveCount(1);
});

test('gate 4.22: the window passes the accessibility check and fits a small laptop window', async ({ page }) => {
  await page.setViewportSize({ width: 1024, height: 700 });
  await openWindow(page);
  await axeCheck(page);
  const box = (await win(page).boundingBox())!;
  expect(box.x).toBeGreaterThanOrEqual(0);
  expect(box.y).toBeGreaterThanOrEqual(0);
  expect(box.x + box.width).toBeLessThanOrEqual(1024);
  expect(box.y + box.height).toBeLessThanOrEqual(700);
  // Its own body scrolls when the form is taller than the window.
  await page.setViewportSize({ width: 1024, height: 420 });
  const fits = await win(page).evaluate((el) => el.getBoundingClientRect().bottom <= window.innerHeight);
  expect(fits).toBe(true);
});

// Gate 4.28: if the window can't open, nothing is lost.
test.describe('with JavaScript switched off', () => {
  test.use({ javaScriptEnabled: false });

  test('gate 4.28: + Add task is a link to a page that adds a task with the same fields', async ({ page, server }) => {
    await page.goto(server.baseURL + '/tasks/board');
    const add = page.getByRole('link', { name: '+ Add task' });
    await expect(add).toHaveAttribute('href', '/tasks/new');
    await add.click();
    await expect(page).toHaveURL(/\/tasks\/new$/);
    for (const id of ['new-task-title', 'new-task-notes', 'new-task-size', 'new-task-stage', 'new-task-due-date', 'new-task-people']) {
      await expect(page.locator(`#${id}`), id).toBeVisible();
    }
    const title = uniqueName('No script window');
    await page.fill('#new-task-title', title);
    await page.fill('#new-task-notes', 'A description');
    await page.selectOption('#new-task-size', 'S');
    await page.click('.add-task-form button[type="submit"]');
    await expect(page).toHaveURL(/\/tasks\/board$/);
    await expect(page.locator('.task-card', { hasText: title }).locator('.size')).toHaveText('Small');
  });
});

test('gate 4.28: if the window cannot load its form, + Add task still takes you to the page', async ({ page, server }) => {
  await page.route('**/tasks/new?fragment=1', (route) => route.abort());
  await addLink(page).click();
  await expect(page).toHaveURL(/\/tasks\/new$/);
  await expect(page.locator('#new-task-title')).toBeVisible();
  expect(server.baseURL).toBeTruthy();
});
