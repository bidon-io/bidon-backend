<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton :path="resourcesPath" />
    </NavigationContainer>
    <ResourceCard title="API Credential" :fields="fields" :resource="resource">
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

<script setup lang="ts">
import useDeleteResource from "@/composables/useDeleteResource";
import { $apiFetch } from "~/utils/$apiFetch";

const route = useRoute();
const id = route.params.id;

const resourcesPath = "/api_keys";

const resource = await $apiFetch(`${resourcesPath}/${id}`);
const endpoint = `${window.location.origin}/api`;

const fields = [
  { label: "ID", key: "id" },
  { label: "Endpoint", type: "static", value: endpoint, copyable: true },
  { label: "API Key", key: "value", copyable: true },
  { label: "Last Accessed At", key: "lastAccessedAt" },
];

const deleteHandle = useDeleteResource({
  path: resourcesPath,
  hook: async () => await navigateTo(resourcesPath),
  successDetail: "API credential deleted.",
});
</script>
