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
      {{ isEditMode ? "Edit" : "New" }} Demand Profile
      <span v-if="activeNetworkLabel"> — {{ activeNetworkLabel }}</span>
    </p>

    <div
      class="rounded-lg overflow-hidden divide-y mb-3"
      style="
        border: 1px solid var(--bidon-border-default);
        background-color: var(--bidon-bg-card);
      "
    >
      <!-- Network selector (only when networkKey not pre-set and not editing) -->
      <FormField
        v-if="!props.networkKey && !isEditMode"
        label="Network"
        required
      >
        <Dropdown
          v-model="selectedNetworkKey"
          :options="networkOptions"
          option-label="label"
          option-value="key"
          placeholder="Select network…"
          class="w-full"
          filter
        />
      </FormField>

      <template v-if="activeNetworkKey">
        <div
          v-if="accountsLoading"
          class="px-4 py-3 text-xs"
          style="color: var(--bidon-muted)"
        >
          Loading accounts…
        </div>
        <div
          v-else-if="!accounts.length"
          class="px-4 py-3 text-xs text-red-500"
        >
          No demand source accounts found for {{ activeNetworkLabel }}.
          <NuxtLink to="/demand_sources" class="underline hover:text-red-700">
            Set one up here.
          </NuxtLink>
        </div>

        <template v-else>
          <FormField label="Account" :required="accounts.length > 1">
            <template v-if="accounts.length > 1">
              <Dropdown
                v-model="accountId"
                :options="accounts"
                option-label="label"
                option-value="id"
                placeholder="Select account…"
                class="w-full"
              />
              <small
                v-if="errors.accountId"
                class="text-red-500 text-xs mt-1 block"
              >
                {{ errors.accountId }}
              </small>
            </template>
            <span
              v-else
              class="text-sm"
              style="color: var(--bidon-text-primary)"
            >
              {{ accounts[0].label }}
            </span>
          </FormField>

          <AppDemandProfileDataFormFields
            v-model:schema="dataSchema"
            v-model:data="data"
            :account-type="activeAccountType"
          />

          <FormField label="Enabled">
            <Checkbox v-model="enabled" :binary="true" />
          </FormField>
        </template>
      </template>
    </div>

    <p v-if="submitError" class="text-xs text-red-500 mb-2">
      {{ submitError }}
    </p>

    <div class="flex justify-end gap-2">
      <button type="button" class="btn-cancel btn-sm" @click="$emit('cancel')">
        <i class="pi pi-times" /> Cancel
      </button>
      <Button
        v-if="activeNetworkKey && accounts.length"
        type="button"
        :label="isEditMode ? 'Update' : 'Save'"
        icon="pi pi-check"
        size="small"
        :loading="saving"
        @click="save"
      />
    </div>
  </div>
</template>

<script setup>
import * as yup from "yup";
import {
  NETWORK_DEFS,
  NETWORK_ACCOUNT_TYPE_BY_KEY,
  NETWORK_LABEL_BY_KEY,
} from "@/constants/Networks.js";

const props = defineProps({
  appId: { type: [Number, String], required: true },
  networkKey: { type: String, default: null },
  initialProfile: { type: Object, default: null },
});

const emit = defineEmits(["created", "updated", "cancel"]);

const isEditMode = computed(() => !!props.initialProfile);

// When networkKey is pre-set (edit or contextual create), use it directly.
// Otherwise let the user pick via dropdown.
const selectedNetworkKey = ref(props.networkKey ?? null);
const activeNetworkKey = computed(
  () => props.networkKey ?? selectedNetworkKey.value,
);

const activeNetworkLabel = computed(() =>
  activeNetworkKey.value
    ? (NETWORK_LABEL_BY_KEY[activeNetworkKey.value] ?? activeNetworkKey.value)
    : null,
);
const activeAccountType = computed(() =>
  activeNetworkKey.value
    ? (NETWORK_ACCOUNT_TYPE_BY_KEY[activeNetworkKey.value] ??
      activeNetworkKey.value)
    : null,
);

