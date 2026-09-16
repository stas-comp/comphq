import { deflateSync } from 'node:zlib';

// A minimal, dependency-free PNG writer, used only to generate an oversize
// fixture at test time (P1-10): committing a real 21 MB file isn't worth
// it when a few dozen lines of zlib + CRC32 can build one on demand.

const CRC_TABLE = (() => {
  const table = new Uint32Array(256);
  for (let n = 0; n < 256; n++) {
    let c = n;
    for (let k = 0; k < 8; k++) {
      c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    }
    table[n] = c >>> 0;
  }
  return table;
})();

function crc32(buf: Buffer): number {
  let c = 0xffffffff;
  for (let i = 0; i < buf.length; i++) {
    c = CRC_TABLE[(c ^ buf[i]) & 0xff] ^ (c >>> 8);
  }
  return (c ^ 0xffffffff) >>> 0;
}

function chunk(type: string, data: Buffer): Buffer {
  const typeBuf = Buffer.from(type, 'ascii');
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length, 0);
  const crcInput = Buffer.concat([typeBuf, data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(crcInput), 0);
  return Buffer.concat([length, typeBuf, data, crc]);
}

/**
 * Builds a real, valid greyscale PNG at least minBytes long. Pixel data is
 * pseudo-random so it doesn't compress away to nothing, which is what lets
 * us reach a predictable multi-megabyte size without a huge canvas.
 */
export function generateOversizePNG(minBytes = 21 * 1024 * 1024): Buffer {
  const PNG_SIGNATURE = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);

  // Compressed random data runs a bit under 1:1, so oversize the raw
  // buffer a little and trim once we know the real compressed length.
  const width = 4096;
  const bytesPerRow = width + 1; // +1 for the per-row filter-type byte
  const rows = Math.ceil((minBytes * 1.05) / bytesPerRow);
  const height = rows;

  const raw = Buffer.alloc(rows * bytesPerRow);
  // A simple xorshift32 PRNG: fast, deterministic, and good enough that
  // deflate can't compress the output meaningfully.
  let seed = 0x2f6e2b1;
  for (let i = 0; i < raw.length; i++) {
    seed ^= seed << 13;
    seed ^= seed >>> 17;
    seed ^= seed << 5;
    seed |= 0;
    if (i % bytesPerRow === 0) {
      raw[i] = 0; // filter type "None" for this row
    } else {
      raw[i] = seed & 0xff;
    }
  }

  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8; // bit depth
  ihdr[9] = 0; // colour type: greyscale
  ihdr[10] = 0; // compression method
  ihdr[11] = 0; // filter method
  ihdr[12] = 0; // interlace method

  const idat = deflateSync(raw, { level: 1 });

  return Buffer.concat([
    PNG_SIGNATURE,
    chunk('IHDR', ihdr),
    chunk('IDAT', idat),
    chunk('IEND', Buffer.alloc(0)),
  ]);
}
