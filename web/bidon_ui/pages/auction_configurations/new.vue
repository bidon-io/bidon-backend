<template>
  <Toast />
  <AuctionConfigurationForm :value="resource" @submit="handleSubmit" />
</template>

<script setup>
import axios from "@/services/ApiService.js";
import { useToast } from "primevue/usetoast";
const resource = {};

const toast = useToast();
const handleSubmit = (event) => {
  axios
    .post("/auction_configurations", event)
    .then(() => {
      toast.add({
        severity: "success",
        summary: "Success",
        detail: "Auction configuration created",
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
