<template>
  <form class="flex-1 p-6 mx-auto w-full" @submit.prevent="handleSubmit">
    <Card>
      <template #title>Auction Config {{ localResource.id }}</template>
      <template #content>
        <div class="divide-y">
          <slot></slot>
        </div>
      </template>
    </Card>
  </form>
</template>

<script setup>
import { ref } from "vue";

const props = defineProps({
  resource: {
    type: Object,
    required: true,
  },
  handleSubmit: {
    type: Function,
    required: true,
  },
});
const localResource = ref(props.resource);
</script>
