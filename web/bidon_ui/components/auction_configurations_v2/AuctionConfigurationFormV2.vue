<template>
  <form @submit="onSubmit">
    <FormCard title="Auction Configuration">
      <FormField label="Name" :error="errors.name" required>
        <InputText v-model="name" type="text" placeholder="Name" />
      </FormField>
      <AppDropdown v-model="appId" :error="errors.appId" required />
      <AdTypeDropdown v-model="adType" :error="errors.adType" required />
      <FormField label="Price floor" :error="errors.pricefloor" required>
        <InputNumber
          v-model="pricefloor"
          input-id="pricefloor"
          :min-fraction-digits="2"
          :max-fraction-digits="5"
          placeholder="Price floor"
        />
      </FormField>
      <SegmentDropdown v-model="segmentId" :error="errors.segmentId" />
      <FormField
        label="External Win Notification"
        :error="errors.externalWinNotifications"
      >
        <Checkbox v-model="externalWinNotifications" :binary="true" />
      </FormField>
      <FormField v-if="showNetworks" label="CPM Networks">
        <NetworkAccordion
          v-model:networks="cpmNetworks"
          :ad-type="adType"
          :app-id="appId"
          :is-bidding="false"
          :ad-unit-ids="adUnitIds"
        />
      </FormField>
      <FormField v-if="showNetworks" label="Bidding Networks">
        <NetworkAccordion
          v-model:networks="biddingNetworks"
          :ad-type="adType"
          :app-id="appId"
          :is-bidding="true"
          :ad-unit-ids="adUnitIds"
        />
      </FormField>
      <FormSubmitButton />
      <div>
        <pre>{{ JSON.stringify(adUnitIds, null, 2) }}</pre>
        <pre>{{ JSON.stringify(demands, null, 2) }}</pre>
        <pre>{{ JSON.stringify(cpmNetworks, null, 2) }}</pre>
      </div>
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
});
const emit = defineEmits(["submit"]);
const resource = ref(props.value);

const { errors, useFieldModel, handleSubmit } = useForm({
  validationSchema: yup.object({
    name: yup.string().required().label("Name"),
    appId: yup.number().required().label("App Id"),
    adType: yup.string().required().label("AdType"),
    pricefloor: yup.number().positive().required().label("Pricefloor"),
    segmentId: yup.number().nullable(true).label("Segment Id"),
    externalWinNotifications: yup.boolean(),
    settings: yup.object(),
  }),
  initialValues: {
    name: resource.value.name || "",
    appId: resource.value.appId || null,
    adType: resource.value.adType || "",
    pricefloor: resource.value.pricefloor || null,
    segmentId: resource.value.segmentId || null,
    externalWinNotifications: resource.value.externalWinNotifications || false,
    settings: resource.value.settings || {},
  },
});

const name = useFieldModel("name");
const appId = useFieldModel("appId");
const adType = useFieldModel("adType");
const pricefloor = useFieldModel("pricefloor");
const segmentId = useFieldModel("segmentId");
const externalWinNotifications = useFieldModel("externalWinNotifications");

// TODO: Remove hardcoded networks, fetch from API
const cpmNetworks = ref(
  [
    { label: "Admob", key: "admob" },
    { label: "Applovin", key: "applovin" },
    { label: "DtExchange", key: "dtexchange" },
    { label: "Google Ad Manager", key: "gam" },
    { label: "UnityAds", key: "unityads" },
  ].map((network) => ({
    ...network,
    enabled: false,
    adUnits: [],
    selectedAdUnitIds: [],
  })),
);

const biddingNetworks = ref(
  [
    { label: "BidMachine", key: "bidmachine" },
    { label: "Bigoads", key: "bigoads" },
    { label: "Inmobi", key: "inmobi" },
    { label: "Meta", key: "meta" },
    { label: "Mintegral", key: "mintegral" },
    { label: "MobileFuse", key: "mobilefuse" },
    { label: "Vungle", key: "vungle" },
  ].map((network) => ({
    ...network,
    enabled: false,
    adUnits: [],
    selectedAdUnitIds: [],
  })),
);

const showNetworks = computed(() => appId.value && adType.value);
const demands = computed(() =>
  cpmNetworks.value.filter((network) => network.enabled).map(({ key }) => key),
);
const bidding = computed(() =>
  biddingNetworks.value
    .filter((network) => network.enabled)
    .map(({ key }) => key),
);
const adUnitIds = computed(() =>
  cpmNetworks.value.map((network) => network.selectedAdUnitIds).flat(),
);

// admob
// amazon
// applovin
// bidmachine
// bigoads
// dtexchange
// gam
// inmobi
// meta
// mintegral
// mobilefuse
// unityads
// vungle

const onSubmit = handleSubmit((values) =>
  emit("submit", {
    ...values,
    adUnitIds: adUnitIds.value,
    demands: demands.value,
    bidding: bidding.value,
  }),
);
</script>
