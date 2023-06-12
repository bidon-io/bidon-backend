<template>
  <Toast />
  <div class="flex-1 mx-auto w-full">
    <div class="flex mb-4 items-start space-x-2">
      <NuxtLink to="/auction_configurations/">
        <Button label="Go back" icon="pi pi-arrow-left" severity="secondary" text />
      </NuxtLink>
    </div>
    <AuctionConfigurationForm v-if="isReady" :value="resource" @submit="handleSubmit" />
  </div>
</template>

<script setup>
import { useAsyncState } from "@vueuse/core";
import axios from "@/services/ApiService.js";
import { useToast } from "primevue/usetoast";

const route = useRoute();
const id = route.params.id;

const { state: resource, isReady } = useAsyncState(async () => {
  const response = await axios.get(`/auction_configurations/${id}`);
  return response.data;
});

const toast = useToast();
const handleSubmit = () => {
  axios
    .patch(`/auction_configurations/${id}`, resource.value)
    .then(() => {
      toast.add({
        severity: "success",
        summary: "Success",
        detail: "Auction configuration updated",
      });
    })
    .catch((error) => {
      console.error(error);
      toast.add({
        severity: "error",
        summary: "Error",
        detail: error.message,
      });
    });
};
</script>
