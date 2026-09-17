import http from 'node:http';

// A tiny local "external" image host for SPEC gate 1.19: the comphq
// process under test is allowed to fetch from server.stubImageHost (and
// only that address — see e2e/helpers/fixtures.ts), so a stub bound to
// exactly that host:port stands in for a real website's image.
export type StubImageServer = { url: string; close: () => Promise<void> };

// A URL that's guaranteed to fail an image fetch identically in every
// environment (local dev, CI, and a fully offline container with no
// network at all): the app's own SSRF guard refuses to dial any loopback
// address, so this fails fast and deterministically without needing a
// real network round trip or a per-worker stub server. Prefer this over
// startStubImageServer whenever a test only needs *some* fetch failure,
// not proof that a reachable one succeeds (PLAN.md P1-38: the full E2E
// suite must also pass against the genuinely offline container).
export const UNREACHABLE_IMAGE_URL = 'http://127.0.0.1:1/gone.png';

export async function startStubImageServer(hostPort: string, imageBytes: Buffer, contentType = 'image/png'): Promise<StubImageServer> {
  const [host, portStr] = hostPort.split(':');
  const port = Number(portStr);

  const server = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': contentType });
    res.end(imageBytes);
  });

  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(port, host, resolve);
  });

  return {
    url: `http://${hostPort}/picture.png`,
    close: () => new Promise((resolve) => server.close(() => resolve())),
  };
}
