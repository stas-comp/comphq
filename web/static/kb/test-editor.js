// Glue code for the P1-09 spike harness (GET /__test/editor, COMPHQ_TEST_MODE
// only). Exercises every toolbar command the Knowledge Base editor needs,
// plus the FileHandler paste/drop hooks, under the app's real CSP.
document.addEventListener('DOMContentLoaded', () => {
  const el = document.getElementById('editor')
  const pasteLog = document.getElementById('paste-log')

  const editor = window.ComphqEditor.create(el, {
    onPaste: (_editor, files) => {
      pasteLog.textContent = files.map((f) => f.name || f.type).join(',')
      pasteLog.dataset.pasted = 'true'
    },
    onDrop: (_editor, files) => {
      pasteLog.textContent = files.map((f) => f.name || f.type).join(',')
      pasteLog.dataset.dropped = 'true'
    },
  })

  const bind = (id, fn) => document.getElementById(id).addEventListener('click', fn)

  bind('btn-h2', () => editor.chain().focus().toggleHeading({ level: 2 }).run())
  bind('btn-h3', () => editor.chain().focus().toggleHeading({ level: 3 }).run())
  bind('btn-bold', () => editor.chain().focus().toggleBold().run())
  bind('btn-italic', () => editor.chain().focus().toggleItalic().run())
  bind('btn-bullet', () => editor.chain().focus().toggleBulletList().run())
  bind('btn-ordered', () => editor.chain().focus().toggleOrderedList().run())
  // mailto avoids an http(s) literal in a shipped static file (repository
  // policy: no raw http(s) URLs outside web/static/vendor), while still
  // proving the link toolbar command against one of the allowed protocols.
  bind('btn-link', () => editor.chain().focus().extendMarkRange('link').setLink({ href: 'mailto:sam@example.test' }).run())
  bind('btn-table', () => editor.chain().focus().insertTable({ rows: 2, cols: 2, withHeaderRow: true }).run())
  bind('btn-add-row', () => editor.chain().focus().addRowAfter().run())
  bind('btn-remove-row', () => editor.chain().focus().deleteRow().run())
  bind('btn-add-col', () => editor.chain().focus().addColumnAfter().run())
  bind('btn-remove-col', () => editor.chain().focus().deleteColumn().run())
  bind('btn-image', () => editor.chain().focus().setImage({ src: '/static/kb/test-fixture.png', alt: 'test image' }).run())

  document.body.setAttribute('data-ready', '')
})
