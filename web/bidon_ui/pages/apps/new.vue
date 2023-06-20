<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton />
    </NavigationContainer>
    <AppForm :value="resource" @submit="handleSubmit" />
  </PageContainer>
</template>

<script setup>
const resource = {};
const resourcesPath = "/apps";
const handleSubmit = useCreateResource({ path: resourcesPath, message: "App created!" });
</script>
