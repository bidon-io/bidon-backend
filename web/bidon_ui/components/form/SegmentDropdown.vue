<template>
  <FormField lable="Segment">
    <Dropdown
      v-model="value"
      :options="segments"
      option-label="name"
      option-value="id"
      class="w-full md:w-14rem"
      placeholder="Select Segment"
    />
  </FormField>
</template>

<script setup>
import { computed } from "vue";
import axios from "@/services/ApiService";

const props = defineProps({
  modelValue: {
    type: Number,
    required: true,
  },
});
const emit = defineEmits(["update:modelValue"]);

const value = computed({
  get() {
    return props.modelValue;
  },
  set(value) {
    emit("update:modelValue", value);
  },
});

const segments = ref([]);
axios
  .get("/segments")
  .then((response) => {
    segments.value = response.data;
  })
  .catch((error) => {
    console.error(error);
  });
</script>
