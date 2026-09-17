// Copies SortableJS's own prebuilt, already-minified bundle into
// web/static/vendor/sortable (dev only). Unlike the editor bundle
// (scripts/vendor-editor/build.mjs), there's nothing to bundle — npm's
// package already ships one minified file — so this just copies it,
// pinned to an exact version in its filename, and records its checksum.
import { createHash } from 'node:crypto'
import fs from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..')
const version = JSON.parse(fs.readFileSync(path.join(root, 'node_modules', 'sortablejs', 'package.json'), 'utf8')).version

const outDir = path.join(root, 'web', 'static', 'vendor', 'sortable')
const outFile = path.join(outDir, `Sortable-${version}.min.js`)

fs.mkdirSync(outDir, { recursive: true })
fs.copyFileSync(path.join(root, 'node_modules', 'sortablejs', 'Sortable.min.js'), outFile)
console.log(`wrote web/static/vendor/sortable/Sortable-${version}.min.js`)

fs.copyFileSync(path.join(root, 'node_modules', 'sortablejs', 'LICENSE'), path.join(outDir, 'LICENSE.txt'))
console.log('wrote web/static/vendor/sortable/LICENSE.txt')

writeVendorEntry()
console.log('updated web/static/vendor/VENDOR.md')

function writeVendorEntry() {
  const bytes = fs.readFileSync(outFile)
  const sha256 = createHash('sha256').update(bytes).digest('hex')
  const vendorMdPath = path.join(root, 'web', 'static', 'vendor', 'VENDOR.md')
  const name = `Sortable-${version}.min.js`

  const row = `| ${name} | SortableJS ${version} | MIT | npm: sortablejs | \`sortable/${name}\` | ${sha256} |`

  const header =
    '# Vendored third-party files\n\n' +
    'Recomputed by the policy tests (`tools/policy`), which fail if a checksum here stops matching the file.\n\n' +
    '| Name | Version | Licence | Source | Path | SHA-256 |\n' +
    '|---|---|---|---|---|---|\n'

  let rows = [row]
  if (fs.existsSync(vendorMdPath)) {
    const existing = fs.readFileSync(vendorMdPath, 'utf8')
    const lines = existing.split('\n')
    const dataLines = lines.filter(
      (l) => l.startsWith('|') && !l.startsWith('| Name') && !l.startsWith('|---'),
    )
    const otherRows = dataLines.filter((l) => !l.includes(name))
    rows = [...otherRows, row]
  }
  // Sorted by name (not appended at the end): another vendor script
  // touching this same shared file, run in either order, must produce
  // byte-identical output, or a "bundle is reproducible" CI check that
  // only re-runs one script and diffs the file sees a false failure.
  rows.sort()
  const content = header + rows.join('\n') + '\n'

  fs.writeFileSync(vendorMdPath, content)
}
