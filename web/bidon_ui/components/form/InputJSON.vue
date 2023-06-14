<template>
  <InputText v-model="value" type="text" placeholder="Filters" style="min-width: 400px" />
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  modelValue: {
    type: Object,
    required: true,
  },
});
const emit = defineEmits(["update:modelValue"]);
const value = computed({
  get: () => JSON.stringify(props.modelValue),
  set: (newValue) => {
    try {
      emit("update:modelValue", JSON.parse(newValue));
    } catch {}
  },
});
</script>
