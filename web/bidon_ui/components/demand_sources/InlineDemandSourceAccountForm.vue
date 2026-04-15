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
      {{ isEditMode ? "Edit" : "New" }} Demand Source Account
    </p>

    <div
      class="rounded-lg overflow-hidden divide-y mb-3"
      style="
        border: 1px solid var(--bidon-border-default);
        background-color: var(--bidon-bg-card);
      "
    >
      <OwnerWithSharedDropdown
        v-if="currentUser.isAdmin"
        v-model="userId"
        :error="errors.userId"
      />

      <FormField label="Label" :error="errors.label" required>
        <InputText v-model="label" type="text" placeholder="Label" />
      </FormField>

      <DemandSourceAccountExtraFormFields
        v-model:schema="extraSchema"
        :api-key="demandSourceApiKey"
      />
    </div>

    <p v-if="submitError" class="text-xs text-red-500 mb-2">
      {{ submitError }}
    </p>

    <div class="flex justify-end gap-2">
      <button type="button" class="btn-cancel btn-sm" @click="$emit('cancel')">
        <i class="pi pi-times" /> Cancel
      </button>
      <Button
        type="button"
        :label="isEditMode ? 'Update' : 'Save'"
        icon="pi pi-check"
        size="small"
        :loading="saving"
        @click="onSubmit"
      />
    </div>
  </div>
</template>

<script setup>
import * as yup from "yup";
import { useToast } from "primevue/usetoast";
import { NETWORK_ACCOUNT_TYPE_BY_KEY } from "@/constants/Networks.js";
import OwnerWithSharedDropdown from "~/components/form/OwnerWithSharedDropdown.vue";

const props = defineProps({
  demandSourceId: { type: [Number, String], required: true },
  demandSourceApiKey: { type: String, required: true },
  initialAccount: { type: Object, default: null },
});

const emit = defineEmits(["created", "updated", "cancel"]);

const { currentUser } = useAuthStore();
const toast = useToast();
const isEditMode = computed(() => !!props.initialAccount);
const saving = ref(false);
const submitError = ref("");

const extraSchema = ref(yup.object());

const { errors, useFieldModel, handleSubmit } = useForm({
  validationSchema: computed(() => {
    const validationFields = {
      label: yup.string().required().label("Label"),
      extra: extraSchema.value,
    };
    if (currentUser.isAdmin) {
      validationFields.userId = yup.number().nullable(true).label("User Id");
    }
    return yup.object(validationFields);
  }),
  initialValues: {
    userId: props.initialAccount?.userId ?? null,
    label: props.initialAccount?.label ?? "",
    extra: props.initialAccount?.extra ?? {},
  },
});

const userId = useFieldModel("userId");
const label = useFieldModel("label");

const accountType = computed(() => {
  return (
    NETWORK_ACCOUNT_TYPE_BY_KEY[
      String(props.demandSourceApiKey).toLowerCase()
    ] ?? null
  );
});

const onSubmit = handleSubmit(async (values) => {
  if (!accountType.value) {
    submitError.value = `No account type mapping found for network "${props.demandSourceApiKey}".`;
    return;
  }

  saving.value = true;
  submitError.value = "";

  const payload = {
    userId: values.userId ?? undefined,
    label: values.label,
    extra: values.extra,
    demandSourceId: Number(props.demandSourceId),
    type: accountType.value,
  };

  try {
    if (isEditMode.value) {
      const account = await $apiFetch(
        `/demand_source_accounts/${props.initialAccount.id}`,
        {
          method: "PATCH",
          body: payload,
        },
      );
      toast.add({
        severity: "success",
        summary: "Success",
        detail: "Demand source account updated.",
        life: 3000,
      });
      emit("updated", account);
      return;
    }

    const account = await $apiFetch("/demand_source_accounts", {
      method: "POST",
      body: payload,
    });
    toast.add({
      severity: "success",
      summary: "Success",
      detail: "Demand source account created.",
      life: 3000,
    });
    emit("created", account);
  } catch (e) {
    const apiError = e?.data?.error;
    submitError.value = apiError
      ? `Status Code ${apiError.code} ${apiError.message}`
      : "Could not save demand source account.";
  } finally {
    saving.value = false;
  }
});
</script>
