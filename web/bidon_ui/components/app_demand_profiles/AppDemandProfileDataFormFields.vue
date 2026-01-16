<template>
  <template v-if="accountType == 'DemandSourceAccount::Admob'">
    <VeeFormFieldWrapper field="data.appId" label="App Id" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::Amazon'">
    <VeeFormFieldWrapper field="data.appKey" label="App Key" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::Applovin'">
    <FormField label="Ad Unit IDs">
      <Textarea
        v-model="adUnitIdsText"
        rows="3"
        placeholder="abc123def456, xyz789ghi012 (comma-separated)"
      />
    </FormField>
    <VeeFormFieldWrapper
      field="data.mediator"
      label="Mediator"
      placeholder="Bidon (default)"
    />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::BigoAds'">
    <VeeFormFieldWrapper field="data.appId" label="App Id" required />
    <VeeFormFieldWrapper field="data.appChannel" label="App Channel" />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::Chartboost'">
    <VeeFormFieldWrapper field="data.appId" label="App Id" required />
    <VeeFormFieldWrapper
      field="data.appSignature"
      label="App Signature"
      required
    />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::GAM'">
    <VeeFormFieldWrapper field="data.appId" label="App Id" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::DtExchange'">
    <VeeFormFieldWrapper field="data.appId" label="App Id" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::Vungle'">
    <VeeFormFieldWrapper field="data.appId" label="App Id" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::VKAds'">
    <VeeFormFieldWrapper field="data.appId" label="App Id" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::Meta'">
    <VeeFormFieldWrapper field="data.appId" label="App Id" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::Mintegral'">
    <VeeFormFieldWrapper field="data.appId" label="App Id" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::Moloco'">
    <VeeFormFieldWrapper field="data.appKey" label="App Key" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::StartIO'">
    <VeeFormFieldWrapper field="data.appId" label="App Id" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::TaurusX'">
    <VeeFormFieldWrapper field="data.appId" label="App Id" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::UnityAds'">
    <VeeFormFieldWrapper field="data.gameId" label="Game Id" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::Inmobi'">
    <VeeFormFieldWrapper field="data.appKey" label="App Key" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::IronSource'">
    <VeeFormFieldWrapper field="data.appKey" label="App Key" required />
  </template>
  <template v-if="accountType == 'DemandSourceAccount::Yandex'">
    <VeeFormFieldWrapper field="data.metricaId" label="Metrica ID" required />
  </template>
</template>

<script setup>
import * as yup from "yup";

const props = defineProps({
  schema: {
    type: Object,
    required: true,
  },
  accountType: {
    type: String,
    required: true,
  },
  data: {
    type: Object,
    required: true,
  },
});
const emit = defineEmits(["update:schema", "update:data"]);

// Ad Unit IDs field for Applovin - convert between array and comma-separated string
const adUnitIdsText = computed({
  get: () => {
    const ids = props.data?.adUnitIds;
    return Array.isArray(ids) ? ids.join(", ") : "";
  },
  set: (v) => {
    emit("update:data", {
      ...props.data,
      adUnitIds: v
        ? v
            .split(",")
            .map((s) => s.trim())
            .filter((s) => s)
        : [],
    });
  },
});

const dataSchemas = {
  "DemandSourceAccount::Admob": yup.object({
    appId: yup.string().required().label("App Id"),
  }),
  "DemandSourceAccount::Amazon": yup.object({
    appKey: yup.string().required().label("App Key"),
  }),
  "DemandSourceAccount::Applovin": yup.object({
    adUnitIds: yup.array().of(yup.string()).label("Ad Unit IDs"),
    mediator: yup.string().label("Mediator"),
  }),
  "DemandSourceAccount::BigoAds": yup.object({
    appId: yup.number().required().label("App Id"),
    apphannel: yup.string().label("App Channel"),
  }),
  "DemandSourceAccount::Chartboost": yup.object({
    appId: yup.string().required().label("App Id"),
    appSignature: yup.string().required().label("App Signature"),
  }),
  "DemandSourceAccount::GAM": yup.object({
    appId: yup.string().required().label("App Id"),
  }),
  "DemandSourceAccount::DtExchange": yup.object({
    appId: yup.number().required().label("App Id"),
  }),
  "DemandSourceAccount::Vungle": yup.object({
    appId: yup.string().required().label("App Id"),
  }),
  "DemandSourceAccount::VKAds": yup.object({
    appId: yup.string().required().label("App Id"),
  }),
  "DemandSourceAccount::Meta": yup.object({
    appId: yup.number().required().label("App Id"),
    appSecret: yup.string().required().label("App Secret"),
  }),
  "DemandSourceAccount::Mintegral": yup.object({
    appId: yup.number().required().label("App Id"),
  }),
  "DemandSourceAccount::Moloco": yup.object({
    appKey: yup.string().required().label("App Key"),
  }),
  "DemandSourceAccount::StartIO": yup.object({
    appId: yup.string().required().label("App Id"),
  }),
  "DemandSourceAccount::TaurusX": yup.object({
    appId: yup.string().required().label("App Id"),
  }),
  "DemandSourceAccount::UnityAds": yup.object({
    gameId: yup.number().required().label("Game Id"),
  }),
  "DemandSourceAccount::Inmobi": yup.object({
    appKey: yup.string().required().label("App Key"),
  }),
  "DemandSourceAccount::IronSource": yup.object({
    appKey: yup.string().required().label("App Key"),
  }),
  "DemandSourceAccount::Yandex": yup.object({
    metricaId: yup.string().label("Metrica ID"),
  }),
};

const schema = computed(() => dataSchemas[props.accountType] || yup.object());

watchEffect(() => {
  emit("update:schema", schema.value);
});
</script>
