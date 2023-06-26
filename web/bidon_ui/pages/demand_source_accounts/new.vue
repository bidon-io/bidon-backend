<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton />
    </NavigationContainer>
    <AuctionConfigurationForm :value="resource" @submit="handleSubmit" />
  </PageContainer>
</template>

<script setup>
const resource = {};
const resourcesPath = "/demand_source_accounts";
const handleSubmit = useCreateResource({ path: resourcesPath, message: "Auction configuration created!" });
</script>
