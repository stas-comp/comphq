// User-facing wording for the editor's JS-originated messages, kept out of
// editor.js itself (SPEC B1: "User-facing wording lives in the templates,
// not scattered through Go code" — native confirm()/beforeunload dialogs
// and client-only messages can't live in a template, so this file is that
// same single place for them). Upload failure messages (too large, wrong
// type) come from the server response body instead, since the 20 MB and
// file-type rules live in internal/kb/images — one source of truth, not
// duplicated here.
window.ComphqMessages = {
  LEAVE_WITHOUT_SAVING: 'Leave without saving?',
  WORD_IMPORT_COMING_SOON: "Import from Word is coming soon. Word documents can't be added yet.",
  WORD_PICTURES_MISSING: "Some pictures from Word couldn't be pasted. Use Import from Word to bring them in.",
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
