<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton v-if="isReady && resource" :path="backPath" />
    </NavigationContainer>
    <DemandSourceAccountForm
      v-if="isReady"
      :value="resource"
      :submit-error="error"
      @submit="handleSubmit"
    />
  </PageContainer>
</template>

<script setup>
import { useAsyncState } from "@vueuse/core";
import axios from "@/services/ApiService";

const route = useRoute();
const id = route.params.id;
const resourcePath = `/demand_source_accounts/${id}`;

const { state: resource, isReady } = useAsyncState(async () => {
  const response = await axios.get(resourcePath);
  return response.data;
});

const backPath = computed(() =>
  resource.value?.demandSourceId != null
    ? `/demand_sources/${resource.value.demandSourceId}`
    : "/demand_sources",
);

const error = ref(null);
const handleSubmit = useUpdateResource({
  path: resourcePath,
  message: "Demand source account updated!",
  onError: async (e) => (error.value = e),
});
</script>
