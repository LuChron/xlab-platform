<script setup>
// 设备类别图标：统一线性风格，1.5px 描边，与品牌气质一致
defineProps({
  category: { type: String, default: '' },
})
</script>

<template>
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
    <!-- 计算设备 -->
    <template v-if="category === '计算设备'">
      <rect x="3" y="4" width="18" height="12" rx="1.6" />
      <path d="M2 19h20" />
    </template>
    <!-- 显示设备 -->
    <template v-else-if="category === '显示设备'">
      <rect x="2.5" y="4" width="19" height="12.5" rx="1.6" />
      <path d="M8.5 20.5h7M12 16.5v4" />
    </template>
    <!-- 影音设备 -->
    <template v-else-if="category === '影音设备'">
      <rect x="2.5" y="7" width="14" height="10" rx="2" />
      <path d="M16.5 11.5l4.5-2.6v6.2l-4.5-2.6z" />
      <circle cx="7" cy="12" r="1.6" />
    </template>
    <!-- 工程设备 -->
    <template v-else-if="category === '工程设备'">
      <circle cx="12" cy="12" r="3" />
      <path d="M12 2.5v3M12 18.5v3M2.5 12h3M18.5 12h3M5.2 5.2l2.1 2.1M16.7 16.7l2.1 2.1M18.8 5.2l-2.1 2.1M7.3 16.7l-2.1 2.1" />
    </template>
    <!-- 展示设备 -->
    <template v-else-if="category === '展示设备'">
      <rect x="2.5" y="4" width="19" height="13" rx="1.6" />
      <path d="M7 20.5l5-3.5 5 3.5" />
    </template>
    <!-- 协作工具 -->
    <template v-else-if="category === '协作工具'">
      <rect x="3" y="4.5" width="18" height="15" rx="1.6" />
      <path d="M7.5 9.5h9M7.5 14h5.5" />
    </template>
    <!-- 存储设备 -->
    <template v-else-if="category === '存储设备'">
      <rect x="3.5" y="4" width="17" height="7" rx="1.4" />
      <rect x="3.5" y="13" width="17" height="7" rx="1.4" />
      <path d="M7 7.5h.01M7 16.5h.01" />
    </template>
    <!-- 兜底 -->
    <template v-else>
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <path d="M9 9h6v6H9z" />
    </template>
  </svg>
</template>
