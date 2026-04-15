<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton :path="resourcesPath" />
      <EditButton
        v-if="resource._permissions.update"
        :id="id"
        :path="resourcesPath"
      />
    </NavigationContainer>
    <ResourceCard title="Line Item" :fields="fields" :resource="resource">
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
const resourcesPath = "/line_items";

const response = await axios.get(`${resourcesPath}/${id}`);
const resource = response.data;

const deleteHandle = useDeleteResource({
  path: resourcesPath,
  hook: async () => await navigateTo(resourcesPath),
});

const jsonFields = jsonToFields(resource.extra, "extra", "static", true);
const fields = [
  ResourceCardFields.PublicUid,
  ResourceCardFields.HumanName,
  ResourceCardFields.App,
  ResourceCardFields.BidFloor,
  ResourceCardFields.AdType,
  ...(resource.format ? [{ key: "format", label: "Format" }] : []),
  ResourceCardFields.DemandSourceAccount,
  { key: "isBidding", label: "Bidding" },
  ...jsonFields,
];
</script>
