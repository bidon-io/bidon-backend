<template>
  <ResourcesGrid
    resources-path="/demand_source_accounts"
    new-path="/demand_source_accounts/new"
    new-label="New Account"
    :filters="filters"
    empty-message="No demand source accounts yet."
    delete-success-detail="Account deleted."
  >
    <template #card="{ item, deleteHandle }">
      <div class="card flex flex-col">

        <!-- Card header: label + account type badge -->
        <div class="card-header gap-3">
          <div class="min-w-0">
            <NuxtLink
              :to="`/demand_source_accounts/${item.id}`"
              class="font-semibold text-sm block truncate table-resource-link"
            >{{ item.label || `Account #${item.id}` }}</NuxtLink>
            <span v-if="item.demandSource?.humanName" class="text-xs block truncate mt-0.5" style="color: var(--bidon-muted);">
              {{ item.demandSource.humanName }}
            </span>
          </div>
          <span v-if="item.type" class="badge badge-platform shrink-0">
            {{ String(item.type).split('::')[1] }}
          </span>
        </div>

        <!-- Card body: owner -->
        <div class="card-body flex-1 py-3 space-y-1.5">
          <div v-if="item.user?.email" class="flex items-center gap-2">
            <i class="pi pi-user text-xs shrink-0 w-3.5" style="color: var(--bidon-muted);" />
            <NuxtLink :to="`/users/${item.user.id}`" class="text-xs truncate table-resource-link">
              {{ item.user.email }}
            </NuxtLink>
          </div>
        </div>

        <!-- Card footer: actions -->
        <div class="card-footer flex items-center justify-between">
          <NuxtLink :to="`/demand_source_accounts/${item.id}`" class="btn-view btn-sm">
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
    label: 'Label',
    type: 'text',
    placeholder: 'Search…',
    match: (item, value) =>
      String(item.label ?? '').toLowerCase().includes(value.trim().toLowerCase()),
  },
  {
    key: 'demandSource',
    label: 'Demand Source',
    type: 'text',
    placeholder: 'Filter…',
    match: (item, value) => {
      const ds = item.demandSource as { humanName?: string } | undefined;
      return String(ds?.humanName ?? '').toLowerCase().includes(value.trim().toLowerCase());
    },
  },
  {
    key: 'owner',
    label: 'Owner',
    type: 'text',
    placeholder: 'Email…',
    match: (item, value) => {
      const user = item.user as { email?: string } | undefined;
      return String(user?.email ?? '').toLowerCase().includes(value.trim().toLowerCase());
    },
  },
];
</script>
