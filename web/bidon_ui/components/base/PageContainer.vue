<template>
  <div class="flex-1 mx-auto w-full">
    <slot></slot>
  </div>
</template>

<script setup>
const showToasts = useShowToasts();
onMounted(showToasts);
</script>
