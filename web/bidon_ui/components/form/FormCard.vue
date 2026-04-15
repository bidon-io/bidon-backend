<template>
  <div class="card mb-6">
    <div
      v-if="props.title || $slots.headerActions"
      class="card-header flex items-center justify-between gap-3"
    >
      <span
        v-if="props.title"
        class="font-semibold min-w-0"
        style="color: var(--bidon-text-primary)"
      >
        {{ props.title }}
      </span>
      <div v-if="$slots.headerActions" class="shrink-0 flex items-center gap-2">
        <slot name="headerActions" />
      </div>
    </div>
    <div class="divide-y" style="border-color: var(--bidon-border-default)">
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
