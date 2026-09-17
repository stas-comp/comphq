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
    el.textContent = text
    el.hidden = false
  },
  clearMessage() {
    const el = document.getElementById('editor-message')
    if (!el) return
    el.hidden = true
    el.textContent = ''
  },
}
