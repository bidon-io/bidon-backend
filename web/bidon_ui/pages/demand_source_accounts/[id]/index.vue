<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton :path="backPath" />
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
    >
      <template v-if="resource._permissions?.delete" #footer>
        <button
          type="button"
          class="table-action-btn table-action-btn--delete"
          title="Delete"
          aria-label="Delete"
          @click="deleteHandle(String(id))"
        >
          <i class="pi pi-trash" />
        </button>
      </template>
    </ResourceCard>
  </PageContainer>
</template>

<script setup>
import useDeleteResource from "@/composables/useDeleteResource";
import axios from "@/services/ApiService.js";
import { ResourceCardFields } from "@/constants";

const route = useRoute();
const id = route.params.id;
const accountResourcesPath = "/demand_source_accounts";

const response = await axios.get(`${accountResourcesPath}/${id}`);
const resource = response.data;

const deleteHandle = useDeleteResource({
  path: accountResourcesPath,
  hook: async () => await navigateTo(accountResourcesPath),
});

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
