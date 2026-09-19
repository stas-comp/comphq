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
