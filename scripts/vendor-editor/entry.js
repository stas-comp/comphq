import { Editor, Node, mergeAttributes } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { Table, TableCell, TableHeader, TableRow } from '@tiptap/extension-table'
import Image from '@tiptap/extension-image'
import { FileHandler } from '@tiptap/extension-file-handler'

// SPEC B4: a picture pasted from Word (an <img src="file:..."> the browser
// can never load, since Word doesn't hand the actual bytes to the paste
// event) becomes this placeholder instead. editor.js's transformPastedHTML
// hook does the file: -> div[data-missing-kind] rewrite on the raw pasted
// HTML string before ProseMirror ever parses it; this node just makes the
// resulting div a real, persistent part of the document (surviving both
// re-parsing and getHTML()) rather than an unrecognised element ProseMirror
// would otherwise drop.
const MissingPicture = Node.create({
  name: 'missingPicture',
  group: 'block',
  atom: true,
  selectable: true,
  addAttributes() {
    return {
      'data-missing-kind': { default: 'picture' },
      'data-missing-src': { default: null },
    }
  },
  parseHTML() {
    return [{ tag: 'div[data-missing-kind]' }]
  },
  renderHTML({ HTMLAttributes }) {
    return ['div', mergeAttributes(HTMLAttributes)]
  },
})

// The Knowledge Base toolbar is deliberately small (SPEC A5): two heading
// sizes, bold, italic, lists, links, simple tables, images. Everything
// StarterKit brings that isn't on that list is turned off here, once, so
// no later task can accidentally wire up an unsupported feature.
function create(element, options = {}) {
  const editor = new Editor({
    element,
    injectCSS: false,
    content: options.content || '',
    editorProps: {
      transformPastedHTML(html) {
        if (typeof options.transformPastedHTML === 'function') {
          return options.transformPastedHTML(html)
        }
        return html
      },
    },
    extensions: [
      StarterKit.configure({
        heading: { levels: [2, 3] },
        link: { protocols: ['http', 'https', 'mailto'] },
        blockquote: false,
        code: false,
        codeBlock: false,
        horizontalRule: false,
        strike: false,
        underline: false,
      }),
      Table.configure({ resizable: false }),
      TableRow,
      TableHeader,
      TableCell,
      Image,
      MissingPicture,
      FileHandler.configure({
        onPaste: (currentEditor, files, htmlContent) => {
          if (typeof options.onPaste === 'function') {
            options.onPaste(currentEditor, files, htmlContent)
          }
        },
        onDrop: (currentEditor, files, pos) => {
          if (typeof options.onDrop === 'function') {
            options.onDrop(currentEditor, files, pos)
          }
        },
      }),
    ],
  })
  return editor
}

window.ComphqEditor = { create }
