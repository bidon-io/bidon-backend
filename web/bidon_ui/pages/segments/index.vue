<template>
  <ResourcesGrid
    resources-path="/segments"
    new-path="/segments/new"
    new-label="New Segment"
    :filters="filters"
    empty-message="No segments yet."
    delete-success-detail="Segment deleted."
  >
    <template #card="{ item, deleteHandle }">
      <div class="card flex flex-col">
        <!-- Card header: name + enabled badge -->
        <div class="card-header gap-3">
          <div class="min-w-0">
            <NuxtLink
              :to="`/segments/${item.id}`"
              class="font-semibold text-sm block truncate table-resource-link"
              >{{ item.name }}</NuxtLink
            >
            <span
              v-if="item.app?.packageName"
              class="text-xs block truncate mt-0.5 font-mono"
              style="color: var(--bidon-muted); font-size: 0.7rem"
            >
              {{ item.app.packageName }}
            </span>
          </div>
          <span
            class="badge shrink-0"
            :class="item.enabled ? 'badge-active' : 'badge-disabled'"
          >
            {{ item.enabled ? "Enabled" : "Disabled" }}
          </span>
        </div>

        <!-- Card body: app platform + priority -->
        <div class="card-body flex-1 py-3 space-y-1.5">
          <div v-if="item.app?.platformId" class="flex items-center gap-2">
            <i
              class="pi pi-mobile text-xs shrink-0 w-3.5"
              style="color: var(--bidon-muted)"
            />
            <span class="badge badge-platform">{{ item.app.platformId }}</span>
          </div>
          <div v-if="item.priority != null" class="flex items-center gap-2">
            <i
              class="pi pi-sort-amount-up text-xs shrink-0 w-3.5"
              style="color: var(--bidon-muted)"
            />
            <span class="text-xs" style="color: var(--bidon-text-secondary)"
              >Priority: {{ item.priority }}</span
            >
          </div>
        </div>

        <!-- Card footer: actions -->
        <div class="card-footer flex items-center justify-between">
          <NuxtLink :to="`/segments/${item.id}`" class="btn-view btn-sm">
            <i class="pi pi-eye text-xs" /> View
          </NuxtLink>
          <button
            v-if="item._permissions?.delete"
            type="button"
            class="table-action-btn table-action-btn--delete"
            title="Delete"
            aria-label="Delete"
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
import type { FilterConfig } from "~/components/resources/ResourcesGrid.vue";

const filters: FilterConfig[] = [
  {
    key: "search",
    label: "Name",
    type: "text",
    placeholder: "Search…",
    match: (item, value) =>
      String(item.name ?? "")
        .toLowerCase()
        .includes(value.trim().toLowerCase()),
  },
  {
    key: "platform",
    label: "Platform",
    type: "select",
    placeholder: "All",
    options: [
      { label: "iOS", value: "ios" },
      { label: "Android", value: "android" },
    ],
    match: (item, value) => {
      const app = item.app as { platformId?: string } | undefined;
      return app?.platformId === value;
    },
  },
  {
    key: "enabled",
    label: "Status",
    type: "select",
    placeholder: "All",
    options: [
      { label: "Enabled", value: "true" },
      { label: "Disabled", value: "false" },
    ],
    match: (item, value) => String(item.enabled) === value,
  },
];
</script>
