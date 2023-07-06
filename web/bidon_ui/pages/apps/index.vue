<template>
  <CreateResourceButton label="New App" :resources-path="resourcesPath" />
  <ResourcesTable :columns="columns" :resources-path="resourcesPath" />
</template>

<script setup>
const columns = [
  { field: "id", header: "Id", sortable: true },
  { field: "platformId", header: "Platform Id" },
  { field: "humanName", header: "Human Name" },
  { field: "packageName", header: "Package Name" },
  {
    field: "user_id",
    header: "User",
    link: { basePath: "users", textField: "email", dataField: "user" },
  },
  { field: "appKey", header: "App Key" },
];
const resourcesPath = "/apps";
</script>
