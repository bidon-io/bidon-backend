<template>
  <form @submit.prevent="onSubmit">
    <FormCard title="Demand source account">
      <UserDropdown v-model="userId" :error="errors.userId" required />
      <DemandSourceTypeDropdown v-model="demandSourceType" :error="errors.demandSourceType" required />
      <DemandSourceDropdown v-model="demandSourceId" :error="errors.demandSourceId" required />
      <FormField label="Human Name" :error="errors.humanName" required>
        <InputText v-model="humanName" type="text" placeholder="Name" />
      </FormField>
      <FormField label="Bidding">
        <Checkbox v-model="isBidding" :binary="true" />
      </FormField>
      <FormField label="Extra">
        <TextareaJSON v-model="extra" rows="5" />
      </FormField>
      <FormSubmitButton />
    </FormCard>
  </form>
</template>

<script setup>
import { useForm } from "vee-validate";
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
    userId: yup.number().required().label("User Id"),
    demandSourceType: yup.string().required().label("Demand Source Type"),
    demandSourceId: yup.number().required().label("Deamand Source Id"),
    humanName: yup.string().required().label("Human Name"),
    isBidding: yup.boolean(),
    extra: yup.object(),
  }),
  initialValues: {
    userId: resource.value.userId || null,
    demandSourceType: resource.value.demandSourceType || "",
    demandSourceId: resource.value.demandSourceId || null,
    humanName: resource.value.humanName || "",
    isBidding: resource.value.isBidding || false,
    extra: resource.value.extra || {},
  },
});

const userId = useFieldModel("userId");
const demandSourceType = useFieldModel("demandSourceType");
const demandSourceId = useFieldModel("demandSourceId");
const humanName = useFieldModel("humanName");
const isBidding = useFieldModel("isBidding");
const extra = useFieldModel("extra");

const onSubmit = handleSubmit((values) => emit("submit", values));
</script>
