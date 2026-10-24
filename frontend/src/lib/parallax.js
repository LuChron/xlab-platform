import { onBeforeUnmount, onMounted } from 'vue'

/**
 * 滚动视差：图片在它所在的容器里缓慢错位，页面滚动时景物比文字走得慢一点。
 * 只写一个 CSS 变量 --py（像素），位移交给样式；rAF 合并同一帧的多次 scroll。
 * 触屏与 reduced-motion 下不启用。
 */
export function useParallax(selector = '[data-parallax]', strength = 0.09) {
  const items = []
  let raf = 0
  let enabled = false

  const measure = () => {
    items.length = 0
    document.querySelectorAll(selector).forEach((el) => {
      // 一个容器可能装多张图（实景带），全部跟着容器一起走，避免同一条带子里只有一张在动
      el.querySelectorAll('img, video').forEach((inner) => items.push({ el, inner }))
    })
  }

  const update = () => {
    raf = 0
    if (!enabled) return
    const vh = window.innerHeight
    for (const { el, inner } of items) {
      const r = el.getBoundingClientRect()
      if (r.bottom < -240 || r.top > vh + 240) continue
      // -1（刚进视口底部）→ 1（已出视口顶部）
      const p = (r.top + r.height / 2 - vh / 2) / vh
      inner.style.setProperty('--py', `${(p * strength * 100).toFixed(2)}px`)
    }
  }

  const schedule = () => {
    if (!raf) raf = requestAnimationFrame(update)
  }

  onMounted(() => {
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    const coarse = window.matchMedia('(hover: none)').matches
    if (reduce || coarse) return
    enabled = true
    measure()
    window.addEventListener('scroll', schedule, { passive: true })
    window.addEventListener('resize', schedule)
    schedule()
    // 异步渲染出来的图也要量
    const mo = new MutationObserver(() => {
      measure()
      schedule()
    })
    mo.observe(document.body, { childList: true, subtree: true })
    onBeforeUnmount(() => mo.disconnect())
  })

  onBeforeUnmount(() => {
    cancelAnimationFrame(raf)
    window.removeEventListener('scroll', schedule)
    window.removeEventListener('resize', schedule)
  })
}
