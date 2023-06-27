<template>
  <CreateResourceButton label="New Demand Source Account" :resources-path="resourcesPath" />
  <ResourcesTable :columns="columns" :resources-path="resourcesPath" />
</template>

<script setup>
const columns = [
  { field: "id", header: "Id", sortable: true },
  { field: "user_id", header: "User" },
  { field: "type", header: "Type" },
  { field: "demand_source_id", header: "Demand Source" },
  { field: "is_bidding", header: "Bidding" },
];
const resourcesPath = "/demand_source_accounts";
</script>
