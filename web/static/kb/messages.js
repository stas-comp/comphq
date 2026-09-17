// User-facing wording for the editor's JS-originated dialogs, kept out of
// editor.js itself (SPEC B1: "User-facing wording lives in the templates,
// not scattered through Go code" — native confirm()/beforeunload dialogs
// can't live in a template, so this file is that same single place for
// them). Later Knowledge Base editor work (paste clean-up, image upload
// failures) adds to this rather than inlining new strings elsewhere.
window.ComphqMessages = {
  LEAVE_WITHOUT_SAVING: 'Leave without saving?',
}
