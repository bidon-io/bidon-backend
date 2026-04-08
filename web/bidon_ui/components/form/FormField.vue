<template>
  <div class="flex flex-row py-3.5 px-6">
    <div class="w-1/4 pr-4">
      <span class="text-sm font-medium text-gray-500">{{ props.label }}</span>
      <span v-if="required" class="text-red-500 ml-1">*</span>
    </div>
    <div class="w-3/4">
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

<style>
input,
textarea {
  min-width: 100%;
}
</style>
