<template>
  <PageContainer>
    <NavigationContainer>
      <GoBackButton :path="resourcesPath" />
      <EditButton
        v-if="resource._permissions.update"
        :id="id"
        :path="resourcesPath"
      />
    </NavigationContainer>
    <ResourceCard
      title="Auction Configuration"
      :fields="fields"
      :resource="resource"
    >
      <NetworksDisplay
        :app-id="resource.appId"
        :ad-type="resource.adType"
        :bidding="resource.bidding"
        :ad-unit-ids="resource.adUnitIds"
      />
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

<script setup>
import useDeleteResource from "@/composables/useDeleteResource";
import axios from "@/services/ApiService.js";
import { ResourceCardFields } from "@/constants";

const route = useRoute();
const id = route.params.id;
const resourcesPath = "/v2/auction_configurations";

const response = await axios.get(`${resourcesPath}/${id}`);
const resource = response.data;

const deleteHandle = useDeleteResource({
  path: resourcesPath,
  hook: async () => await navigateTo(resourcesPath),
});

const fields = [
  ResourceCardFields.PublicUid,
  ResourceCardFields.App,
  { label: "Name", key: "name" },
  { label: "Auction Key", key: "auctionKey", copyable: true },
  { label: "Ad type", key: "adType" },
  { label: "Price floor", key: "pricefloor" },
  { label: "Timeout", key: "timeout" },
  { label: "Default", key: "isDefault" },
  { label: "External Win Notification", key: "externalWinNotifications" },
  ResourceCardFields.Segment,
];
</script>
