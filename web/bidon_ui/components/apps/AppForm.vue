<template>
  <form @submit.prevent="emit('submit', resource)">
    <FormCard title="Appp">
      <PlatformIdDropdown v-model="resource.platformId" required />
      <FormField label="Human Name" required>
        <InputText v-model="resource.humanName" type="text" placeholder="Name" />
      </FormField>
      <FormField label="Package Name" required>
        <InputText v-model="resource.packageName" type="text" placeholder="Name" />
      </FormField>
      <UserDropdown v-model="resource.userId" required />
      <FormField label="App Key" required>
        <InputText v-model="resource.appKey" type="text" placeholder="Name" />
      </FormField>
      <FormField label="Settings">
        <TextareaJSON v-model="resource.settings" rows="5" cols="50" />
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
const resource = ref(props.value);
</script>
