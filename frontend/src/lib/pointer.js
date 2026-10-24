import { onBeforeUnmount, onMounted, reactive } from 'vue'

/**
 * 页面级指针反馈：把鼠标位置写成一个 CSS 变量，交给组件做位移/高光。
 * 只写 --mx / --my 与 --tilt，不直接改样式，避免和 Vue 的响应式渲染打架。
 * 触屏与 reduced-motion 下不启用（指针没有意义，还会白耗帧）。
 */
export function usePointerTilt() {
  const state = reactive({ active: false })
  let raf = 0
  let target = null
  let px = 0
  let py = 0

  const apply = () => {
    raf = 0
    if (!target) return
    const r = target.getBoundingClientRect()
    const x = (px - r.left) / r.width
    const y = (py - r.top) / r.height
    target.style.setProperty('--mx', `${(x * 100).toFixed(1)}%`)
    target.style.setProperty('--my', `${(y * 100).toFixed(1)}%`)
    target.style.setProperty('--tilt', `${((x - 0.5) * 4).toFixed(2)}deg`)
  }

  const onMove = (e) => {
    if (!state.active) return
    const el = e.target.closest?.('[data-tilt]')
    target = el
    if (!el) return
    px = e.clientX
    py = e.clientY
    if (!raf) raf = requestAnimationFrame(apply)
  }

  const onLeave = () => {
    if (target) {
      target.style.removeProperty('--mx')
      target.style.removeProperty('--my')
      target.style.removeProperty('--tilt')
      target = null
    }
  }

  onMounted(() => {
    const fine = window.matchMedia('(hover: hover) and (pointer: fine)').matches
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (!fine || reduce) return
    state.active = true
    document.addEventListener('pointermove', onMove, { passive: true })
    document.addEventListener('pointerleave', onLeave)
  })

  onBeforeUnmount(() => {
    cancelAnimationFrame(raf)
    document.removeEventListener('pointermove', onMove)
    document.removeEventListener('pointerleave', onLeave)
  })
}
