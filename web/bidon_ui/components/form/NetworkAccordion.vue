<template>
  <div>
    <Accordion :active-index="0" :multiple="true">
      <AccordionTab v-for="network in networks" :key="network.key">
        <ToggleButton
          v-model="network.enabled"
          class="w-6rem mb-2"
          on-label="On"
          off-label="Off"
          on-icon="pi pi-check"
          off-icon="pi pi-times"
        />
        <template #header>
          <span class="flex align-items-center gap-2 w-full">
            {{ network.label }}
            <Badge
              :value="network.selectedAdUnitIds?.length"
              :severity="network.enabled ? 'success' : 'danger'"
              class="ml-auto mr-2"
            />
          </span>
        </template>
        <Fieldset legend="Ad Units">
          <div class="flex flex-col gap-3">
            <div
              v-for="adUnit in network.adUnits"
              :key="adUnit.id"
              class="flex align-items-center"
            >
              <Checkbox
                v-model="network.selectedAdUnitIds"
                :value="adUnit.id"
              />
              <div class="flex gap-2 ml-4 text-sm">
                <span><b>Label:</b> {{ adUnit.label }}</span>
                <span><b>UID:</b> {{ adUnit.uid }}</span>
                <span
                  ><b>Price Floor:</b>
                  {{ `$${adUnit.pricefloor.toFixed(2)}` }}</span
                >
                <span><b>Account:</b> {{ adUnit.account }}</span>
                <span><b>Bidding:</b> {{ adUnit.isBidding }}</span>
              </div>
            </div>
          </div>
        </Fieldset>
      </AccordionTab>
    </Accordion>
  </div>
</template>

<script lang="ts" setup>
import axios from "@/services/ApiService";

type Network = {
  label: string;
  key: string;
  enabled: boolean;
  adUnits: AdUnit[];
  selectedAdUnitIds: number[];
};

type AdUnit = {
  id: number;
  uid: string;
  label: string;
  networkKey: string;
  isBidding: boolean;
  pricefloor: number;
  account: string;
};

const props = defineProps({
  appId: {
    type: Number as PropType<number>,
    default: null,
  },
  adType: {
    type: String as PropType<string>,
    default: "",
  },
  isBidding: {
    type: Boolean as PropType<boolean>,
    default: false,
  },
  networks: {
    type: Array as PropType<Network[]>,
    default: () => [],
  },
  adUnitIds: {
    type: Array as PropType<number[]>,
    default: () => [],
  },
});

const emit = defineEmits(["update:networks"]);

const fetchAdUnits = async () => {
  if (!props.appId || !props.adType) return [];
  try {
    const response = await axios.get(
      `/line_items?appId=${props.appId}&adType=${props.adType}`,
    );
    // TODO: Filters on API doesn't work, so filtering here
    const result = response.data.filter(
      (
        adUnit: any, // eslint-disable-line @typescript-eslint/no-explicit-any
      ) =>
        adUnit.adType === props.adType &&
        adUnit.appId === props.appId &&
        adUnit.isBidding === props.isBidding,
    );
    return result.map(
      (
        adUnit: any, // eslint-disable-line @typescript-eslint/no-explicit-any
      ) => ({
        id: adUnit.id,
        label: adUnit.humanName,
        networkKey: adUnit.accountType.split("::")[1].toLowerCase(),
        uid: adUnit.publicUid,
        pricefloor: parseFloat(adUnit.bidFloor),
        account: `${adUnit.accountType.split("::")[1].toLowerCase()} (${
          adUnit.accountId
        })`,
        isBidding: adUnit.isBidding,
      }),
    ) as AdUnit[];
  } catch (error) {
    console.error("Failed to fetch ad units:", error);
    return [];
  }
};

const networks = computed({
  get: () => props.networks,
  set: (value: Network[]) => emit("update:networks", value),
});

watch(
  () => [props.appId, props.adType],
  async () => {
    const adUnits = await fetchAdUnits();
    const updatedNetworks = props.networks.map((network) => {
      const networkAdUnits = adUnits.filter(
        (adUnit) => adUnit.networkKey === network.key,
      );
      return {
        ...network,
        adUnits: networkAdUnits,
        selectedAdUnitIds: props.adUnitIds.filter((id) =>
          networkAdUnits.some((unit) => unit.id === id),
        ),
      };
    });
    emit("update:networks", updatedNetworks);
  },
  { immediate: true },
);
</script>
