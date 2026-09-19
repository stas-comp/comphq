import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { expect, test } from '../helpers/fixtures';
import { loadFixtureFile, pasteFile } from '../helpers/paste';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';
import { openNewTask } from '../helpers/tasks';

type Page = import('@playwright/test').Page;

async function createCategory(page: Page, baseURL: string, name: string) {
  await page.goto(baseURL + '/kb/categories');
  await page.fill('#new-category-name', name);
  await page.click('.create-category-form button[type="submit"]');
  await ready(page);
}

async function publishArticle(
  page: Page,
  baseURL: string,
  category: string,
  title: string,
  withImage: boolean,
): Promise<void> {
  await page.goto(baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
  await page.selectOption('#article-category', { label: category });
  await page.fill('#article-title', title);
  const editor = page.locator('#article-editor .ProseMirror');
  await editor.click();
  await expect(editor).toBeFocused();
  await page.keyboard.type('Some plain body text.');
  if (withImage) {
    const png = loadFixtureFile('images/tiny.png', 'image/png');
    await pasteFile(page, '#article-editor .ProseMirror', png);
    await expect(page.locator('#article-editor img')).toHaveCount(1, { timeout: 3000 });
  }
  await page.click('#btn-publish');
  await ready(page);
}

// SPEC gate 1.34: Settings > Export everything downloads one file that
// opens offline, with a contents page listing every article by category
// (archived ones marked) and every article readable with its images.
test('exported zip opens offline with every article, its image, and archived ones marked', async ({
  page,
  server,
  browser,
}) => {
  const category = uniqueName('IT');
  await signInAsNewPerson(page, server.baseURL, '/kb/categories');
  await createCategory(page, server.baseURL, category);

  const withImageTitle = uniqueName('Article with a picture');
  await publishArticle(page, server.baseURL, category, withImageTitle, true);

  const toArchiveTitle = uniqueName('Article to archive');
  await publishArticle(page, server.baseURL, category, toArchiveTitle, false);
  await page.locator('.inline-form button[type="submit"]', { hasText: 'Archive' }).click();
  await ready(page);

  const res = await page.request.get(server.baseURL + '/settings/export');
  expect(res.ok()).toBeTruthy();
  expect(res.headers()['content-type']).toBe('application/zip');
  const zipBytes = await res.body();

  const workDir = fs.mkdtempSync(path.join(os.tmpdir(), 'comphq-export-'));
  const zipPath = path.join(workDir, 'export.zip');
  const extractDir = path.join(workDir, 'extracted');
  fs.writeFileSync(zipPath, zipBytes);

  const unzipBin = process.env.COMPHQ_UNZIP_BINARY;
  if (!unzipBin) throw new Error('COMPHQ_UNZIP_BINARY is not set; global setup should have built it');
  execFileSync(unzipBin, [zipPath, extractDir]);

  expect(fs.existsSync(path.join(extractDir, 'index.html'))).toBe(true);
  expect(fs.existsSync(path.join(extractDir, 'comphq.db'))).toBe(true);

  const offlineContext = await browser.newContext({ offline: true });
  const offlinePage = await offlineContext.newPage();
  const failedRequests: string[] = [];
  offlinePage.on('requestfailed', (req) => failedRequests.push(req.url()));

  await offlinePage.goto(pathToFileURL(path.join(extractDir, 'index.html')).toString());
  await expect(offlinePage.locator('a', { hasText: withImageTitle })).toHaveCount(1);
  await expect(offlinePage.locator('a', { hasText: toArchiveTitle })).toHaveCount(1);
  await expect(offlinePage.locator('body')).toContainText('archived', { ignoreCase: true });

  await offlinePage.click(`a:has-text("${withImageTitle}")`);
  await expect(offlinePage.locator('h1')).toHaveText(withImageTitle);
  const img = offlinePage.locator('img');
  await expect(img).toHaveCount(1);
  const naturalWidth = await img.evaluate((el: HTMLImageElement) => el.naturalWidth);
  expect(naturalWidth).toBeGreaterThan(0);

  await offlinePage.goto(pathToFileURL(path.join(extractDir, 'index.html')).toString());
  await offlinePage.click(`a:has-text("${toArchiveTitle}")`);
  await expect(offlinePage.locator('h1')).toHaveText(toArchiveTitle);

  expect(failedRequests, JSON.stringify(failedRequests)).toEqual([]);
  await offlineContext.close();
});

// SPEC gate 2.22: Export everything also holds tasks.csv and events.csv,
// which open in Excel — a UTF-8 byte-order mark, CRLF line endings, clear
// column names and readable dates.
test('exported zip includes tasks.csv and events.csv with clear headers and readable dates', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');

  const taskTitle = uniqueName('Export task');
  await openNewTask(page);
  await page.fill('#new-task-title', taskTitle);
  await page.selectOption('#new-task-stage', 'todo');
  await page.fill('#new-task-due-date', '2026-09-19');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  const eventTitle = uniqueName('Export event');
  await page.goto(server.baseURL + '/calendar/new');
  await page.fill('#event-title', eventTitle);
  await page.fill('#event-start-date', '2026-12-12');
  await page.selectOption('#event-recurrence', 'yearly');
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);

  const res = await page.request.get(server.baseURL + '/settings/export');
  expect(res.ok()).toBeTruthy();
  const workDir = fs.mkdtempSync(path.join(os.tmpdir(), 'comphq-export-csv-'));
  const zipPath = path.join(workDir, 'export.zip');
  const extractDir = path.join(workDir, 'extracted');
  fs.writeFileSync(zipPath, await res.body());
  const unzipBin = process.env.COMPHQ_UNZIP_BINARY;
  if (!unzipBin) throw new Error('COMPHQ_UNZIP_BINARY is not set; global setup should have built it');
  execFileSync(unzipBin, [zipPath, extractDir]);

  const read = (name: string) => fs.readFileSync(path.join(extractDir, name));
  const bom = Buffer.from([0xef, 0xbb, 0xbf]);

  const tasksCSV = read('tasks.csv');
  expect(tasksCSV.subarray(0, 3).equals(bom)).toBe(true);
  const tasksText = tasksCSV.subarray(3).toString('utf8');
  expect(tasksText.startsWith('Title,Column,Size,People,Due date,Created by,Created,Finished,Removed\r\n')).toBe(true);
  const taskRow = tasksText.split('\r\n').find((l) => l.includes(taskTitle));
  expect(taskRow, 'the new task is a row in tasks.csv').toBeTruthy();
  expect(taskRow).toContain(',To do,Medium,,Sat 19 Sep 2026,');

  const eventsCSV = read('events.csv');
  expect(eventsCSV.subarray(0, 3).equals(bom)).toBe(true);
  const eventsText = eventsCSV.subarray(3).toString('utf8');
  expect(eventsText.startsWith('Title,Date,End date,Start time,End time,Repeats,Until,Show ahead,Notes,Removed\r\n')).toBe(true);
  const eventRow = eventsText.split('\r\n').find((l) => l.includes(eventTitle));
  expect(eventRow, 'the new event is a row in events.csv').toBeTruthy();
  expect(eventRow).toContain(`${eventTitle},Sat 12 Dec 2026,,,,Yearly,,1 week,,`);
});
