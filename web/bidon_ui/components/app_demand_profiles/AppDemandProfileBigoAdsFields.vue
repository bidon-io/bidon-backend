<template>
  <FormField label="App Id" :error="errorMessage" required>
    <InputText v-model="value" type="text" placeholder="App ID" />
  </FormField>
</template>

<script setup>
import { useField } from "vee-validate";

const { value, errorMessage } = useField("data.appId");
</script>
