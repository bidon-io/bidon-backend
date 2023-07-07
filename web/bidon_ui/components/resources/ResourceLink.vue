<template>
  <NuxtLink v-if="linkData" :to="linkPath" class="text-blue-500">
    {{ linkText }}
  </NuxtLink>
</template>

<script setup lang="ts">
/* eslint-disable @typescript-eslint/no-explicit-any */
const props = defineProps<{
  link: ResourceLink;
  data: any;
}>();

const linkData = props.link.dataField
  ? props.data[props.link.dataField]
  : props.data;
const linkPath = `${props.link.basePath}/${linkData?.id}`;
const linkText = linkData[props.link.textField];
/* eslint-enable @typescript-eslint/no-explicit-any */
</script>
