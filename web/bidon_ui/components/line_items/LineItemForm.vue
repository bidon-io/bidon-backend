<template>
  <form @submit="onSubmit">
    <FormCard title="Line Item">
      <AppDropdown v-model="appId" :error="errors.appId" required />
      <AdTypeDropdown v-model="adType" :error="errors.adType" required />
      <AdFormatDropdown v-model="format" :error="errors.format" required />
      <DemandSourceTypeDropdown
        v-model="accountType"
        :error="errors.accountType"
        required
      />
      <DemandSourceAccountDropdown
        v-model="accountId"
        :error="errors.accountId"
        :accounts="demandSourceAccounts"
        required
      />
      <FormField label="Label" :error="errors.humanName" required>
        <InputText v-model="humanName" type="text" placeholder="Label" />
      </FormField>
      <FormField label="Bid Floor" :error="errors.bidFloor" required>
        <InputNumber
          v-model="bidFloor"
          input-id="bidFloor"
          :min-fraction-digits="2"
          :max-fraction-digits="5"
          placeholder="Bid Floor"
        />
      </FormField>
      <FormField label="Code" :error="errors.code" required>
        <InputText v-model="code" type="text" placeholder="Code" />
      </FormField>
      <LineItemExtraFormFields v-model:schema="extraSchema" :api-key="apiKey" />
      <FormSubmitButton />
    </FormCard>
  </form>
</template>

<script setup>
import axios from "@/services/ApiService";
import * as yup from "yup";

const props = defineProps({
  value: {
    type: Object,
    required: true,
  },
});
const emit = defineEmits(["submit"]);
const resource = ref(props.value);

const extraSchema = ref(yup.object());
const { errors, useFieldModel, handleSubmit } = useForm({
  validationSchema: computed(() =>
    yup.object({
      humanName: yup.string().required().label("Label"),
      appId: yup.number().required().label("App Id"),
      bidFloor: yup.number().positive().required().label("Bid Floor"),
      adType: yup.string().required().label("AdType"),
      format: yup
        .string()
        .nullable(true)
        .when("adType", {
          is: "banner",
          then: (schema) =>
            schema.required("Format is required for Banner Ad Type"),
        }),
      accountId: yup.number().required().label("Account Id"),
      accountType: yup.string().required().label("Demand Source Type"),
      code: yup.string().required().label("Code"),
      extra: extraSchema.value,
    })
  ),
  initialValues: {
    humanName: resource.value.humanName || "",
    appId: resource.value.appId || null,
    bidFloor: resource.value.bidFloor || null,
    adType: resource.value.adType || "",
    adFormat: resource.value.format || "",
    accountId: resource.value.accountId || null,
    accountType: resource.value.accountType || "",
    code: resource.value.code || "",
    extra: resource.value.extra || {},
  },
});

const humanName = useFieldModel("humanName");
const appId = useFieldModel("appId");
const bidFloor = useFieldModel("bidFloor");
const adType = useFieldModel("adType");
const format = useFieldModel("format");
const accountId = useFieldModel("accountId");
const accountType = useFieldModel("accountType");
const code = useFieldModel("code");

const response = await axios.get("/demand_source_accounts");
const demandSourceAccountsAll = response.data;
const demandSourceAccounts = computed(() =>
  demandSourceAccountsAll.filter(
    (account) => account.type === accountType.value
  )
);

const apiKey = computed(() =>
  accountType.value ? accountType.value.split("::")[1].toLowerCase() : ""
);

// reset accountId when accountType changes
watch(accountType, () => (accountId.value = null));

const onSubmit = handleSubmit((values) => emit("submit", values));
</script>
