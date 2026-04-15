<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton :path="backPath" />
      <DestroyButton
        v-if="resource._permissions.delete"
        :id="id"
        :path="accountResourcesPath"
      />
      <EditButton
        v-if="resource._permissions.update"
        :id="id"
        :path="accountResourcesPath"
      />
    </NavigationContainer>
    <ResourceCard
      title="Demand Source Account"
      :fields="fields"
      :resource="resource"
    />
  </PageContainer>
</template>

<script setup>
import axios from "@/services/ApiService.js";
import { ResourceCardFields } from "@/constants";

const route = useRoute();
const id = route.params.id;
const accountResourcesPath = "/demand_source_accounts";

const response = await axios.get(`${accountResourcesPath}/${id}`);
const resource = response.data;

const backPath = `/demand_sources/${resource.demandSourceId}`;

const jsonFields = jsonToFields(resource.extra, "extra", "static", true);
const fields = [
  ResourceCardFields.PublicUid,
  ResourceCardFields.DemandSource,
  ResourceCardFields.Owner,
  { label: "Label", key: "label" },
  ...jsonFields,
];
</script>