const networkOptions = NETWORK_DEFS.map((n) => ({
  key: n.key,
  label: n.label,
}));

const dataSchema = ref(yup.object());

const { errors, useFieldModel, handleSubmit } = useForm({
  validationSchema: computed(() =>
    yup.object({
      accountId: yup.number().required().label("Account"),
      data: dataSchema.value,
      enabled: yup.boolean(),
    }),
  ),
  initialValues: {
    accountId: props.initialProfile?.accountId ?? null,
    data: props.initialProfile?.data ?? {},
    enabled: props.initialProfile?.enabled ?? true,
  },
});

const accountId = useFieldModel("accountId");
const data = useFieldModel("data");
const enabled = useFieldModel("enabled");

const accountsLoading = ref(false);
const accounts = ref([]);
const allAccounts = ref([]);

onMounted(async () => {
  accountsLoading.value = true;
  try {
    const result = await $apiFetch("/demand_source_accounts", {
      params: { limit: 1000 },
    });
    allAccounts.value = (
      Array.isArray(result) ? result : (result.items ?? [])
    ).map((a) => ({
      id: a.id,
      type: a.type,
      label: a.label ? `${a.label} (#${a.id})` : `#${a.id}`,
      demandSourceId: a.demandSourceId,
    }));
  } finally {
    accountsLoading.value = false;
  }
  loadAccountsForNetwork();
});

function loadAccountsForNetwork() {
  if (!activeNetworkKey.value) {
    accounts.value = [];
    return;
  }
  if (isEditMode.value && props.initialProfile?.demandSourceId) {
    // Edit mode: match by demandSourceId — more reliable than type string comparison
    accounts.value = allAccounts.value.filter(
      (a) => a.demandSourceId === props.initialProfile.demandSourceId,
    );
  } else if (activeAccountType.value) {
    // Create mode: case-insensitive type match for robustness
    const target = activeAccountType.value.toLowerCase();
    accounts.value = allAccounts.value.filter(
      (a) => a.type?.toLowerCase() === target,
    );
  } else {
    accounts.value = [];
    return;
  }
  if (accounts.value.length === 1) {
    accountId.value = accounts.value[0].id;
  } else if (isEditMode.value && props.initialProfile?.accountId) {
    accountId.value = props.initialProfile.accountId;
  } else {
    accountId.value = null;
  }
}

watch(activeNetworkKey, () => {
  loadAccountsForNetwork();
});

const demandSourceId = computed(
  () =>
    allAccounts.value.find((a) => a.id === accountId.value)?.demandSourceId ??
    null,
);

const saving = ref(false);
const submitError = ref("");

const save = handleSubmit(async (values) => {
  saving.value = true;
  submitError.value = "";
  try {
    if (isEditMode.value) {
      const result = await $apiFetch(
        `/app_demand_profiles/${props.initialProfile.id}`,
        {
          method: "PATCH",
          body: {
            accountId: values.accountId,
            demandSourceId: demandSourceId.value,
            accountType: activeAccountType.value,
            data: values.data ?? {},
            enabled: values.enabled,
          },
        },
      );
      emit("updated", result);
    } else {
      const result = await $apiFetch("/app_demand_profiles", {
        method: "POST",
        body: {
          appId: Number(props.appId),
          accountId: values.accountId,
          demandSourceId: demandSourceId.value,
          accountType: activeAccountType.value,
          data: values.data ?? {},
          enabled: values.enabled,
        },
      });
      emit("created", result);
    }
  } catch (err) {
    const msg = err.data?.error?.message;
    submitError.value =
      msg ||
      (isEditMode.value
        ? "Failed to update demand profile"
        : "Failed to create demand profile");
  } finally {
    saving.value = false;
  }
});
</script>
