<template>
  <div class="card mb-6">
    <div v-if="props.title" class="card-header">
      <span class="font-semibold" style="color: var(--bidon-text-primary);">{{ props.title }}</span>
    </div>
    <div class="divide-y" style="border-color: var(--bidon-border-default);">
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
