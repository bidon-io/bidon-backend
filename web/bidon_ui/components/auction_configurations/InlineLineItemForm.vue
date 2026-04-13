<template>
  <div class="border-t border-blue-100 bg-blue-50 px-4 py-3">
    <p class="text-xs font-semibold text-blue-700 uppercase tracking-wide mb-3">
      {{ isEditMode ? "Edit" : "New" }} Line Item — {{ networkKey }}
    </p>

    <div v-if="accountsLoading" class="text-xs text-gray-400 mb-2">
      Loading accounts…
    </div>
    <div v-else-if="!accounts.length" class="text-xs text-red-500 mb-2">
      No demand source accounts found for {{ networkKey }}. Please create one
      first.
    </div>

    <template v-else>
      <div
        class="border border-blue-200 rounded-lg bg-white overflow-hidden divide-y divide-gray-100 mb-3"
      >
        <!-- Account (read-only when single, dropdown when multiple) -->
        <FormField label="Account" :required="accounts.length > 1">
          <template v-if="accounts.length > 1">
            <Dropdown
              v-model="accountId"
              :options="accounts"
              option-label="label"
              option-value="id"
              placeholder="Select account…"
              class="w-full"
            >
              <template #option="{ option }">
                <span>{{ option.label || `#${option.id}` }}</span>
              </template>
            </Dropdown>
            <small
              v-if="errors.accountId"
              class="text-red-500 text-xs mt-1 block"
            >
              {{ errors.accountId }}
            </small>
          </template>
          <span v-else class="text-sm text-gray-700">
            {{ accounts[0].label || `#${accounts[0].id}` }}
          </span>
        </FormField>

        <!-- Label -->
        <FormField label="Label" :error="errors.humanName" required>
          <InputText v-model="humanName" type="text" placeholder="Label" />
        </FormField>

        <!-- Bid Floor (waterfall only) -->
        <FormField
          v-if="!isBidding"
          label="Bid Floor"
          :error="errors.bidFloor"
          required
        >
          <InputNumber
            v-model="bidFloor"
            input-id="inlineBidFloor"
            :min-fraction-digits="2"
            :max-fraction-digits="5"
            placeholder="0.00"
          />
        </FormField>

        <!-- Network-specific extra fields -->
        <LineItemExtraFormFields
          v-model:schema="extraSchema"
          :api-key="networkKey"
          :ad-type="adType"
          :ad-type-with-format="adTypeWithFormat"
        />
      </div>

      <p v-if="submitWarning" class="text-xs text-amber-800 mb-2">
        {{ submitWarning }}
      </p>
      <p v-else-if="submitError" class="text-xs text-red-500 mb-2">
        {{ submitError }}
      </p>

      <div class="flex justify-end gap-2">
        <button
          type="button"
          class="btn-cancel btn-sm"
          @click="$emit('cancel')"
        >
          <i class="pi pi-times" /> Cancel
        </button>
        <Button
          type="button"
          :label="isEditMode ? 'Update' : 'Save'"
          icon="pi pi-check"
          size="small"
          :loading="saving"
          @click="save"
        />
      </div>
    </template>

    <div
      v-if="!accountsLoading && !accounts.length"
      class="flex justify-end mt-2"
    >
      <button type="button" class="btn-cancel btn-sm" @click="$emit('cancel')">
        <i class="pi pi-times" /> Cancel
      </button>
    </div>
  </div>
</template>

<script setup>
import * as yup from "yup";
import { NETWORK_ACCOUNT_TYPE_BY_KEY } from "@/constants/Networks.js";
import { $apiFetch } from "~/utils/$apiFetch";

const props = defineProps({
  appId: { type: [Number, String], required: true },
  adType: { type: String, required: true },
  networkKey: { type: String, required: true },
  isBidding: { type: Boolean, required: true },
  initialItem: { type: Object, default: null },
});

const emit = defineEmits(["created", "updated", "cancel"]);

const isEditMode = computed(() => !!props.initialItem);

const accountType = computed(
  () => NETWORK_ACCOUNT_TYPE_BY_KEY[props.networkKey] ?? props.networkKey,
);

const adTypeWithFormat = computed(() => ({
  adType: props.adType,
  format: null,
}));

const extraSchema = ref(yup.object());

const validationSchema = computed(() =>
  yup.object({
    humanName: yup.string().required().label("Label"),
    accountId: yup.number().required().label("Account"),
    bidFloor: props.isBidding
      ? yup.number().nullable(true).label("Bid Floor")
      : yup.number().required().positive().label("Bid Floor"),
    extra: yup.lazy(() => extraSchema.value),
  }),
);

const { errors, useFieldModel, handleSubmit } = useForm({
  validationSchema,
  initialValues: {
    humanName: props.initialItem?.humanName ?? "",
    accountId: props.initialItem?.accountId ?? null,
    bidFloor: props.initialItem?.bidFloor ?? null,
    extra: props.initialItem?.extra ?? {},
  },
});

const humanName = useFieldModel("humanName");
const accountId = useFieldModel("accountId");
const bidFloor = useFieldModel("bidFloor");

const accountsLoading = ref(true);
const accounts = ref([]);

onMounted(async () => {
  try {
    const data = await $apiFetch("/demand_source_accounts", {
      params: { limit: 1000 },
    });
    const allAccounts = Array.isArray(data) ? data : (data.items ?? []);
    const target = (accountType.value ?? "").toLowerCase();
    accounts.value = allAccounts.filter(
      (a) => (a.type ?? "").toLowerCase() === target,
    );
    if (accounts.value.length === 1) {
      accountId.value = accounts.value[0].id;
    } else if (isEditMode.value && props.initialItem?.accountId) {
      accountId.value = props.initialItem.accountId;
    }
  } finally {
    accountsLoading.value = false;
  }
});

const saving = ref(false);
const submitError = ref("");
const submitWarning = ref("");

const save = handleSubmit(async (values) => {
  saving.value = true;
  submitError.value = "";
  submitWarning.value = "";
  try {
    if (isEditMode.value) {
      const data = await $apiFetch(`/line_items/${props.initialItem.id}`, {
        method: "PATCH",
        body: {
          humanName: values.humanName,
          accountId: values.accountId,
          bidFloor: props.isBidding ? null : values.bidFloor,
          extra: values.extra ?? {},
        },
      });
      emit("updated", data);
    } else {
      // POST returns 201 for a new row and 200 when an identical line item
      // already exists (deduped server-side). Only 201 should add to the list.
      const response = await $apiFetch.raw("/line_items", {
        method: "POST",
        body: {
          humanName: values.humanName,
          appId: Number(props.appId),
          adType: props.adType,
          format: null,
          accountId: values.accountId,
          accountType: accountType.value,
          isBidding: props.isBidding,
          bidFloor: props.isBidding ? null : values.bidFloor,
          extra: values.extra ?? {},
        },
      });
      // Admin API: 201 = new row, 200 = existing row (deduped). See CreateLineItem.
      console.log(response);
      if (response.status === 200) {
        submitWarning.value =
          "A line item with the same app, account, ad type, and settings already exists. Nothing new was added. Edit that line item or change placement/extra so the combination is unique.";
        return;
      }
      emit("created", response._data);
    }
  } catch (err) {
    const msg = err.data?.error?.message;
    submitError.value =
      msg ||
      (isEditMode.value
        ? "Failed to update line item"
        : "Failed to create line item");
  } finally {
    saving.value = false;
  }
});
</script>
