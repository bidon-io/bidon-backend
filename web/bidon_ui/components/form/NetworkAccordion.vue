<template>
  <div v-if="loading" class="flex items-center p-4">
    <i class="pi pi-spin pi-spinner mr-2"></i>
    <span>Loading networks...</span>
  </div>
  <Accordion v-else :active-index="0" :multiple="true">
    <AccordionTab v-for="network in networks" :key="network.key">
      <ToggleButton
        :model-value="enabledNetworkKeys.includes(network.key)"
        class="w-6rem mb-2"
        on-label="On"
        off-label="Off"
        on-icon="pi pi-check"
        off-icon="pi pi-times"
        @update:model-value="(val) => onToggleNetwork(network.key, val)"
      />
      <template #header>
        <span class="flex align-items-center gap-2 w-full">
          {{ network.label }}
          <Badge
            :value="getNetworkSelectedCount(network)"
            :severity="
              enabledNetworkKeys.includes(network.key) ? 'success' : 'danger'
            "
            class="ml-auto mr-2"
          />
        </span>
      </template>
      <Fieldset legend="Ad Units" class="p-fieldset">
        <div class="flex flex-col gap-2">
          <div
            v-for="(adUnit, index) in network.adUnits"
            :key="adUnit.id"
            class="flex items-start p-2"
            :class="{
              'border-b border-gray-200': index !== network.adUnits.length - 1,
            }"
          >
            <Checkbox
              :model-value="modelValue"
              :value="adUnit.id"
              class="mr-3 mt-1"
              @update:model-value="onAdUnitsChange"
            />
            <div class="flex-grow">
              <NuxtLink
                :to="`/line_items/${adUnit.id}`"
                class="text-blue-600 hover:text-blue-800 hover:underline font-medium"
              >
                {{ adUnit.label }}
              </NuxtLink>
              <div v-if="!network.isBidding" class="text-sm text-gray-600 mt-1">
                <span class="font-medium"
                  >Floor: ${{ adUnit.pricefloor.toFixed(2) }}</span
                >
              </div>
              <div
                v-if="Object.keys(adUnit.extra).length > 0"
                class="text-sm text-gray-600 mt-1"
              >
                <div
                  v-for="field in getExtraFields(adUnit.extra)"
                  :key="field.key"
                  class="mt-0.5"
                >
                  <span class="font-medium">{{ field.label }}:</span>
                  {{ field.value }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </Fieldset>
    </AccordionTab>
  </Accordion>
</template>

<script lang="ts" setup>
import { formatLabel } from "@/utils/jsonToFields";

type AdUnit = {
  id: number;
  uid: string;
  label: string;
  networkKey: string;
  isBidding: boolean;
  pricefloor: number;
  account: string;
  extra: Record<string, string | number | boolean>;
};

type Network = {
  label: string;
  key: string;
  isBidding: boolean;
  adUnits: AdUnit[];
};

const props = defineProps({
  loading: {
    type: Boolean,
    default: false,
  },
  networks: {
    type: Array as PropType<Network[]>,
    default: () => [],
  },
  enabledNetworkKeys: {
    type: Array as PropType<string[]>,
    default: () => [],
  },
  // Selected Ad Unit IDs
  modelValue: {
    type: Array as PropType<number[]>,
    default: () => [],
  },
});

const emit = defineEmits([
  "update:modelValue",
  "update:enabledNetworkKeys",
  "network-enabled",
  "network-disabled",
]);

const getExtraFields = (extra: Record<string, string | number | boolean>) => {
  if (!extra) {
    return [];
  }

  return Object.keys(extra).map((key) => ({
    key,
    label: formatLabel(key),
    value: extra[key],
  }));
};

const getNetworkSelectedCount = (network: Network) => {
  return network.adUnits.filter((unit) => props.modelValue.includes(unit.id))
    .length;
};

const onToggleNetwork = (key: string, enabled: boolean) => {
  const newKeys = enabled
    ? [...props.enabledNetworkKeys, key]
    : props.enabledNetworkKeys.filter((k) => k !== key);

  emit("update:enabledNetworkKeys", newKeys);
  emit(enabled ? "network-enabled" : "network-disabled", key);
};

const onAdUnitsChange = (newIds: number[]) => {
  emit("update:modelValue", newIds);
};
</script>

<style scoped>
.p-datatable-header,
.p-datatable-row {
  display: grid;
}
.p-datatable-header {
  font-weight: bold;
  background-color: var(--surface-b);
  border-bottom: 1px solid var(--surface-d);
}
.p-datatable-row {
  border-bottom: 1px solid var(--surface-d);
}
</style>
