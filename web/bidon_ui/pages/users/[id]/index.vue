<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton :path="resourcesPath" />
    </NavigationContainer>

    <UserOverviewCard
      :id="id"
      :resource="resource"
      :resources-path="resourcesPath"
    />
  </PageContainer>
</template>

<script setup>
import axios from "@/services/ApiService.js";

const route = useRoute();
const id = route.params.id;
const resourcesPath = "/users";

const response = await axios.get(`${resourcesPath}/${id}`);
const resource = response.data;
</script>
