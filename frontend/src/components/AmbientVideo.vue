<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  src: { type: String, required: true },
  poster: { type: String, required: true },
  active: { type: Boolean, default: true },
  videoClass: { type: [String, Array, Object], default: '' },
})

const video = ref(null)
const paused = ref(false)
let mediaQuery

function syncMotion() {
  if (!video.value) return
  if (!props.active || mediaQuery?.matches || paused.value) video.value.pause()
  else video.value.play().catch(() => {})
}

function toggle() {
  paused.value = !video.value?.paused
  syncMotion()
}

function onPreferenceChange() {
  paused.value = mediaQuery.matches
  syncMotion()
}

watch(() => props.active, syncMotion)

onMounted(() => {
  mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  paused.value = mediaQuery.matches
  mediaQuery.addEventListener?.('change', onPreferenceChange)
  syncMotion()
})

onBeforeUnmount(() => mediaQuery?.removeEventListener?.('change', onPreferenceChange))
</script>

<template>
  <video
    ref="video"
    v-bind="$attrs"
    :class="[videoClass, { on: active }]"
    :src="src"
    :poster="poster"
    autoplay
    muted
    loop
    playsinline
    preload="metadata"
    aria-hidden="true"
  />
  <button
    v-if="active"
    type="button"
    class="media-toggle"
    :aria-label="paused ? '播放动态画面' : '暂停动态画面'"
    :title="paused ? '播放' : '暂停'"
    @click="toggle"
  >
    <span aria-hidden="true">{{ paused ? '▶' : 'Ⅱ' }}</span>
  </button>
</template>
