<template>
  <nav class="mt-6">
    <NuxtLink
      v-for="item in menuItems"
      :key="item.resourceNname"
      :to="item.path"
      :class="[
        'flex items-center mt-4 px-6 py-2 text-gray-600 hover:bg-gray-700 hover:bg-opacity-25 hover:text-gray-100',
        route.path === item.path
          ? 'bg-gray-700 bg-opacity-25 text-blue-100'
          : '',
      ]"
    >
      <span>{{ item.resourceName }}</span>
    </NuxtLink>
  </nav>
</template>

<script setup>
import { useAuthStore } from "@/stores/AuthStore";

const { getResourcesPermissions } = useAuthStore();
const { data: menuItems } = useAsyncData(getResourcesPermissions);

// const menuItems = [
//   { name: "App Demand Profile", link: "/app_demand_profiles" },
//   { name: "Apps", link: "/apps" },
//   { name: "Auction Configurations", link: "/auction_configurations" },
//   { name: "Demand Source Accounts", link: "/demand_source_accounts" },
//   { name: "Line Items", link: "/line_items" },
//   { name: "Segments", link: "/segments" },
//   { name: "Demand Sources", link: "/demand_sources" },
//   { name: "Country", link: "/countries" },
//   { name: "User", link: "/users" },
// ];
const route = useRoute();
</script>
