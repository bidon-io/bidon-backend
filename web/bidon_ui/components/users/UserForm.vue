<template>
  <transition-group name="p-message" tag="div">
    <Message v-for="(msg, index) in errorMsgs" :key="index" severity="error">{{
      msg
    }}</Message>
  </transition-group>
  <form @submit="onSubmit">
    <FormCard title="User">
      <FormField label="Email" :error="errors.email" required>
        <InputText v-model="email" type="string" placeholder="Email" />
      </FormField>
      <FormField label="Is Admin">
        <Checkbox v-model="isAdmin" :binary="true" />
      </FormField>
      <FormField label="Password" :error="errors.password" required>
        <InputText v-model="password" type="password" placeholder="Password" />
      </FormField>
      <template #footer>
        <FormSubmitButton />
      </template>
    </FormCard>
  </form>
</template>

<script setup>
import * as yup from "yup";

const props = defineProps({
  value: {
    type: Object,
    required: true,
  },
  submitError: {
    type: [Error, null],
    default: null,
  },
});
const emit = defineEmits(["submit"]);
const resource = ref(props.value);

const errorMsgs = ref([]);
watch(
  () => props.submitError,
  () => {
    if (!props.submitError) return;

    const error = props.submitError.response?.data?.error;
    const errorMessage = error
      ? `Status Code ${error.code} ${error.message}`
      : `Status Code ${props.submitError.response?.status ?? ""} ${props.submitError.response?.statusText ?? ""}`;
    errorMsgs.value.push(errorMessage);
  },
);

const { errors, useFieldModel, handleSubmit } = useForm({
  validationSchema: yup.object({
    email: yup.string().required().label("Email"),
    isAdmin: yup.boolean(),
    password: yup.string().required().label("Password"),
  }),
  initialValues: {
    email: resource.value.email || "",
    password: "",
    isAdmin: resource.value.isAdmin || false,
  },
});

const email = useFieldModel("email");
const isAdmin = useFieldModel("isAdmin");
const password = useFieldModel("password");

const onSubmit = handleSubmit((values) => emit("submit", values));
</script>
