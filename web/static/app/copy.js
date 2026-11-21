// Copy buttons, shared by the Setup page and the COSMOS places on a job (SPEC
// B6, B14.2). One click handler on the document, so a button that arrives later
// (a section swapped in by a script, the task window's fragment) works too.
//
// The text to copy is either the text of the element named by data-target (the
// Setup page's commands) or the button's own data-copy (a place's path).
//
// The Clipboard API only exists on a secure page: on the NAS's ordinary
// plain-HTTP address (no padlock) there is no navigator.clipboard. So it is tried
// first, then a hidden text box and the browser's own copy command (which works
// from a click on a plain page), and only then is the text selected for Ctrl+C.
(function () {
  function copyWithSelection(text, host) {
    const box = document.createElement('textarea')
    box.value = text
    box.setAttribute('readonly', '')
    box.setAttribute('aria-hidden', 'true')
    box.style.position = 'fixed'
    box.style.top = '0'
    box.style.left = '0'
    box.style.opacity = '0'
    // Inside the open window if there is one: a modal dialog makes everything
    // outside it inert, and an inert box can't be selected and copied.
    ;(host || document.body).appendChild(box)
    box.focus()
    box.select()
    let ok = false
    try {
      ok = document.execCommand('copy')
    } catch {
      ok = false
    }
    box.remove()
    return ok
  }

  async function copyText(text, host) {
    if (navigator.clipboard && window.isSecureContext !== false) {
      try {
        await navigator.clipboard.writeText(text)
        return true
      } catch {
        // fall through to the older way
      }
    }
    return copyWithSelection(text, host)
  }

  document.addEventListener('click', async (event) => {
    const button = event.target instanceof Element ? event.target.closest('.copy-button') : null
    if (!button) return
    const status = document.getElementById('copy-status')

    // A place's path: the button itself says what to copy and what to read after.
    if (button.dataset.copy !== undefined) {
      const text = button.dataset.copy
      const ok = await copyText(text, button.closest('dialog'))
      const label = button.dataset.label || button.textContent
      button.dataset.label = label
      if (ok) {
        button.textContent = button.dataset.copiedLabel || 'Copied'
        clearTimeout(button._copyTimer)
        button._copyTimer = setTimeout(() => {
          button.textContent = label
        }, 2000)
      } else {
        // Nothing worked: select the path so Ctrl+C does.
        const line = button.closest('.ref-place')
        const path = line && line.querySelector('.ref-path')
        if (path) {
          const range = document.createRange()
          range.selectNodeContents(path)
          const selection = window.getSelection()
          selection.removeAllRanges()
          selection.addRange(range)
        }
        button.textContent = 'Press Ctrl+C'
        clearTimeout(button._copyTimer)
        button._copyTimer = setTimeout(() => {
          button.textContent = label
        }, 2000)
      }
      return
    }

    const target = document.getElementById(button.dataset.target)
    if (!target) return
    const text = target.textContent || ''
    try {
      await navigator.clipboard.writeText(text)
      if (status) status.textContent = 'Copied.'
    } catch {
      const range = document.createRange()
      range.selectNodeContents(target)
      const selection = window.getSelection()
      selection.removeAllRanges()
      selection.addRange(range)
      if (status) status.textContent = 'Selected — press Ctrl+C to copy.'
    }
  })
})()
