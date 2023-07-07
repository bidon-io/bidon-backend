<template>
  <CreateResourceButton
    label="New Auction Configuration"
    :resources-path="resourcesPath"
  />
  <ResourcesTable :columns="columns" :resources-path="resourcesPath" />
</template>

<script setup>
const columns = [
  { field: "id", header: "Id", sortable: true },
  {
    field: "appId",
    header: "App",
    link: { basePath: "apps", textField: "packageName", dataField: "app" },
  },
  { field: "name", header: "Name" },
  { field: "adType", header: "Ad Type" },
  { field: "pricefloor", header: "Price Floor" },
  {
    field: "segmentId",
    header: "Segment",
    link: { basePath: "segments", textField: "name", dataField: "segment" },
  },
];
const resourcesPath = "/auction_configurations";
</script>
