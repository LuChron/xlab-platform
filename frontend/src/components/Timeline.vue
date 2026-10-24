<script setup>
defineProps({
  items: { type: Array, default: () => [] }, // [{txId, title, detail, time, tone}]
})
</script>

<template>
  <div class="timeline tl-list">
    <div
      v-for="(it, i) in items"
      :key="it.txId || i"
      class="tl-item"
      :class="{ warn: it.tone === 'warn', bad: it.tone === 'bad' }"
    >
      <span class="tl-time">{{ it.time }}</span>
      <div class="tl-body">
        <strong>{{ it.title }}</strong>
        <p v-if="it.detail">{{ it.detail }}</p>
        <span v-if="it.txId" class="tl-tx">tx {{ it.txId.slice(0, 28) }}…</span>
      </div>
    </div>
    <p v-if="!items.length" class="tl-empty">
      完成一次业务操作后，这里会出现链上轨迹。
    </p>
  </div>
</template>

<style scoped>
.timeline { display: grid; }

.tl-item {
  display: grid;
  grid-template-columns: 74px minmax(0, 1fr);
  gap: 20px;
  padding: 15px 18px;
  background: transparent;
  transition: background 0.16s ease;
}
.tl-item:hover { background: var(--surface-2); }

.tl-time {
  font-size: 14px;
  color: var(--ink-3);
  font-variant-numeric: tabular-nums;
  padding-top: 2px;
}

.tl-body strong {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: var(--ink);
  letter-spacing: -0.01em;
}

.tl-item.warn .tl-body strong { color: var(--warn); }
.tl-item.bad .tl-body strong { color: var(--bad); }

.tl-body p {
  margin-top: 5px;
  font-size: 15px;
  line-height: 1.6;
  color: var(--ink-2);
}

.tl-tx {
  display: block;
  margin-top: 6px;
  font-size: 14px;
  color: var(--ink-3);
  font-variant-numeric: tabular-nums;
}

.tl-empty { font-size: 15px; color: var(--ink-3); }

@media (max-width: 640px) {
  .tl-item { grid-template-columns: 1fr; gap: 6px; }
}
</style>
