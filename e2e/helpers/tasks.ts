import type { Page } from '@playwright/test';
import { ready } from './ready';

// The board no longer carries an add-a-task form (SPEC gate 4.15): + Add
// task opens the task window, and the same form is always available as a
// page (gate 4.28). Tests that create a task open that page, fill the
// fields (ids new-task-title, -notes, -size, -stage, -due-date, -people),
// and press the form's submit button (.add-task-form button[type=submit]),
// which lands back on the Board.
export async function openNewTask(page: Page): Promise<void> {
  await page.goto(new URL('/tasks/new', page.url()).href);
  await ready(page);
}

// Clicking a card's title opens the task window (SPEC gate 4.23); the same
// link, followed as a link, is the task's own page (gate 4.27). Tests that
// mean the page go there by the link's address.
export async function openTaskPage(page: Page, title: string): Promise<void> {
  const href = await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').first().getAttribute('href');
  if (!href) throw new Error(`no card titled ${title}`);
  await page.goto(new URL(href, page.url()).href);
  await ready(page);
}
