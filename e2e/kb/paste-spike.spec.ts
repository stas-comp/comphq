import fs from 'node:fs';
import path from 'node:path';
import { expect, test } from '../helpers/fixtures';
import { dropFile, loadFixtureFile, pasteFile, pasteHTML } from '../helpers/paste';
import { ready } from '../helpers/ready';

// P1-10: proves the chosen paste-testing method (D-12) works for both a
// pasted image (reaches the FileHandler callback) and pasted HTML (reaches
// the editor), against the P1-09 harness.

test('pasted PNG reaches the FileHandler callback within 3s', async ({ page, server }) => {
  await page.goto(server.baseURL + '/__test/editor');
  await ready(page);

  const png = loadFixtureFile('images/tiny.png', 'image/png');
  await pasteFile(page, '#editor .ProseMirror', png);

  const log = page.locator('#paste-log');
  await expect(log).toHaveAttribute('data-pasted', 'true', { timeout: 3000 });
  await expect(log).toHaveText('tiny.png');
});

test('dropped PNG reaches the FileHandler callback within 3s', async ({ page, server }) => {
  await page.goto(server.baseURL + '/__test/editor');
  await ready(page);

  const png = loadFixtureFile('images/tiny.png', 'image/png');
  await dropFile(page, '#editor .ProseMirror', png);

  const log = page.locator('#paste-log');
  await expect(log).toHaveAttribute('data-dropped', 'true', { timeout: 3000 });
  await expect(log).toHaveText('tiny.png');
});

test('pasted web-page HTML reaches the editor', async ({ page, server }) => {
  await page.goto(server.baseURL + '/__test/editor');
  await ready(page);

  const html = fs.readFileSync(path.join(__dirname, '..', 'fixtures', 'paste', 'web-page.html'), 'utf8');
  await pasteHTML(page, '#editor .ProseMirror', html);

  await expect(page.locator('#editor')).toContainText('toner');
});

test('pasted Word HTML reaches the editor', async ({ page, server }) => {
  await page.goto(server.baseURL + '/__test/editor');
  await ready(page);

  const html = fs.readFileSync(path.join(__dirname, '..', 'fixtures', 'paste', 'word.html'), 'utf8');
  await pasteHTML(page, '#editor .ProseMirror', html);

  await expect(page.locator('#editor')).toContainText('Printer Supplies Policy');
});
