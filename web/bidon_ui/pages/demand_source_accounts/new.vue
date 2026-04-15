<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton :path="backPath" />
    </NavigationContainer>
    <DemandSourceAccountForm
      :value="resource"
      :submit-error="error"
      :lock-demand-source="lockDemandSource"
      @submit="handleSubmit"
    />
  </PageContainer>
</template>

<script setup>
import { NETWORK_ACCOUNT_TYPE_BY_KEY } from "@/constants/Networks.js";

const route = useRoute();
const demandSourceId = route.query.demand_source_id;

const backPath = demandSourceId
  ? `/demand_sources/${demandSourceId}`
  : "/demand_sources";

const resource = ref({});
const lockDemandSource = ref(false);

if (demandSourceId) {
  const ds = await $apiFetch(`/demand_sources/${demandSourceId}`);
  const accountType =
    NETWORK_ACCOUNT_TYPE_BY_KEY[String(ds.apiKey).toLowerCase()] ?? null;
  resource.value = {
    demandSourceId: Number(demandSourceId),
    type: accountType,
  };
  lockDemandSource.value = accountType != null;
}

const resourcesPath = "/demand_source_accounts";

const error = ref(null);
const handleSubmit = useCreateResource({
  path: resourcesPath,
  message: "Demand source account created!",
  onError: async (e) => (error.value = e),
});
</script>
