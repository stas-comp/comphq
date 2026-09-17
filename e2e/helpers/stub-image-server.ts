import http from 'node:http';

// A tiny local "external" image host for SPEC gate 1.19: the comphq
// process under test is allowed to fetch from server.stubImageHost (and
// only that address — see e2e/helpers/fixtures.ts), so a stub bound to
// exactly that host:port stands in for a real website's image.
export type StubImageServer = { url: string; close: () => Promise<void> };

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
