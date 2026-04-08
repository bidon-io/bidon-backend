<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton :path="resourcesPath" />
    </NavigationContainer>

    <AppOverviewCard
      :id="id"
      :resource="resource"
      :resources-path="resourcesPath"
    />

    <AppDemandProfilesSection :demand-profiles="demandProfiles" :app-id="id" />

    <AppAuctionConfigurationsSection
      :initial-auction-configs="initialAuctionConfigs"
      :initial-line-items="initialLineItems"
      :app-id="id"
    />
  </PageContainer>
</template>

<script setup>
import axios from "@/services/ApiService.js";

const route = useRoute();
const id = route.params.id;
const resourcesPath = "/apps";

const [appResponse, configsData, lineItemsData, demandProfilesData] =
  await Promise.all([
    axios.get(`/apps/${id}`),
    $apiFetch("/v2/auction_configurations_collection", {
      params: { app_id: id, page: 1, limit: 200 },
    }),
    $apiFetch("/line_items_collection", {
      params: { app_id: id, page: 1, limit: 1000 },
    }),
    $apiFetch("/app_demand_profiles_collection", {
      params: { app_id: id, page: 1, limit: 200 },
    }),
  ]);

const resource = appResponse.data;
const demandProfiles = demandProfilesData.items ?? [];
const initialAuctionConfigs = configsData.items ?? [];
const initialLineItems = lineItemsData.items ?? [];
</script>
