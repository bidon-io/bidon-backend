<template>
  <NuxtLink :to="`${path}/${id}/edit`">
    <Button label="Edit" icon="pi pi-pencil" />
  </NuxtLink>
</template>

<script setup lang="ts">
const props = defineProps<{
  id: string;
  path: string;
}>();
</script>
