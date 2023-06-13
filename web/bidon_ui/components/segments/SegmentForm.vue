<template>
  <form @submit.prevent="emit('submit', resource)">
    <FormCard title="Auction Configuration">
      <FormField lable="Name">
        <InputText v-model="resource.name" type="text" placeholder="Name" />
      </FormField>
      <FormField lable="Description">
        <Textarea v-model="resource.description" rows="5" cols="50" />
      </FormField>
      <FormField lable="Filters">
        <InputText v-model="filters" type="text" placeholder="Filters" style="min-width: 400px" />
      </FormField>
      <FormField lable="Enabled">
        <Checkbox v-model="resource.enabled" :binary="true" />
      </FormField>
      <AppDropdown v-model="resource.app_id" />
      <FormSubmitButton />
    </FormCard>
  </form>
</template>

<script setup>
import { computed } from "vue";
const props = defineProps({
  value: {
    type: Object,
    required: true,
  },
});
const resource = ref(props.value);
const emit = defineEmits(["submit"]);

const filters = computed({
  get: () => JSON.stringify(resource.value.filters),
  set: (value) => {
    try {
      resource.value.filters = JSON.parse(value);
    } catch {}
  },
});
</script>
