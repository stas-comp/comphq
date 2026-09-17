// Scrolls to and highlights the block a search result linked to (SPEC
// gate 1.28), and marks every other block the query also matched. The
// server can see ?q= but never the #b-N fragment (browsers don't send
// fragments), so it marks every matching block with a data-hl attribute
// and this script decides which one to scroll to.
document.addEventListener('DOMContentLoaded', () => {
  const body = document.querySelector('.kb-article-body')
  if (!body) return

  // The title (data-b="0") lives outside .kb-article-body, as its own
  // <h1>, so highlightable blocks are searched for across the whole
  // document rather than just inside the body.
  const highlighted = []
  document.querySelectorAll('[data-hl]').forEach((el) => {
    el.dataset.original = el.innerHTML
    el.innerHTML = el.getAttribute('data-hl')
    el.classList.add('kb-highlighted')
    highlighted.push(el)
  })

  const match = /^#b-(\d+)$/.exec(window.location.hash)
  if (match) {
    const target = document.querySelector('[data-b="' + match[1] + '"]')
    if (target) {
      target.scrollIntoView({ behavior: 'smooth', block: 'center' })
      if (!target.classList.contains('kb-highlighted')) {
        target.classList.add('kb-highlighted')
        highlighted.push(target)
      }
    }
  }

  if (highlighted.length === 0) return

  const clearButton = document.createElement('button')
  clearButton.type = 'button'
  clearButton.id = 'clear-highlights'
  clearButton.textContent = 'Clear highlights'
  clearButton.addEventListener('click', () => {
    highlighted.forEach((el) => {
      if (el.dataset.original !== undefined) el.innerHTML = el.dataset.original
      el.classList.remove('kb-highlighted')
    })
    clearButton.remove()
  })
  body.parentElement.insertBefore(clearButton, body)
})
