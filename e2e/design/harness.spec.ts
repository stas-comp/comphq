import { expect, test } from './fixtures';
import { diffStyles, expandProps, expectMatchesMockup, firstFamily, type Pair } from './mockup';

// Self-test for the mockup-comparison harness (PLAN P4-02): before every
// later task leans on it, prove it reports a match, reports a deliberate
// mismatch by naming the property, and needs no internet.

const STAMP: Pair = { mockup: '.stamp.today', screen: 'briefing', app: '#probe' };
const PROPS = [
  'font-family',
  'font-weight',
  'font-size',
  'letter-spacing',
  'text-transform',
  'color',
  'background-color',
  'border-top-width',
  'border-top-style',
  'border-top-color',
  'border-radius',
];

// Builds a stand-in "app" element whose inline style is exactly what the
// mockup computed, then lets a test change one thing.
async function probeFrom(
  mockup: import('./mockup').Mockup,
  page: import('@playwright/test').Page,
  change: Record<string, string> = {},
): Promise<void> {
  await mockup.show('briefing');
  const styles = { ...(await mockup.styles(STAMP.mockup, PROPS)), ...change };
  const css = expandProps(PROPS)
    .map((p) => `${p}: ${styles[p].replace(/"/g, '&quot;')}`)
    .join('; ');
  await page.setContent(`<span id="probe" style="display: inline-block; ${css}">TODAY</span>`);
}

test('harness: an element that agrees with the mockup passes', async ({ page, mockup }) => {
  await probeFrom(mockup, page);
  await expectMatchesMockup(mockup, page, STAMP, PROPS);
});

test('harness: a deliberate mismatch fails and names the property and both values', async ({ page, mockup }) => {
  await probeFrom(mockup, page, { 'background-color': 'rgb(194, 65, 12)' });
  let message = '';
  try {
    await expectMatchesMockup(mockup, page, STAMP, PROPS);
  } catch (err) {
    message = String((err as Error).message);
  }
  expect(message, 'the mismatch must be reported, not swallowed').toContain('background-color');
  expect(message).toContain('rgb(194, 65, 12)');
  expect(message).toContain('rgb(255, 107, 26)');
  // Only the property that differs is named.
  expect(message).not.toContain('font-weight: mockup');
});

test('harness: font-family is compared by its first family, not the whole stack', async ({ page, mockup }) => {
  await probeFrom(mockup, page, { 'font-family': '"Big Shoulders Display", Impact, sans-serif' });
  await expectMatchesMockup(mockup, page, STAMP, ['font-family']);

  await probeFrom(mockup, page, { 'font-family': 'Georgia, "Big Shoulders Display", serif' });
  await expect(expectMatchesMockup(mockup, page, STAMP, ['font-family'])).rejects.toThrow(/font-family/);
});

test('harness: an element that is missing, or matches nothing, is an error and never a pass', async ({ page, mockup }) => {
  await page.setContent('<p>nothing here</p>');
  await expect(expectMatchesMockup(mockup, page, STAMP, ['color'])).rejects.toThrow(/no element matches "#probe"/);
  await expect(
    expectMatchesMockup(mockup, page, { mockup: '.no-such-thing', screen: 'briefing', app: 'p' }, ['color']),
  ).rejects.toThrow(/no element matches "\.no-such-thing"/);
});

test('harness: it switches mockup screens, so an element on a hidden one can be read', async ({ mockup }) => {
  await mockup.show('board');
  const col = await mockup.styles('.col.todo', ['background-color', 'font-family']);
  expect(col['background-color']).toBe('rgb(255, 243, 234)'); // --accent-wash
  await mockup.show('calendar');
  const sat = await mockup.styles('.day.sat', ['background-color']);
  expect(sat['background-color']).toBe('rgb(255, 243, 234)');
});

test('harness: it needs no internet — every network request the mockup makes is refused', async ({ mockup }) => {
  // The mockup's Google Fonts link is real and expected; it must have been
  // stopped rather than fetched, and the page must still have rendered.
  expect(mockup.refusedRequests.length).toBeGreaterThan(0);
  expect(mockup.refusedRequests.every((u) => /^https?:\/\/fonts\.g/.test(u))).toBe(true);
  const stamp = await mockup.styles('h1', ['display']);
  expect(stamp.display).not.toBe('');
});

test('harness: pure helpers', () => {
  expect(firstFamily('"Big Shoulders Display", "Arial Narrow", Impact, sans-serif')).toBe('big shoulders display');
  expect(firstFamily("'IBM Plex Mono', monospace")).toBe('ibm plex mono');
  expect(firstFamily('Consolas, monospace')).toBe('consolas');
  expect(diffStyles({ color: 'red' }, { color: 'red' })).toEqual([]);
  expect(diffStyles({ color: 'red' }, { color: 'blue' })).toEqual([{ prop: 'color', mockup: 'red', app: 'blue' }]);
  expect(diffStyles({ color: 'red' }, {})).toEqual([{ prop: 'color', mockup: 'red', app: '' }]);
});
