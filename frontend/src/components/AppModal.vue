<script setup>
import { nextTick, onMounted, onUnmounted, ref } from 'vue'

const props = defineProps({
  title: String,
  subtitle: String,
  width: { type: String, default: '600px' },
})
const emit = defineEmits(['close'])
const modal = ref(null)
const modalId = `xlab-modal-${Math.random().toString(36).slice(2, 9)}`
let returnFocus = null
let previousOverflow = ''

const focusable = [
  'button:not([disabled])',
  'a[href]',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

function onKey(e) {
  if (e.key === 'Escape') {
    emit('close')
    return
  }
  if (e.key !== 'Tab' || !modal.value) return
  const items = [...modal.value.querySelectorAll(focusable)]
  if (!items.length) {
    e.preventDefault()
    modal.value.focus()
    return
  }
  const first = items[0]
  const last = items.at(-1)
  if (e.shiftKey && document.activeElement === first) {
    e.preventDefault()
    last.focus()
  } else if (!e.shiftKey && document.activeElement === last) {
    e.preventDefault()
    first.focus()
  }
}

onMounted(async () => {
  returnFocus = document.activeElement
  window.addEventListener('keydown', onKey)
  previousOverflow = document.body.style.overflow
  document.body.style.overflow = 'hidden'
  await nextTick()
  const first = modal.value?.querySelector(focusable)
  ;(first || modal.value)?.focus()
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
  document.body.style.overflow = previousOverflow
  returnFocus?.focus?.()
})
</script>

<template>
  <div class="modal-back" @click.self="emit('close')">
    <div
      ref="modal"
      class="modal"
      :style="{ width: props.width }"
      role="dialog"
      aria-modal="true"
      :aria-labelledby="`${modalId}-title`"
      :aria-describedby="subtitle ? `${modalId}-subtitle` : undefined"
      tabindex="-1"
    >
      <div class="modal-head">
        <div>
          <h2 :id="`${modalId}-title`">{{ title }}</h2>
          <p v-if="subtitle" :id="`${modalId}-subtitle`" class="muted small" style="margin-top: 3px">{{ subtitle }}</p>
        </div>
        <button class="btn btn-icon btn-ghost" @click="emit('close')" aria-label="关闭">
          <svg viewBox="0 0 20 20" width="16" height="16" fill="none" aria-hidden="true">
            <path d="M5.5 5.5 14.5 14.5M14.5 5.5 5.5 14.5" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
          </svg>
        </button>
      </div>
      <div class="modal-body">
        <slot />
      </div>
      <div class="modal-foot" v-if="$slots.foot">
        <slot name="foot" />
      </div>
    </div>
  </div>
</template>
