import fs from 'node:fs';
import path from 'node:path';
import type { Page } from '@playwright/test';

// P1-10: chosen paste/drop testing method (D-12). A synthetic
// ClipboardEvent/DragEvent with a hand-built DataTransfer proved green
// 10/10 locally and in CI — no OS clipboard, no `clipboard-read`/
// `clipboard-write` permissions, so it's not sensitive to headless
// Chromium's clipboard support varying by platform.

export type FixtureFile = { name: string; mimeType: string; base64: string };

/** Reads a file under e2e/fixtures/ into a FixtureFile the page can build a File from. */
export function loadFixtureFile(relPath: string, mimeType: string): FixtureFile {
  const abs = path.join(__dirname, '..', 'fixtures', relPath);
  const bytes = fs.readFileSync(abs);
  return { name: path.basename(relPath), mimeType, base64: bytes.toString('base64') };
}

/** A FixtureFile from bytes already in memory (e.g. a generated large file), not read from disk. */
export function fixtureFromBytes(name: string, mimeType: string, bytes: Buffer): FixtureFile {
  return { name, mimeType, base64: bytes.toString('base64') };
}

/** Dispatches a paste event carrying a file (e.g. a pasted screenshot) at the given element. */
export async function pasteFile(page: Page, selector: string, file: FixtureFile): Promise<void> {
  await page.evaluate(
    ({ selector, file }) => {
      const el = document.querySelector(selector) as HTMLElement | null;
      if (!el) throw new Error(`pasteFile: no element matches ${selector}`);
      const bytes = Uint8Array.from(atob(file.base64), (c) => c.charCodeAt(0));
      const dt = new DataTransfer();
      dt.items.add(new File([bytes], file.name, { type: file.mimeType }));
      el.dispatchEvent(new ClipboardEvent('paste', { bubbles: true, cancelable: true, clipboardData: dt }));
    },
    { selector, file },
  );
}

/** Dispatches a paste event carrying text/html (e.g. copied from a web page or Word). */
export async function pasteHTML(page: Page, selector: string, html: string): Promise<void> {
  await page.evaluate(
    ({ selector, html }) => {
      const el = document.querySelector(selector) as HTMLElement | null;
      if (!el) throw new Error(`pasteHTML: no element matches ${selector}`);
      const dt = new DataTransfer();
      dt.setData('text/html', html);
      el.dispatchEvent(new ClipboardEvent('paste', { bubbles: true, cancelable: true, clipboardData: dt }));
    },
    { selector, html },
  );
}

/** Dispatches a drop event carrying a file, the same way a dragged-in photo would arrive. */
export async function dropFile(page: Page, selector: string, file: FixtureFile): Promise<void> {
  await page.evaluate(
    ({ selector, file }) => {
      const el = document.querySelector(selector) as HTMLElement | null;
      if (!el) throw new Error(`dropFile: no element matches ${selector}`);
      const bytes = Uint8Array.from(atob(file.base64), (c) => c.charCodeAt(0));
      const dt = new DataTransfer();
      dt.items.add(new File([bytes], file.name, { type: file.mimeType }));
      const rect = el.getBoundingClientRect();
      const event = new DragEvent('drop', {
        bubbles: true,
        cancelable: true,
        clientX: rect.x + rect.width / 2,
        clientY: rect.y + rect.height / 2,
      });
      Object.defineProperty(event, 'dataTransfer', { value: dt });
      el.dispatchEvent(event);
    },
    { selector, file },
  );
}
