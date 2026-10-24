<script setup>
defineProps({
  label: String,
  tone: { type: String, default: 'idle' },
})
</script>

<template>
  <span class="chip" :class="tone">{{ label }}</span>
</template>
