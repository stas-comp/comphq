import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { Table, TableCell, TableHeader, TableRow } from '@tiptap/extension-table'
import Image from '@tiptap/extension-image'
import { FileHandler } from '@tiptap/extension-file-handler'

// The Knowledge Base toolbar is deliberately small (SPEC A5): two heading
// sizes, bold, italic, lists, links, simple tables, images. Everything
// StarterKit brings that isn't on that list is turned off here, once, so
// no later task can accidentally wire up an unsupported feature.
function create(element, options = {}) {
  const editor = new Editor({
    element,
    injectCSS: false,
    content: options.content || '',
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
