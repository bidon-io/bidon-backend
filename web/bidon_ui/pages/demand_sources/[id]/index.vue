<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton :path="resourcesPath" />
      <DestroyButton
        v-if="resource._permissions.delete"
        :id="id"
        :path="resourcesPath"
      />
      <EditButton
        v-if="resource._permissions.update"
        :id="id"
        :path="resourcesPath"
      />
    </NavigationContainer>
    <ResourceCard title="Demand Source" :fields="fields" :resource="resource" />
    <DemandSourceAccountsSection
      :accounts="accounts"
      :demand-source-id="id"
      :demand-source-api-key="resource.apiKey"
      @refresh="reloadAccounts"
    />
  </PageContainer>
</template>

<script setup>
import axios from "@/services/ApiService.js";
import { ResourceCardFields } from "@/constants";

const route = useRoute();
const id = route.params.id;
const resourcesPath = "/demand_sources";

const response = await axios.get(`${resourcesPath}/${id}`);
const resource = response.data;

const accounts = ref([]);
async function reloadAccounts() {
  const result = await $apiFetch("/demand_source_accounts", {
    params: { demandSourceId: id },
  });
  accounts.value = (
    Array.isArray(result) ? result : (result?.items ?? [])
  ).filter((account) => Number(account.demandSourceId) === Number(id));
}
await reloadAccounts();

const fields = [
  ResourceCardFields.Id,
  { label: "Human Name", key: "humanName" },
  { label: "Api Key", key: "apiKey", copyable: true },
];
</script>
