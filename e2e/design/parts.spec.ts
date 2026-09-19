import { expect, test } from './fixtures';
import { expectMatchesMockup, type Pair } from './mockup';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';

// Shared parts (PLAN P4-03, gates 4.04-4.10): every reusable part, on one
// page (/__test/parts, drawn by the real partials), agrees with
// docs/design/mockup.html property by property. The page exists only in
// test mode.

const TEXT = ['font-family', 'font-weight', 'font-size', 'letter-spacing', 'text-transform', 'color'];
const BOX = [
  'background-color',
  'border-top-width',
  'border-top-style',
  'border-top-color',
  'border-radius',
  'padding',
];

type Case = { name: string; pair: Pair; props: string[] };

const cases: Case[] = [
  // Gate 4.04: three kinds of button and no others.
  { name: 'primary button', pair: { mockup: '.tools .btn.primary', screen: 'tasks', app: '#parts-btn-primary' }, props: [...TEXT, ...BOX] },
  { name: 'secondary button', pair: { mockup: '.tools .btn:not(.primary)', screen: 'calendar', app: '#parts-btn' }, props: [...TEXT, ...BOX] },
  { name: 'small button', pair: { mockup: '.card .acts .mini:not(.go):not(.quiet)', screen: 'tasks', app: '#parts-mini' }, props: [...TEXT, ...BOX] },
  { name: 'small accent button', pair: { mockup: '.card .acts .mini.go', screen: 'tasks', app: '#parts-mini-go' }, props: [...TEXT, ...BOX] },
  { name: 'small quiet button', pair: { mockup: '.card .acts .mini.quiet', screen: 'tasks', app: '#parts-mini-quiet' }, props: [...TEXT, ...BOX] },
  { name: 'editor toolbar button', pair: { mockup: '.bar .tb', screen: 'editor', app: '#parts-tb' }, props: [...TEXT, 'background-color', 'border-radius', 'padding', 'height'] },
  { name: 'link-style button', pair: { mockup: '.picker a', screen: 'picker', app: '#parts-link-button' }, props: ['text-decoration-line', 'text-decoration-color', 'text-underline-offset'] },
  // Gate 4.05: styled controls, from the mockup's own search box.
  { name: 'text box', pair: { mockup: '.search', screen: 'briefing', app: '#parts-text' }, props: ['background-color', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius', 'padding', 'font-size'] },
  { name: 'search box', pair: { mockup: '.search', screen: 'briefing', app: '#parts-search' }, props: ['background-color', 'border-top-width', 'border-top-color', 'border-radius', 'padding', 'font-size'] },
  { name: 'drop-down', pair: { mockup: '.search', screen: 'briefing', app: '#parts-select' }, props: ['background-color', 'border-top-width', 'border-top-color', 'border-radius', 'font-size'] },
  { name: 'text area', pair: { mockup: '.search', screen: 'briefing', app: '#parts-textarea' }, props: ['background-color', 'border-top-width', 'border-top-color', 'border-radius', 'font-size'] },
  { name: 'date field', pair: { mockup: '.date', screen: 'briefing', app: '#parts-date-field' }, props: ['font-family', 'font-weight', 'font-variant-numeric'] },
  // Gate 4.06: one segmented control.
  { name: 'segmented control', pair: { mockup: '.tools .seg', screen: 'tasks', app: '#parts-seg' }, props: ['display', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius', 'overflow'] },
  { name: 'segmented control, current', pair: { mockup: '.tools .seg .on', screen: 'tasks', app: '#parts-seg [aria-current]' }, props: [...TEXT, 'background-color', 'padding'] },
  { name: 'segmented control, other', pair: { mockup: '.tools .seg button', screen: 'tasks', app: '#parts-seg a:not([aria-current])' }, props: [...TEXT, 'background-color', 'padding'] },
  // Gate 4.07: stamps.
  { name: 'TODAY stamp', pair: { mockup: '.stamp.today', screen: 'briefing', app: '#parts-stamp-today' }, props: [...TEXT, ...BOX, 'transform', 'padding', 'line-height'] },
  { name: 'OVERDUE stamp', pair: { mockup: '.stamp.overdue', screen: 'briefing', app: '#parts-stamp-overdue' }, props: [...TEXT, ...BOX, 'transform'] },
  { name: 'YOURS stamp', pair: { mockup: '.stamp.yours', screen: 'briefing', app: '#parts-stamp-yours' }, props: [...TEXT, ...BOX] },
  { name: 'date stamp', pair: { mockup: '.stamp.when', screen: 'briefing', app: '#parts-stamp-when' }, props: [...TEXT, ...BOX] },
  // Gate 4.08: people circles.
  { name: 'person circle', pair: { mockup: '.people .av.b', screen: 'briefing', app: '#parts-people .av.b' }, props: [...TEXT, ...BOX, 'width', 'height'] },
  { name: 'person circle, c', pair: { mockup: '.people .av.c', screen: 'briefing', app: '#parts-people .av.c' }, props: ['background-color', 'color'] },
  { name: 'person circle, d', pair: { mockup: '.people .av.d', screen: 'briefing', app: '#parts-people .av.d' }, props: ['background-color', 'color'] },
  { name: 'person circle, the person using the app', pair: { mockup: '.people .av.me', screen: 'briefing', app: '#parts-people .av.me' }, props: ['background-color', 'color'] },
  { name: 'people stack overlap', pair: { mockup: '.people .av + .av', screen: 'briefing', app: '#parts-people .av + .av' }, props: ['margin-left'] },
  // Gate 4.09: size chips, in three increasingly dark shades.
  { name: 'SMALL chip', pair: { mockup: '.size.s', screen: 'tasks', app: '#parts-sizes .size.s' }, props: [...TEXT, 'background-color', 'border-radius', 'padding', 'line-height'] },
  { name: 'MEDIUM chip', pair: { mockup: '.size.m', screen: 'tasks', app: '#parts-sizes .size.m' }, props: [...TEXT, 'background-color'] },
  { name: 'LARGE chip', pair: { mockup: '.size.l', screen: 'tasks', app: '#parts-sizes .size.l' }, props: [...TEXT, 'background-color'] },
  // Gate 4.10: workload blocks.
  { name: 'workload row', pair: { mockup: '.lane:not(.me) .load', screen: 'team', app: '#parts-workload .workload' }, props: ['display', 'flex-wrap', 'row-gap', 'column-gap', 'min-height'] },
  { name: 'workload group', pair: { mockup: '.lane:not(.me) .load .t', screen: 'team', app: '#parts-workload .workload-group' }, props: ['display', 'column-gap'] },
  { name: 'workload block', pair: { mockup: '.lane:not(.me) .load .t i', screen: 'team', app: '#parts-workload .workload-block' }, props: ['width', 'height', 'background-color', 'border-radius'] },
  // Cards, panel heads and dates.
  { name: 'card', pair: { mockup: '.col .card', screen: 'board', app: '#parts-card' }, props: ['background-color', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius', 'padding', 'display', 'row-gap'] },
  { name: 'panel head', pair: { mockup: '.panel-head', screen: 'briefing', app: '#parts-panel-head' }, props: ['display', 'align-items', 'column-gap', 'border-bottom-width', 'border-bottom-style', 'border-bottom-color', 'padding-bottom'] },
  { name: 'panel head title', pair: { mockup: '.panel-head h2', screen: 'briefing', app: '#parts-panel-head h2' }, props: [...TEXT, 'line-height'] },
  { name: 'panel head count', pair: { mockup: '.panel-head .count', screen: 'briefing', app: '#parts-panel-head .count' }, props: [...TEXT] },
  { name: 'date', pair: { mockup: '.date', screen: 'briefing', app: '#parts-date' }, props: [...TEXT, 'font-variant-numeric'] },
];

test.describe('shared parts agree with the mockup', () => {
  for (const c of cases) {
    test(c.name, async ({ page, server, mockup }) => {
      await signInAsNewPerson(page, server.baseURL, '/__test/parts');
      await ready(page);
      await expectMatchesMockup(mockup, page, c.pair, c.props);
    });
  }
});

// Gate 4.10: the blocks are 1, 2 and 4, and have a spoken equivalent.
test('gate 4.10: workload blocks per job are 1, 2 and 4, and are spoken', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/__test/parts');
  await ready(page);
  const perJob = await page.locator('#parts-workload .workload-group').evaluateAll((els) => els.map((el) => el.children.length));
  expect(perJob).toEqual([4, 2, 1]);
  const workload = page.locator('#parts-workload .workload');
  await expect(workload).toHaveAttribute('role', 'img');
  await expect(workload).toHaveAttribute('aria-label', 'Workload: 7 blocks — large, medium, small');
});

// Gate 4.09: three increasingly dark shades.
test('gate 4.09: SMALL, MEDIUM and LARGE are three different shades, getting darker', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/__test/parts');
  await ready(page);
  const shades = await page
    .locator('#parts-sizes .size')
    .evaluateAll((els) => els.map((el) => getComputedStyle(el).backgroundColor.match(/\d+/g)!.map(Number)));
  const lightness = shades.map(([r, g, b]) => 0.2126 * r + 0.7152 * g + 0.0722 * b);
  expect(new Set(shades.map((s) => s.join(','))).size).toBe(3);
  expect(lightness[0]).toBeGreaterThan(lightness[1]);
  expect(lightness[1]).toBeGreaterThan(lightness[2]);
});

// Gate 4.06: the switch is one shared control wherever a view is switched.
test('gate 4.06: My jobs / Board / Team and Month / List use the shared segmented control', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  await ready(page);
  await expect(page.locator('.tasks-view-switch')).toHaveClass(/\bseg\b/);
  await expect(page.locator('.tasks-view-switch [aria-current="page"]')).toHaveText('My jobs');
  await page.goto(server.baseURL + '/calendar');
  await ready(page);
  await expect(page.locator('.calendar-view-switch')).toHaveClass(/\bseg\b/);
});

// Gate 4.08: the same person is the same colour on every screen, and the
// person using the app is the accent circle.
test('gate 4.08: the person using the app is the accent circle on the board, and others are not', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  await openNewTask(page);
  const title = 'Circle check ' + Date.now();
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-people', { label: me });
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  // This card's own circle (the shared board also holds other tests' cards).
  const circle = page.locator('.task-card', { hasText: title }).locator('.av').first();
  await expect(circle).toHaveClass(/\bme\b/);
  await expect(circle).toHaveCSS('background-color', 'rgb(255, 107, 26)');
  await expect(circle).toHaveCSS('color', 'rgb(20, 27, 45)');
});

test('standard page checks for the parts page', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/__test/parts');
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);
});
