import { expect, test } from './fixtures';
import { expectMatchesMockup } from './mockup';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

// SPEC gate 7.77 (B13.10, B9.9 layer 1): the Knowledge Base home and a
// category's page were never drawn in the mockup, so each part they use is
// compared with the same part wherever the mockup does draw it. They add no
// new colour or font of their own.

type Page = import('@playwright/test').Page;

const TEXT = ['font-family', 'font-weight', 'font-size', 'letter-spacing', 'text-transform', 'color'];
const BOX = ['background-color', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius'];

async function setup(page: Page, baseURL: string) {
  await signInAsNewPerson(page, baseURL, '/kb');
  const name = uniqueName('Design');
  const res = await page.request.post(baseURL + '/kb/categories', { form: { name }, headers: { origin: baseURL } });
  expect(res.ok()).toBe(true);
  await page.goto(baseURL + '/kb/new');
  const id = (await page.locator('select[name="category_id"] option', { hasText: name }).getAttribute('value'))!;
  for (const title of ['One', 'Two']) {
    const r = await page.request.post(baseURL + '/kb/articles', {
      form: { title: `${uniqueName(title)}`, category_id: id, body_html: '<p>Some opening words for the row.</p>' },
      headers: { origin: baseURL },
    });
    expect(r.ok()).toBe(true);
  }
  return { id, name };
}

test('gate 7.77: the home uses the mockup’s heading buttons, card surface and panel head', async ({ page, server, mockup }) => {
  await setup(page, server.baseURL);
  await page.goto(server.baseURL + '/kb');
  await ready(page);

  await expectMatchesMockup(mockup, page, { mockup: '.article .by .btn.primary', screen: 'kb', app: '.kb-head-actions .btn.primary' }, [...TEXT, ...BOX, 'padding']);
  await expectMatchesMockup(mockup, page, { mockup: '.article .by .btn:not(.primary)', screen: 'kb', app: '.kb-head-actions .btn:not(.primary)' }, [...TEXT.filter((p) => p !== 'color'), ...BOX, 'padding']); // (the mockup's inherits its byline's grey, which is not the button's own)
  await expectMatchesMockup(mockup, page, { mockup: '.panel-head', screen: 'briefing', app: '.kb-recent .panel-head' }, ['display', 'align-items', 'column-gap', 'border-bottom-width', 'border-bottom-style', 'border-bottom-color', 'padding-bottom']);
  await expectMatchesMockup(mockup, page, { mockup: '.panel-head h2', screen: 'briefing', app: '.kb-recent .panel-head h2' }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.panel-head .count', screen: 'briefing', app: '.kb-recent .panel-head .count' }, [...TEXT]);
  await expectMatchesMockup(mockup, page, { mockup: '.card', screen: 'board', app: '.kb-tile' }, BOX);
  await expectMatchesMockup(mockup, page, { mockup: '.article .by .date', screen: 'kb', app: '.kb-recent-by .date' }, [...TEXT, 'font-variant-numeric']);

  // The tile's name is the display face; the articles in it are the readable one.
  await expect(page.locator('.kb-tile h2').first()).toHaveCSS('font-family', /Big Shoulders Display/);
  await expect(page.locator('.kb-tile-articles a').first()).toHaveCSS('font-family', /Atkinson Hyperlegible Next/);
  await expect(page.locator('body')).toHaveCSS('background-color', 'rgb(245, 243, 238)');
});

test('gate 7.77: a category page uses the mockup’s trail, button and card surface', async ({ page, server, mockup }) => {
  const cat = await setup(page, server.baseURL);
  await page.goto(`${server.baseURL}/kb/categories/${cat.id}`);
  await ready(page);

  await expectMatchesMockup(mockup, page, { mockup: '.crumbs', screen: 'kb', app: '.kb-crumbs' }, ['font-size', 'color', 'font-family']);
  await expectMatchesMockup(mockup, page, { mockup: '.crumbs a', screen: 'kb', app: '.kb-crumbs a' }, ['color']);
  await expectMatchesMockup(mockup, page, { mockup: '.article .by .btn.primary', screen: 'kb', app: '.kb-head-actions .btn.primary' }, [...TEXT, ...BOX, 'padding']);
  await expectMatchesMockup(mockup, page, { mockup: '.card', screen: 'board', app: '.kb-article-list li' }, BOX);
  await expectMatchesMockup(mockup, page, { mockup: '.article .by .date', screen: 'kb', app: '.kb-article-by .date' }, [...TEXT, 'font-variant-numeric']);
  await expect(page.locator('.kb-head h1')).toHaveCSS('font-family', /Big Shoulders Display/);
  await expect(page.locator('.kb-article-title').first()).toHaveCSS('font-family', /Atkinson Hyperlegible Next/);
});
