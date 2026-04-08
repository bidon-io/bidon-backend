<template>
  <button type="submit" class="btn-new" :disabled="disabled">
    <i class="pi pi-save" /> Save
  </button>
</template>

<script setup>
defineProps({
  disabled: {
    type: Boolean,
    default: false,
  },
});
</script>
