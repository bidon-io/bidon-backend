<template>
  <Toast />
  <ConfirmDialog />
  <div class="flex-1 mx-auto w-full">
    <div class="flex mb-4 items-start space-x-2">
      <NuxtLink to="/auction_configurations/">
        <Button label="Go back" icon="pi pi-arrow-left" severity="secondary" text />
      </NuxtLink>
      <a href="_" @:click.prevent="deleteHandle(id)">
        <Button label="Delete" icon="pi pi pi-trash" severity="danger" />
      </a>
      <NuxtLink :to="`/auction_configurations/${id}/edit`">
        <Button label="Edit" icon="pi pi-pencil" />
      </NuxtLink>
    </div>
    <ResourceCard :fields="fields" :resource="resource" />
  </div>
</template>

<script setup>
import axios from "@/services/ApiService.js";
const route = useRoute();
const id = route.params.id;
const deleteHandle = useDeleteResource(
  "auction_configurations",
  async () => await navigateTo("/auction_configurations")
);

const response = await axios.get(`auction_configurations/${id}`);
const resource = response.data;
const fields = [
  { label: "ID", key: "id" },
  { label: "App", key: "app", type: "link", link: `/apps/${resource.app_id}` },
  { label: "Name", key: "name" },
  { label: "Ad type", key: "adType" },
  { label: "Price floor", key: "priceFloor" },
  { label: "Rounds", key: "rounds", type: "textarea" },
];
</script>
