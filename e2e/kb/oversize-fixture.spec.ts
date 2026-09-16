import { expect, test } from '@playwright/test';
import { generateOversizePNG } from '../helpers/images';

// Sanity check for the P1-10 oversize-image generator: a real, valid PNG,
// generated at test time rather than committed. No server needed.
test('generateOversizePNG produces a real PNG of at least 21MB', () => {
  const png = generateOversizePNG();

  expect(png.length).toBeGreaterThanOrEqual(21 * 1024 * 1024);
  expect(png.subarray(0, 8)).toEqual(Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]));
  expect(png.subarray(12, 16).toString('ascii')).toBe('IHDR');
});
