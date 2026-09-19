// User-facing wording for JS-originated messages across the Knowledge
// Base, kept out of the scripts that use them (SPEC B1: "User-facing
// wording lives in the templates, not scattered through Go code" — native
// confirm()/beforeunload dialogs and client-only messages can't live in a
// template, so this file, loaded on every page, is that same single place
// for them). Upload failure messages (too large, wrong type) come from
// the server response body instead, since the 20 MB and file-type rules
// live in internal/kb/images — one source of truth, not duplicated here.
window.ComphqMessages = {
  LEAVE_WITHOUT_SAVING: 'Leave without saving?',
  REPLACE_EDITOR_CONTENT: "Replace what's in the editor with this document?",
  WORD_PICTURES_MISSING: "Some pictures from Word couldn't be pasted. Use Import from Word to bring them in.",
  // SPEC gate 1.49: "A message lists what came across and what didn't,
  // for example '1 thing couldn't be brought in: a chart.'" notes is the
  // server's own list of human-readable phrases (docx.Result.Notes).
  // Gate 4.43: the import summary is a notice, headed by what came in.
  docxImported(fileName) {
    return fileName + ' is in the editor. Nothing is published until you press Publish.'
  },
  DISMISS: 'Dismiss',
  IMPORTED: 'Imported',
  // Gate 4.42: the foot of the search results panel.
  searchFoot(count) {
    return count + (count === 1 ? ' article' : ' articles') + ' · press Enter to see all results'
  },
  docxImportSummary(notes) {
    const noun = notes.length === 1 ? 'thing' : 'things'
    return notes.length + ' ' + noun + " couldn't be brought in: " + notes.join(', ') + '.'
  },
  // SPEC gate 1.32 (the asterisks around "words" in the SPEC text are its
  // own markdown emphasis, not literal characters to render).
  noArticlesMatch(query) {
    const span = document.createElement('span')
    span.textContent = query
    return 'No articles match <em>' + span.innerHTML + '</em>.'
  },
}

// A single visible message area for client-only feedback (SPEC gate 1.17:
// "shows a plain message and nothing else changes").
window.ComphqUI = {
  showMessage(text) {
    const el = document.getElementById('editor-message')
    if (!el) return
    el.className = 'message'
    el.setAttribute('role', 'alert')
    el.textContent = text
    el.hidden = false
  },
  // The import summary (gate 4.43): a bordered notice with an IMPORTED
  // stamp, what came in, anything that couldn't, and a way to dismiss it.
  showImported(fileName, notes) {
    const el = document.getElementById('editor-message')
    if (!el) return
    el.textContent = ''
    el.className = 'imported'
    el.setAttribute('role', 'status')
    const stamp = document.createElement('span')
    stamp.className = 'stamp stamp-yours'
    stamp.textContent = window.ComphqMessages.IMPORTED
    const text = document.createElement('div')
    const lead = document.createElement('b')
    lead.textContent = fileName
    text.append(lead, ' ' + window.ComphqMessages.docxImported('').trim())
    if (notes && notes.length > 0) {
      const list = document.createElement('ul')
      const item = document.createElement('li')
      item.textContent = window.ComphqMessages.docxImportSummary(notes)
      list.appendChild(item)
      text.appendChild(list)
    }
    const dismiss = document.createElement('button')
    dismiss.type = 'button'
    dismiss.className = 'mini quiet'
    dismiss.textContent = window.ComphqMessages.DISMISS
    dismiss.addEventListener('click', () => window.ComphqUI.clearMessage())
    el.append(stamp, text, dismiss)
    el.hidden = false
  },
  clearMessage() {
    const el = document.getElementById('editor-message')
    if (!el) return
    el.hidden = true
    el.className = 'message'
    el.textContent = ''
  },
}
