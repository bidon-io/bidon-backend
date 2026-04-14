<template>
  <ResourcesGrid
    resources-path="/apps"
    new-path="/apps/new"
    new-label="New App"
    :filters="filters"
    filter-grid-class="grid-cols-1 sm:grid-cols-2 lg:grid-cols-5"
    :page-size="12"
    empty-message="No apps yet."
    delete-success-detail="App deleted."
  >
    <template #card="{ item: app, deleteHandle }">
      <div class="card flex flex-col">

        <!-- Card header: name + platform badge -->
        <div class="card-header gap-3">
          <div class="min-w-0">
            <NuxtLink
              :to="`/apps/${app.id}`"
              class="font-semibold text-sm block truncate table-resource-link"
            >{{ app.humanName }}</NuxtLink>
            <span class="text-xs block truncate mt-0.5 font-mono" style="color: var(--bidon-muted); font-size: 0.7rem;">
              {{ app.packageName }}
            </span>
          </div>
          <span class="badge badge-platform shrink-0">{{ app.platformId }}</span>
        </div>

        <!-- Card body: owner, store ID, auction config count -->
        <div class="card-body flex-1 py-3 space-y-1.5">
          <div v-if="app.user?.email" class="flex items-center gap-2">
            <i class="pi pi-user text-xs shrink-0 w-3.5" style="color: var(--bidon-muted);" />
            <NuxtLink :to="`/users/${app.user.id}`" class="text-xs truncate table-resource-link">
              {{ app.user.email }}
            </NuxtLink>
          </div>
          <div v-if="app.storeId" class="flex items-center gap-2">
            <i class="pi pi-tag text-xs shrink-0 w-3.5" style="color: var(--bidon-muted);" />
            <span class="text-xs truncate" style="color: var(--bidon-text-secondary);">{{ app.storeId }}</span>
          </div>

          <!-- Auction config count — info only -->
          <div class="flex items-center gap-1.5 pt-2 mt-1 border-t" style="border-color: var(--bidon-border-default);">
            <span class="badge badge-count-rtb">
              <i class="pi pi-sliders-h" style="font-size: 0.6rem;" />
              {{ auctionConfigCount(app.id as number) }} Auction {{ auctionConfigCount(app.id as number) === 1 ? 'Config' : 'Configs' }}
            </span>
          </div>
        </div>

        <!-- Card footer: actions -->
        <div class="card-footer flex items-center justify-between">
          <NuxtLink :to="`/apps/${app.id}`" class="btn-view btn-sm">
            <i class="pi pi-eye text-xs" /> View
          </NuxtLink>
          <button
            v-if="app._permissions?.delete"
            class="table-action-btn table-action-btn--delete"
            title="Delete"
            @click="deleteHandle(app.id as number)"
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

interface CountableItem { appId: number }
interface Collection<T> { items: T[]; meta: { totalCount: number } }

// Fetch auction config counts in parallel with the apps fetch (done inside ResourcesGrid)
const { data: auctionConfigData } = useAsyncData(
  'apps-auction-configs',
  () => $apiFetch<Collection<CountableItem>>('/v2/auction_configurations_collection', {
    params: { limit: 10000 },
  }),
  { default: () => null },
);

const auctionConfigCounts = computed(() => {
  const map = new Map<number, number>();
  for (const item of auctionConfigData.value?.items ?? []) {
    map.set(item.appId, (map.get(item.appId) ?? 0) + 1);
  }
  return map;
});

const auctionConfigCount = (appId: number) => auctionConfigCounts.value.get(appId) ?? 0;

const filters: FilterConfig[] = [
  {
    key: 'search',
    label: 'Name / Package',
    type: 'text',
    placeholder: 'Search…',
    colSpan: 'lg:col-span-2',
    match: (item, value) => {
      const q = value.trim().toLowerCase();
      return (
        String(item.humanName ?? '').toLowerCase().includes(q) ||
        String(item.packageName ?? '').toLowerCase().includes(q)
      );
    },
  },
  {
    key: 'platform',
    label: 'Platform',
    type: 'select',
    placeholder: 'All',
    options: [
      { label: 'iOS', value: 'ios' },
      { label: 'Android', value: 'android' },
    ],
    match: (item, value) => item.platformId === value,
  },
  {
    key: 'storeId',
    label: 'Store ID',
    type: 'text',
    placeholder: 'Filter…',
    match: (item, value) =>
      String(item.storeId ?? '').toLowerCase().includes(value.trim().toLowerCase()),
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
