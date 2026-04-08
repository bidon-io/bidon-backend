<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton :path="resourcesPath" />
    </NavigationContainer>
    <AuctionConfigurationsForm :value="resource" @submit="handleSubmit" />
  </PageContainer>
</template>

<script setup>
const route = useRoute();
const appId = route.query.app_id;
const resource = appId ? { appId: Number(appId) } : {};
const resourcesPath = appId ? `/apps/${appId}` : "/v2/auction_configurations";
const handleSubmit = useCreateResource({
  path: resourcesPath,
  message: "Auction configuration created!",
});
</script>
