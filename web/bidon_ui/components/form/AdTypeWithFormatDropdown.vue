<template>
  <FormField label="Ad Type">
    <Dropdown
      v-model="selectedOption"
      option-label="label"
      :options="options"
      class="w-full md:w-14rem"
      placeholder="Select Ad Format"
    />
  </FormField>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  modelValue: {
    type: [Object, null],
    default: null,
  },
});
const emit = defineEmits(["update:modelValue"]);

const options = [
  { label: "Adaptive Banner", value: { adType: "banner", format: "ADAPTIVE" } },
  { label: "Banner", value: { adType: "banner", format: "BANNER" } },
  { label: "Leaderboard", value: { adType: "banner", format: "LEADERBOARD" } },
  { label: "MREC", value: { adType: "banner", format: "MREC" } },
  { label: "Interstitial", value: { adType: "interstitial", format: "" } },
  { label: "Rewarded", value: { adType: "rewarded", format: "" } },
];

// Find the matching option by field so PrimeVue gets a stable object reference
// rather than relying on deep equality between reactive proxies.
const selectedOption = computed({
  get() {
    if (!props.modelValue?.adType) return null;
    return (
      options.find(
        (o) =>
          o.value.adType === props.modelValue.adType &&
          (o.value.format ?? "") === (props.modelValue.format ?? ""),
      ) ?? null
    );
  },
  set(option) {
    emit("update:modelValue", option?.value ?? null);
  },
});
</script>
