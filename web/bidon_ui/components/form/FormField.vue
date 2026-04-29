<template>
  <div class="flex flex-col gap-2 py-3.5 px-4 sm:px-6 lg:flex-row lg:gap-0">
    <div class="shrink-0 lg:w-1/4 lg:pr-4">
      <span
        class="text-sm font-medium"
        style="color: var(--bidon-text-secondary)"
        >{{ props.label }}</span
      >
      <span v-if="required" class="text-red-500 ml-1">*</span>
    </div>
    <div class="min-w-0 w-full lg:w-3/4">
      <slot></slot>
      <small v-if="error" class="text-red-500 text-xs mt-1 block">
        {{ error }}
      </small>
    </div>
  </div>
</template>

<script setup>
const props = defineProps({
  error: {
    type: String,
    default: "",
  },
  label: {
    type: String,
    required: true,
  },
  required: {
    type: Boolean,
    default: false,
  },
});
</script>

<style scoped>
:deep(.p-dropdown),
:deep(input),
:deep(textarea) {
  min-width: 100%;
  box-sizing: border-box;
}
</style>
