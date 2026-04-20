<template>
  <div
    class="px-4 py-3"
    style="
      border-top: 1px solid var(--bidon-border-default);
      background-color: var(--bidon-bg-card-header);
    "
  >
    <p
      class="text-xs font-semibold uppercase tracking-wide mb-3"
      style="color: var(--bidon-accent)"
    >
      Edit User
    </p>

    <div
      class="rounded-lg overflow-hidden divide-y mb-3"
      style="
        border: 1px solid var(--bidon-border-default);
        background-color: var(--bidon-bg-card);
      "
    >
      <FormField label="Email" :error="errors.email" required>
        <InputText
          v-model="email"
          type="email"
          placeholder="Email"
          @keyup.enter="onSubmit"
        />
      </FormField>

      <FormField label="Is Admin">
        <Checkbox v-model="isAdmin" :binary="true" />
      </FormField>

      <FormField label="Password" :error="errors.password">
        <InputText
          v-model="password"
          type="password"
          placeholder="Set a new password (optional)"
          @keyup.enter="onSubmit"
        />
      </FormField>
    </div>

    <p v-if="submitError" class="text-xs text-red-500 mb-2">
      {{ submitError }}
    </p>

    <div class="flex justify-end gap-2">
      <button type="button" class="btn-cancel btn-sm" :disabled="saving" @click="$emit('cancel')">
        <i class="pi pi-times" /> Cancel
      </button>
      <Button
        type="button"
        label="Update"
        icon="pi pi-check"
        size="small"
        :loading="saving"
        @click="onSubmit"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import * as yup from "yup";
import { useToast } from "primevue/usetoast";

type UserLike = {
  id: number | string;
  email?: string;
  isAdmin?: boolean;
};

const props = defineProps<{
  user: UserLike;
}>();

const emit = defineEmits<{
  updated: [Record<string, unknown>];
  cancel: [];
}>();

const toast = useToast();
const saving = ref(false);
const submitError = ref("");

const { errors, useFieldModel, handleSubmit } = useForm({
  validationSchema: yup.object({
    email: yup.string().email().required().label("Email"),
    isAdmin: yup.boolean(),
    password: yup
      .string()
      .test(
        "password-min-length",
        "Password must be at least 6 characters",
        (value) => !value || value.length >= 6,
      )
      .max(50, "Password must be at most 50 characters")
      .label("Password"),
  }),
  initialValues: {
    email: String(props.user.email ?? ""),
    isAdmin: props.user.isAdmin === true,
    password: "",
  },
});

const email = useFieldModel("email");
const isAdmin = useFieldModel("isAdmin");
const password = useFieldModel("password");

const onSubmit = handleSubmit(async (values) => {
  saving.value = true;
  submitError.value = "";

  const payload: Record<string, unknown> = {
    email: values.email.trim(),
    isAdmin: values.isAdmin === true,
  };

  if (values.password) {
    payload.password = values.password;
  }

  try {
    const response = await $apiFetch<Record<string, unknown>>(
      `/users/${props.user.id}`,
      { method: "PATCH", body: payload },
    );
    toast.add({
      severity: "success",
      summary: "Success",
      detail: "User updated.",
      life: 3000,
    });
    emit("updated", response ?? payload);
  } catch (error: unknown) {
    const fetchError = error as {
      data?: { error?: { message?: string } };
      status?: number;
      statusText?: string;
    };
    submitError.value =
      fetchError.data?.error?.message ??
      `${fetchError.status ?? ""} ${fetchError.statusText ?? "Unable to update user."}`.trim();
  } finally {
    saving.value = false;
  }
});
</script>
