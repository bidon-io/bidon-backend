<template>
  <FormField label="Demand Source Type">
    <Dropdown
      v-model="value"
      :options="options"
      option-label="label"
      option-value="value"
      class="w-full md:w-14rem"
      placeholder="Select Demand source type"
    />
  </FormField>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  modelValue: {
    type: [String, null],
    default: null,
  },
});
const emit = defineEmits(["update:modelValue"]);

const value = computed({
  get() {
    return props.modelValue;
  },
  set(value) {
    emit("update:modelValue", value);
  },
});
const options = ref([
  { label: "BidmachineAccount", value: "DemandSourceAccount::BidMachine" },
  { label: "AdmobAccount:", valuse: "DemandSourceAccount::Admob" },
  { label: "ApplovinAccount", valuse: "DemandSourceAccount::Applovin" },
  { label: "DTExchangeAccount", valuse: "DemandSourceAccount::DtExchange" },
  { label: "UnityAdsAccount", valuse: "DemandSourceAccount::UnityAds" },
]);
</script>
