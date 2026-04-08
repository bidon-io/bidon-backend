<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton :path="resourcesPath" />
    </NavigationContainer>
    <LineItemForm
      :value="resource"
      :submit-error="error"
      @submit="handleSubmit"
    />
  </PageContainer>
</template>

<script setup>
const route = useRoute();
const appId = route.query.app_id;
const adType = route.query.ad_type;
const resource = {
  ...(appId ? { appId: Number(appId) } : {}),
  // banner requires a specific format so we can't pre-select it
  ...(adType && adType !== "banner" ? { adType, format: "" } : {}),
};
const resourcesPath = appId ? `/apps/${appId}` : "/line_items";

const error = ref(null);
const handleSubmit = useCreateResource({
  path: resourcesPath,
  message: (response) =>
    response.status === 200
      ? "Line item with these attributes already exists!"
      : "Line Item created!",
  onError: async (e) => (error.value = e),
});
</script>
