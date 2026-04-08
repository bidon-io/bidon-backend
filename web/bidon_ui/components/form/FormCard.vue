<template>
  <div class="card mb-6">
    <div v-if="props.title" class="card-header">
      <span class="font-semibold text-gray-800">{{ props.title }}</span>
    </div>
    <div class="divide-y divide-gray-100">
      <slot></slot>
    </div>
    <div v-if="$slots.footer" class="card-footer flex justify-end">
      <slot name="footer"></slot>
    </div>
  </div>
</template>

<script setup>
const props = defineProps({
  title: {
    type: String,
    default: "",
  },
});
</script>
