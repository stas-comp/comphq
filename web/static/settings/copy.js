// Copy buttons on the Setup page (SPEC B6): Clipboard API first, with a
// select-text fallback for browsers/contexts where it's unavailable.
document.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('.copy-button').forEach((button) => {
    button.addEventListener('click', async () => {
      const target = document.getElementById(button.dataset.target)
      if (!target) return
      const text = target.textContent || ''
      const status = document.getElementById('copy-status')

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
  })
})
