<template>
  <NuxtLink v-if="isValid" :to="`${link.basePath}/${id}`" class="table-resource-link">
    {{ linkText }}
  </NuxtLink>
</template>

<script setup lang="ts">
const props = defineProps<{
  link: ResourceLink;
  data: object;
}>();
const { isValid, linkText, id } = props.link.extractLinkData(props.data);
</script>
