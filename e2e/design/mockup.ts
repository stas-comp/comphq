import type { BrowserContext, Page } from '@playwright/test';
import { expect } from '@playwright/test';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

// The mockup-comparison harness (SPEC B9.9 layer 1, PLAN P4-02).
//
// docs/design/mockup.html is the contract for how the app looks. This
// opens it in the same browser as the running app, reads the COMPUTED
// style of a named element in each, and asserts they agree on the
// properties given. It never compares screenshots: the two hold different
// content, so such a test would fail forever without teaching anyone
// anything.
//
// It works with no internet. getComputedStyle reports the *declared* font
// stack whether or not the font file loaded, so the mockup's Google Fonts
// link never has to resolve: every request the mockup makes to the
// network is refused, and font-family is compared by its FIRST family
// (never a resolved file).

export const MOCKUP_FILE = path.resolve(__dirname, '..', '..', 'docs', 'design', 'mockup.html');

export type StyleMap = Record<string, string>;
export type Diff = { prop: string; mockup: string; app: string };

/** One element in the mockup and its counterpart in the app. */
export type Pair = {
  /** CSS selector in the mockup (first match wins). */
  mockup: string;
  /** The mockup screen that must be showing for it to be visible, e.g. 'board'. */
  screen?: string;
  /** CSS selector in the app (first match wins). */
  app: string;
  /** A pseudo-element to read on both sides, e.g. '::before'. */
  pseudo?: string;
};

// Shorthands are read as their longhands: Chrome serialises a computed
// shorthand only when every side agrees, and a comparison that silently
// reads "" from both sides would pass for the wrong reason.
const SIDES = ['top', 'right', 'bottom', 'left'];
const CORNERS = ['top-left', 'top-right', 'bottom-right', 'bottom-left'];
const SHORTHANDS: Record<string, string[]> = {
  padding: SIDES.map((s) => `padding-${s}`),
  margin: SIDES.map((s) => `margin-${s}`),
  'border-radius': CORNERS.map((c) => `border-${c}-radius`),
  border: SIDES.flatMap((s) => [`border-${s}-width`, `border-${s}-style`, `border-${s}-color`]),
  'border-width': SIDES.map((s) => `border-${s}-width`),
  'border-color': SIDES.map((s) => `border-${s}-color`),
  outline: ['outline-width', 'outline-style', 'outline-color', 'outline-offset'],
  gap: ['row-gap', 'column-gap'],
};

export function expandProps(props: string[]): string[] {
  return props.flatMap((p) => SHORTHANDS[p] ?? [p]);
}

/** The first family named in a font-family stack, quotes and case removed. */
export function firstFamily(stack: string): string {
  const m = stack.trim().match(/^\s*(?:"([^"]*)"|'([^']*)'|([^,]+))/);
  const name = m ? (m[1] ?? m[2] ?? m[3] ?? '') : '';
  return name.trim().toLowerCase();
}

export function normalise(prop: string, value: string): string {
  if (prop === 'font-family') return firstFamily(value);
  return value.trim();
}

/** Every property on which the two disagree; empty means they match. */
export function diffStyles(mockup: StyleMap, app: StyleMap): Diff[] {
  const diffs: Diff[] = [];
  for (const prop of Object.keys(mockup)) {
    const m = normalise(prop, mockup[prop]);
    const a = normalise(prop, app[prop] ?? '');
    if (m !== a) diffs.push({ prop, mockup: m, app: a });
  }
  return diffs;
}

/**
 * Reads computed styles for the first element matching `selector`. Fails
 * loudly if there is no such element or it isn't rendered, so a selector
 * that quietly matches nothing can never make a test pass.
 */
export async function readStyles(page: Page, selector: string, props: string[], pseudo?: string): Promise<StyleMap> {
  const result = await page.evaluate(
    ({ selector, props, pseudo }) => {
      const el = document.querySelector(selector);
      if (!el) return { missing: true as const };
      const cs = getComputedStyle(el, pseudo || null);
      if (!pseudo && cs.display === 'none') return { hidden: true as const };
      const out: Record<string, string> = {};
      for (const p of props) out[p] = cs.getPropertyValue(p);
      return { styles: out };
    },
    { selector, props: expandProps(props), pseudo },
  );
  if ('missing' in result) throw new Error(`no element matches "${selector}" on ${page.url()}`);
  if ('hidden' in result) throw new Error(`"${selector}" on ${page.url()} is not displayed (display: none)`);
  return result.styles;
}

/** The mockup, open in its own tab of the same browser as the app. */
export class Mockup {
  /** Every network request the mockup tried to make; all are refused. */
  readonly refusedRequests: string[] = [];

  private constructor(readonly page: Page) {}

  static async open(context: BrowserContext): Promise<Mockup> {
    const page = await context.newPage();
    const mockup = new Mockup(page);
    // The mockup fades colours in over 150ms when its screen changes, and a
    // style read mid-fade is a blend that matches nothing. Asking for
    // reduced motion turns its transitions off, so every read is the
    // resting value.
    await page.emulateMedia({ reducedMotion: 'reduce' });
    // Its Google Fonts link is expected (SPEC B9.1) and must never need to
    // resolve. Refusing it at once, rather than waiting for a DNS failure,
    // keeps this fast and identical online and offline.
    await page.route(/^https?:/, (route) => {
      mockup.refusedRequests.push(route.request().url());
      return route.abort();
    });
    await page.goto(pathToFileURL(MOCKUP_FILE).href);
    await page.waitForSelector('#studio');
    return mockup;
  }

  /** Switches the mockup to one of its screens (briefing, kb, editor, tasks, board, team, calendar, picker). */
  async show(screen: string): Promise<void> {
    await this.page.click(`.pill[data-screen-choice="${screen}"]`);
    await expect(this.page.locator(`.pill[data-screen-choice="${screen}"]`)).toHaveAttribute('aria-pressed', 'true');
  }

  async styles(selector: string, props: string[], pseudo?: string): Promise<StyleMap> {
    return readStyles(this.page, selector, props, pseudo);
  }

  async close(): Promise<void> {
    await this.page.close();
  }
}

/**
 * Asserts that `pair.app` on the app page agrees with `pair.mockup` in the
 * mockup on every property in `props`. The failure message names each
 * property with both values, so the mockup's value is the answer.
 */
export async function expectMatchesMockup(mockup: Mockup, app: Page, pair: Pair, props: string[]): Promise<void> {
  await app.emulateMedia({ reducedMotion: 'reduce' });
  if (pair.screen) await mockup.show(pair.screen);
  const want = await mockup.styles(pair.mockup, props, pair.pseudo);
  const got = await readStyles(app, pair.app, props, pair.pseudo);
  const diffs = diffStyles(want, got);
  const report = diffs.map((d) => `  ${d.prop}: mockup ${JSON.stringify(d.mockup)}, app ${JSON.stringify(d.app)}`).join('\n');
  expect(diffs, `"${pair.app}" on ${app.url()} differs from the mockup's "${pair.mockup}":\n${report}`).toEqual([]);
}
