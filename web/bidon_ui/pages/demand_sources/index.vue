<template>
  <ResourcesGrid
    resources-path="/demand_sources"
    new-path="/demand_sources/new"
    new-label="New Demand Source"
    :filters="filters"
    empty-message="No demand sources yet."
    delete-success-detail="Demand source deleted."
  >
    <template #card="{ item, deleteHandle }">
      <div class="card flex flex-col">

        <!-- Card header: name -->
        <div class="card-header">
          <div class="min-w-0">
            <NuxtLink
              :to="`/demand_sources/${item.id}`"
              class="font-semibold text-sm block truncate table-resource-link"
            >{{ item.humanName }}</NuxtLink>
          </div>
        </div>

        <!-- Card body: API key -->
        <div class="card-body flex-1 py-3 space-y-1.5">
          <div v-if="item.apiKey" class="flex items-center gap-2">
            <i class="pi pi-key text-xs shrink-0 w-3.5" style="color: var(--bidon-muted);" />
            <span class="text-xs font-mono truncate" style="color: var(--bidon-text-secondary);">{{ item.apiKey }}</span>
          </div>
        </div>

        <!-- Card footer: actions -->
        <div class="card-footer flex items-center justify-between">
          <NuxtLink :to="`/demand_sources/${item.id}`" class="btn-view btn-sm">
            <i class="pi pi-eye text-xs" /> View
          </NuxtLink>
          <button
            v-if="item._permissions?.delete"
            class="table-action-btn table-action-btn--delete"
            title="Delete"
            @click="deleteHandle(item.id as number)"
          >
            <i class="pi pi-trash" />
          </button>
        </div>

      </div>
    </template>
  </ResourcesGrid>
</template>

<script setup lang="ts">
import type { FilterConfig } from '~/components/resources/ResourcesGrid.vue';

const filters: FilterConfig[] = [
  {
    key: 'search',
    label: 'Name',
    type: 'text',
    placeholder: 'Search…',
    match: (item, value) =>
      String(item.humanName ?? '').toLowerCase().includes(value.trim().toLowerCase()),
  },
  {
    key: 'apiKey',
    label: 'API Key',
    type: 'text',
    placeholder: 'Filter…',
    match: (item, value) =>
      String(item.apiKey ?? '').toLowerCase().includes(value.trim().toLowerCase()),
  },
];
</script>
