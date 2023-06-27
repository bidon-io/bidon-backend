<template>
  <form @submit.prevent="emit('submit', resource)">
    <FormCard title="Demand source account">
      <UserDropdown v-model="resource.user_id" />
      <DemandSourceTypeDropdown v-model="resource.demand_source_type" />
      <DemandSourceDropdown v-model="resource.demand_source_id" />
      <FormField label="Human Name">
        <InputText v-model="resource.human_name" type="text" placeholder="Name" />
      </FormField>
      <FormField label="Bidding">
        <Checkbox v-model="resource.is_bidding" :binary="true" />
      </FormField>
      <FormField label="Extra">
        <TextareaJSON v-model="resource.extra" rows="5" cols="50" />
      </FormField>
      <FormSubmitButton />
    </FormCard>
  </form>
</template>

<script setup>
const props = defineProps({
  value: {
    type: Object,
    required: true,
  },
});
const emit = defineEmits(["submit"]);
const resource = ref({ ...props.value, extra: {} });
</script>
