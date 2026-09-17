// The real Knowledge Base editor (SPEC gates 1.15, 1.20), built on the
// P1-09 bundle. Image upload is P1-21's job: for now the toolbar button
// inserts a local preview so the button and file chooser exist and work,
// but a blob: URL never survives Sanitize (SPEC B4 drops any img not
// already under /images/), so publishing without a real upload in place
// loses the picture — expected until P1-21 lands.
document.addEventListener('DOMContentLoaded', () => {
  const container = document.getElementById('article-editor')
  const form = document.getElementById('article-form')
  const hiddenBodyHTML = document.getElementById('article-body-html')
  const titleInput = document.getElementById('article-title')
  const categorySelect = document.getElementById('article-category')
  const imageFileInput = document.getElementById('image-file-input')

  let dirty = false
  const markDirty = () => { dirty = true }

  const initialContentEl = document.getElementById('article-initial-body-html')
  const initialContent = initialContentEl ? JSON.parse(initialContentEl.textContent) : ''

  const editor = window.ComphqEditor.create(container, {
    content: initialContent,
  })
  editor.on('update', markDirty)
  titleInput.addEventListener('input', markDirty)
  categorySelect.addEventListener('change', markDirty)

  const bind = (id, fn) => document.getElementById(id).addEventListener('click', fn)

  bind('btn-h2', () => editor.chain().focus().toggleHeading({ level: 2 }).run())
  bind('btn-h3', () => editor.chain().focus().toggleHeading({ level: 3 }).run())
  bind('btn-bold', () => editor.chain().focus().toggleBold().run())
  bind('btn-italic', () => editor.chain().focus().toggleItalic().run())
  bind('btn-bullet', () => editor.chain().focus().toggleBulletList().run())
  bind('btn-ordered', () => editor.chain().focus().toggleOrderedList().run())
  bind('btn-link', () => {
    const url = window.prompt('Link address (http, https or mailto):')
    if (!url) return
    editor.chain().focus().extendMarkRange('link').setLink({ href: url }).run()
  })
  bind('btn-table', () => editor.chain().focus().insertTable({ rows: 2, cols: 2, withHeaderRow: true }).run())
  bind('btn-add-row', () => editor.chain().focus().addRowAfter().run())
  bind('btn-remove-row', () => editor.chain().focus().deleteRow().run())
  bind('btn-add-col', () => editor.chain().focus().addColumnAfter().run())
  bind('btn-remove-col', () => editor.chain().focus().deleteColumn().run())

  bind('btn-image', () => imageFileInput.click())
  imageFileInput.addEventListener('change', () => {
    const file = imageFileInput.files && imageFileInput.files[0]
    imageFileInput.value = ''
    if (!file) return
    const src = URL.createObjectURL(file)
    editor.chain().focus().setImage({ src, alt: file.name }).run()
  })

  // The four table-editing buttons only make sense with the cursor inside
  // a table (PLAN.md P1-20: "Add row, Remove row, Add column, Remove
  // column when inside a table").
  const tableButtons = {
    'btn-add-row': () => editor.can().addRowAfter(),
    'btn-remove-row': () => editor.can().deleteRow(),
    'btn-add-col': () => editor.can().addColumnAfter(),
    'btn-remove-col': () => editor.can().deleteColumn(),
  }
  const updateTableButtons = () => {
    for (const [id, canRun] of Object.entries(tableButtons)) {
      document.getElementById(id).disabled = !canRun()
    }
  }
  editor.on('selectionUpdate', updateTableButtons)
  editor.on('transaction', updateTableButtons)
  updateTableButtons()

  form.addEventListener('submit', () => {
    hiddenBodyHTML.value = editor.getHTML()
    dirty = false
  })

  window.addEventListener('beforeunload', (event) => {
    if (!dirty) return
    event.preventDefault()
    event.returnValue = ''
  })

  document.getElementById('btn-cancel').addEventListener('click', () => {
    if (dirty && !window.confirm(window.ComphqMessages.LEAVE_WITHOUT_SAVING)) return
    dirty = false
    window.location.href = '/kb'
  })

  document.body.setAttribute('data-editor-ready', '')
})
