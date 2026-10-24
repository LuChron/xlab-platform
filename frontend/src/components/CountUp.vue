<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

/**
 * 数字进场：值到位时从 0 滚到目标值。
 * 只在真正进入视口后才跑，避免首屏之外的数字空转；reduced-motion 直接给终值。
 */
const props = defineProps({
  to: { type: Number, default: 0 },
  suffix: { type: String, default: '' },
  duration: { type: Number, default: 900 },
})

const el = ref(null)
const shown = ref(0)
let frame = 0
let io = null
let started = false

const reduce = () => window.matchMedia('(prefers-reduced-motion: reduce)').matches

function run(target) {
  cancelAnimationFrame(frame)
  if (reduce()) {
    shown.value = target
    return
  }
  const from = shown.value
  const start = performance.now()
  const tick = (now) => {
    const p = Math.min(1, (now - start) / props.duration)
    // easeOutCubic：起手快、收尾稳，读数时不会觉得数字在爬
    const eased = 1 - Math.pow(1 - p, 3)
    shown.value = Math.round(from + (target - from) * eased)
    if (p < 1) frame = requestAnimationFrame(tick)
  }
  frame = requestAnimationFrame(tick)
}

onMounted(() => {
  io = new IntersectionObserver(
    (entries) => {
      if (!entries[0].isIntersecting || started) return
      started = true
      run(Math.round(props.to))
      io.disconnect()
    },
    { threshold: 0.4 }
  )
  if (el.value) io.observe(el.value)
})

watch(
  () => props.to,
  (v) => {
    if (started) run(Math.round(v || 0))
  }
)

onBeforeUnmount(() => {
  cancelAnimationFrame(frame)
  io?.disconnect()
})
</script>

<template>
  <span ref="el" class="count-up">{{ shown }}<i v-if="suffix">{{ suffix }}</i></span>
</template>

<style scoped>
.count-up { font-variant-numeric: tabular-nums; }
.count-up i { font-style: normal; }
</style>
