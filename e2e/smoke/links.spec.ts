import { inflateRawSync } from 'node:zlib';
import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// Linked jobs across an upgrade and a rollback (SPEC gate 7.49, B13.6, D-96):
// three jobs and their links are made under the new build; the container is
// rolled back to the previous release, which has never heard of links and must
// carry on as if they weren't there; upgraded again, every link is still there.
// The links are read from Export
// everything's links.csv (gate 7.48), which needs no screen the previous
// release lacks.
//
// Tagged @links-seed / @links-rolledback / @links-verify-upgraded (none a
// substring of another) for container-test.sh's phases. Titles are fixed
// because a later phase has to find them; the person is not.
const A = 'Smoke Links Job A';
const B = 'Smoke Links Job B';
const C = 'Smoke Links Job C';

async function makeJob(page: import('@playwright/test').Page, baseURL: string, title: string): Promise<string> {
  const res = await page.request.post(baseURL + '/tasks', { form: { title, stage: 'todo' }, headers: { origin: baseURL }, maxRedirects: 0 });
  expect(res.status()).toBeLessThan(400);
  await page.goto(baseURL + '/tasks/board?q=' + encodeURIComponent(title));
  await ready(page);
  const links = page.locator('.task-card', { hasText: title }).locator('.task-card-title a');
  expect(await links.count(), `${title} should be on the Board exactly once`).toBe(1);
  return /\/tasks\/(\d+)/.exec((await links.first().getAttribute('href'))!)![1];
}

// A zip entry's bytes, by name: enough of the format to read Export
// everything's zip (local headers, deflate or stored).
function zipEntry(zip: Buffer, name: string): Buffer | null {
  let end = zip.length - 22;
  while (end >= 0 && zip.readUInt32LE(end) !== 0x06054b50) end--;
  if (end < 0) throw new Error('not a zip');
  const count = zip.readUInt16LE(end + 10);
  let p = zip.readUInt32LE(end + 16);
  for (let i = 0; i < count; i++) {
    const method = zip.readUInt16LE(p + 10);
    const size = zip.readUInt32LE(p + 20);
    const nameLen = zip.readUInt16LE(p + 28);
    const extraLen = zip.readUInt16LE(p + 30);
    const commentLen = zip.readUInt16LE(p + 32);
    const local = zip.readUInt32LE(p + 42);
    const entryName = zip.toString('utf8', p + 46, p + 46 + nameLen);
    if (entryName === name) {
      const dataStart = local + 30 + zip.readUInt16LE(local + 26) + zip.readUInt16LE(local + 28);
      const raw = zip.subarray(dataStart, dataStart + size);
      return method === 0 ? raw : inflateRawSync(raw);
    }
    p += 46 + nameLen + extraLen + commentLen;
  }
  return null;
}

async function linksCSV(page: import('@playwright/test').Page, baseURL: string): Promise<string[][]> {
  const res = await page.request.get(baseURL + '/settings/export');
  expect(res.ok()).toBe(true);
  const entry = zipEntry(Buffer.from(await res.body()), 'links.csv');
  expect(entry, 'links.csv is in the export').not.toBeNull();
  return entry!
    .toString('utf8')
    .replace(/^﻿/, '')
    .split('\r\n')
    .filter(Boolean)
    .map((line) => line.split(','));
}

test('@links-seed three jobs linked three ways', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const [a, b, c] = [await makeJob(page, server.baseURL, A), await makeJob(page, server.baseURL, B), await makeJob(page, server.baseURL, C)];
  const post = async (path: string, form: Record<string, string>) => {
    const res = await page.request.post(server.baseURL + path, { form, headers: { origin: server.baseURL }, maxRedirects: 0 });
    expect(res.status(), path).toBeLessThan(400);
  };
  await post(`/tasks/${a}/links`, { other_id: b, kind: 'then' }); // A first, then B
  await post(`/tasks/${b}/links`, { other_id: c, kind: 'related' });
  await post(`/tasks/${a}/links`, { other_id: c, kind: 'first' }); // C first, then A
  const rows = await linksCSV(page, server.baseURL);
  const row = (x: string, y: string) => rows.find((r) => r[0] === x && r[1] === y);
  expect(row(A, B)?.[2]).toBe('do first');
  expect(row(C, A)?.[2]).toBe('do first');
});

// Run against the PREVIOUS release's app, which has no links.
test('@links-rolledback the previous release carries on as if there were no links', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  for (const title of [A, B, C]) {
    await page.goto(server.baseURL + '/tasks/board?q=' + encodeURIComponent(title));
    await ready(page);
    const href = await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').first().getAttribute('href');
    await page.goto(server.baseURL + href!);
    await ready(page);
    await expect(page.locator('h1')).toHaveText(title);
    await expect(page.getByText('WAITING')).toHaveCount(0);
  }
});

test('@links-verify-upgraded every link survived the rollback', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const rows = await linksCSV(page, server.baseURL);
  const row = (x: string, y: string) => rows.find((r) => r[0] === x && r[1] === y);
  expect(row(A, B)?.[2]).toBe('do first');
  expect(row(C, A)?.[2]).toBe('do first');
  expect([row(B, C), row(C, B)].some((r) => r?.[2] === 'related')).toBe(true);
});
