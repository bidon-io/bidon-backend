<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton :path="resourcesPath" />
    </NavigationContainer>
    <AppDemandProfileForm
      :value="resource"
      :submit-error="error"
      @submit="handleSubmit"
    />
  </PageContainer>
</template>

<script setup>
const route = useRoute();
const appId = route.query.app_id;
const resource = appId ? { appId: Number(appId) } : {};
const resourcesPath = appId ? `/apps/${appId}` : "/app_demand_profiles";

const error = ref(null);
const handleSubmit = useCreateResource({
  path: resourcesPath,
  message: "App demand profile created!",
  onError: async (e) => (error.value = e),
});
</script>
