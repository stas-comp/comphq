// The real Knowledge Base editor (SPEC gates 1.15, 1.16, 1.17, 1.20),
// built on the P1-09 bundle.
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

  const WORD_MIME = 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
  const isWordFile = (file) => file.type === WORD_MIME || /\.docx$/i.test(file.name)

  // Uploads one file and inserts it as an image (SPEC gates 1.16, 1.17).
  // pos is given for a drop (insert exactly where it landed) and omitted
  // for a paste or the toolbar's file chooser (insert at the cursor). A
  // .docx is routed to Import from Word, not upload — that command isn't
  // built until P1-33, so it shows a "coming soon" message for now.
  async function uploadAndInsert(file, pos) {
    if (isWordFile(file)) {
      window.ComphqUI.showMessage(window.ComphqMessages.WORD_IMPORT_COMING_SOON)
      return
    }
    try {
      const res = await fetch('/kb/images', {
        method: 'POST',
        body: file,
        headers: { 'Content-Type': file.type || 'application/octet-stream' },
      })
      if (!res.ok) {
        window.ComphqUI.showMessage(await res.text())
        return
      }
      const data = await res.json()
      window.ComphqUI.clearMessage()
      markDirty()
      const chain = editor.chain().focus()
      if (typeof pos === 'number') {
        chain.insertContentAt(pos, { type: 'image', attrs: { src: data.src, alt: file.name } }).run()
      } else {
        chain.setImage({ src: data.src, alt: file.name }).run()
      }
    } catch (err) {
      window.ComphqUI.showMessage('That file could not be uploaded.')
    }
  }

  const editor = window.ComphqEditor.create(container, {
    content: initialContent,
    onPaste: (_editor, files) => {
      for (const file of files) uploadAndInsert(file)
    },
    onDrop: (_editor, files, pos) => {
      for (const file of files) uploadAndInsert(file, pos)
    },
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
    uploadAndInsert(file)
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
