<template>
  <div
    class="inline-auction-config-form border border-gray-200 rounded-lg bg-white overflow-hidden mb-4 [&_.card]:mb-0"
  >
    <AuctionConfigurationsForm
      :key="formKey"
      :value="initialValue"
      :compact="isEditMode"
      @submit="handleSubmit"
    />

    <p v-if="submitError" class="text-xs text-red-500 px-6 pb-3">
      {{ submitError }}
    </p>
  </div>
</template>

<script setup>
import { useToast } from "primevue/usetoast";

const props = defineProps({
  appId: { type: [Number, String], required: true },
  cloneFrom: { type: Object, default: null },
  editConfig: { type: Object, default: null },
});

const emit = defineEmits(["created", "updated", "cancel"]);

const isEditMode = computed(() => !!props.editConfig);

const formKey = computed(
  () => props.editConfig?.id ?? props.cloneFrom?.id ?? "new",
);

const initialValue = computed(() => {
  if (isEditMode.value) {
    const {
      id,
      name,
      appId,
      adType,
      pricefloor,
      segmentId,
      isDefault,
      externalWinNotifications,
      timeout,
      demands,
      bidding,
      adUnitIds,
      settings,
    } = props.editConfig;
    return {
      id,
      name,
      appId: Number(appId),
      adType,
      pricefloor,
      segmentId,
      isDefault,
      externalWinNotifications,
      timeout,
      demands,
      bidding,
      adUnitIds,
      settings,
    };
  }

  const base = { appId: Number(props.appId) };
  if (!props.cloneFrom) return base;

  const {
    adType,
    pricefloor,
    segmentId,
    externalWinNotifications,
    timeout,
    demands,
    bidding,
    adUnitIds,
    settings,
  } = props.cloneFrom;
  return {
    ...base,
    adType,
    pricefloor,
    segmentId,
    externalWinNotifications,
    timeout,
    demands,
    bidding,
    adUnitIds,
    settings,
    name: "",
    isDefault: false,
  };
});

const toast = useToast();
const submitError = ref("");

const handleSubmit = async (values) => {
  submitError.value = "";
  try {
    if (isEditMode.value) {
      const data = await $apiFetch(
        `/v2/auction_configurations/${props.editConfig.id}`,
        { method: "PATCH", body: values },
      );
      toast.add({
        severity: "success",
        summary: "Success",
        detail: "Auction configuration updated!",
        life: 3000,
      });
      emit("updated", data);
    } else {
      const data = await $apiFetch("/v2/auction_configurations", {
        method: "POST",
        body: values,
      });
      toast.add({
        severity: "success",
        summary: "Success",
        detail: "Auction configuration created!",
        life: 3000,
      });
      emit("created", data);
    }
  } catch (err) {
    const msg = err.data?.error?.message;
    submitError.value = isEditMode.value
      ? msg || "Failed to update auction configuration"
      : msg || "Failed to create auction configuration";
  }
};
</script>
