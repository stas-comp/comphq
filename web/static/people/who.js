document.addEventListener('DOMContentLoaded', () => {
  const link = document.getElementById('add-name-link')
  const form = document.getElementById('add-name-form')
  if (link && form) {
    link.addEventListener('click', () => {
      link.hidden = true
      form.hidden = false
      const input = document.getElementById('add-name-input')
      if (input) input.focus()
    })
  }
  document.body.setAttribute('data-ready', '')
})
