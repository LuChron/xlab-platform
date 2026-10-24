export function scrollElementIntoView(element, block = 'start') {
  if (!element) return
  const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  element.scrollIntoView({ behavior: reduce ? 'auto' : 'smooth', block })
}
