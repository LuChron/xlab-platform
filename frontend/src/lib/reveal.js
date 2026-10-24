import { onBeforeUnmount, onMounted } from 'vue'

/**
 * 滚动入场：给每个 [data-reveal] 元素在进入视口时加 .in 类。
 * 只改 opacity / transform，遇 reduced-motion 直接显示（样式在 main.css 里兜）。
 * 每个视图挂一次，观察范围内所有带标记的元素，包括异步渲染出来的清单行。
 */
export function useReveal() {
  let observer = null

  const scan = () => {
    if (!observer) return
    document.querySelectorAll('[data-reveal]:not(.in)').forEach((el) => observer.observe(el))
  }

  onMounted(() => {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      document.querySelectorAll('[data-reveal]').forEach((el) => el.classList.add('in'))
      return
    }

    observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue
          const el = entry.target
          const delay = Number(el.dataset.delay || 0)
          setTimeout(() => el.classList.add('in'), delay)
          observer.unobserve(el)
        }
      },
      { threshold: 0.08, rootMargin: '0px 0px -6% 0px' }
    )
    scan()

    // 数据到达后新渲染的行也要观察
    const mo = new MutationObserver(() => scan())
    mo.observe(document.body, { childList: true, subtree: true })
    onBeforeUnmount(() => mo.disconnect())
  })

  onBeforeUnmount(() => observer?.disconnect())
}
