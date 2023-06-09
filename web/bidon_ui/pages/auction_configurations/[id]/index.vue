<template>
  <ResourceCard :fields="fields" :resource="resource" />
</template>

<script setup>
import axios from "@/services/ApiService.js";
const route = useRoute();

const response = await axios.get(`auction_configurations/${route.params.id}`);
const resource = response.data;
const fields = [
  { label: "ID", key: "id" },
  { label: "App", key: "app", type: "link", link: "/apps/1" },
  { label: "Name", key: "name" },
  { label: "Ad type", key: "adType" },
  { label: "Price floor", key: "priceFloor" },
  { label: "Rounds", key: "rounds", type: "textarea" },
];
</script>
