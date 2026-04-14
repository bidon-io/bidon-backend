<script lang="ts">
/**
 * FilterConfig — exported so pages can import the type alongside the component.
 *
 * Usage:
 *   import type { FilterConfig } from '~/components/resources/ResourcesGrid.vue'
 */
export interface FilterConfig {
  /** Reactive key used to store the current filter value */
  key: string;
  /** Label shown above the input */
  label: string;
  type: 'text' | 'select';
  placeholder?: string;
  /** Optional Tailwind col-span class, e.g. 'lg:col-span-2' */
  colSpan?: string;
  /** Static options for select inputs */
  options?: Array<{ label: string; value: string }>;
  /**
   * Returns true when the item should be included for the given filter value.
   * Called once per item per non-empty filter.
   */
  match: (item: Record<string, unknown>, value: string) => boolean;
}
</script>

<script setup lang="ts">
import type { FilterConfig } from './ResourcesGrid.vue';

const props = withDefaults(
  defineProps<{
    /** API path — used for both fetching and CRUD operations */
    resourcesPath: string;
    /** Route for the "New …" button. Omit to hide the button. */
    newPath?: string;
    /** Label for the "New …" button (default: "New") */
    newLabel?: string;
    /** Filter definitions. Omit to hide the filter bar. */
    filters?: FilterConfig[];
    /**
     * Tailwind grid class applied to the filter row.
     * Default: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-4'
     */
    filterGridClass?: string;
    /** Items per page (default: 12) */
    pageSize?: number;
    /** Empty-state body text */
    emptyMessage?: string;
    /** Toast detail after a successful delete */
    deleteSuccessDetail?: string;
  }>(),
  {
    newLabel: 'New',
    filters: () => [],
    filterGridClass: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-4',
    pageSize: 12,
    emptyMessage: 'No records found.',
    deleteSuccessDetail: 'Record deleted.',
  },
);

// ── Fetch ───────────────────────────────────────────────────────────────────

type AnyRecord = Record<string, unknown> & { id: number; _permissions?: { update?: boolean; delete?: boolean } };

const { data: items, pending, error, execute: refresh } = useAsyncData<AnyRecord[]>(
  `resources-grid-${props.resourcesPath}`,
  () => $apiFetch<AnyRecord[]>(props.resourcesPath),
  { default: (): AnyRecord[] => [] },
);

// ── Filters ─────────────────────────────────────────────────────────────────

const filterValues = reactive<Record<string, string>>(
  Object.fromEntries((props.filters ?? []).map((f) => [f.key, ''])),
);

const hasFilters = computed(() => Object.values(filterValues).some((v) => v !== ''));

watch(filterValues, () => { currentPage.value = 1; });

const filteredItems = computed(() => {
  const all = items.value ?? [];
  const activeFilters = (props.filters ?? []).filter((f) => filterValues[f.key] !== '');
  if (!activeFilters.length) return all;

  return all.filter((item) =>
    activeFilters.every((f) => f.match(item, filterValues[f.key])),
  );
});

// ── Pagination ───────────────────────────────────────────────────────────────

const currentPage = ref(1);
const size = computed(() => props.pageSize);

const totalPages = computed(() => Math.max(1, Math.ceil(filteredItems.value.length / size.value)));
const pageStart = computed(() => (currentPage.value - 1) * size.value + 1);
const pageEnd = computed(() => Math.min(currentPage.value * size.value, filteredItems.value.length));

const currentPageItems = computed(() =>
  filteredItems.value.slice(
    (currentPage.value - 1) * size.value,
    currentPage.value * size.value,
  ),
);

const pageNumbers = computed((): (number | '...')[] => {
  const total = totalPages.value;
  const cur = currentPage.value;
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
  const pages: (number | '...')[] = [1];
  if (cur > 3) pages.push('...');
  for (let p = Math.max(2, cur - 1); p <= Math.min(total - 1, cur + 1); p++) pages.push(p);
  if (cur < total - 2) pages.push('...');
  pages.push(total);
  return pages;
});

// ── Delete ───────────────────────────────────────────────────────────────────

const deleteHandle = useDeleteResource({
  path: props.resourcesPath,
  hook: () => refresh(),
  successDetail: props.deleteSuccessDetail,
});
</script>

<template>
  <!-- ── Filter toolbar ───────────────────────────────────────────────────── -->
  <div v-if="filters.length" :class="['grid gap-2 mb-4', filterGridClass]">
    <label
      v-for="filter in filters"
      :key="filter.key"
      class="resources-grid-filter-group"
      :class="filter.colSpan"
    >
      <span class="resources-grid-filter-label">{{ filter.label }}</span>

      <select v-if="filter.type === 'select'" v-model="filterValues[filter.key]" class="resources-grid-filter-input">
        <option value="">{{ filter.placeholder ?? 'All' }}</option>
        <option v-for="opt in filter.options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
      </select>

      <div v-else class="relative">
        <input
          v-model="filterValues[filter.key]"
          type="text"
          :placeholder="filter.placeholder ?? 'Filter…'"
          class="resources-grid-filter-input pr-7"
        />
        <i class="pi pi-search absolute right-2.5 top-1/2 -translate-y-1/2 text-xs pointer-events-none" style="color: var(--bidon-muted);" />
      </div>
    </label>
  </div>

  <!-- ── New button ───────────────────────────────────────────────────────── -->
  <div v-if="newPath" class="flex justify-end mb-4">
    <NuxtLink :to="newPath" class="btn-new">
      <i class="pi pi-plus text-xs" /> {{ newLabel }}
    </NuxtLink>
  </div>

  <!-- ── Loading skeleton ─────────────────────────────────────────────────── -->
  <div v-if="pending" class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
    <div v-for="i in pageSize" :key="i" class="card" style="min-height: 160px; opacity: 0.45;">
      <div class="card-header">
        <div class="flex-1 space-y-2">
          <div class="h-3 rounded w-3/4" style="background-color: var(--bidon-border-default);" />
          <div class="h-2 rounded w-1/2" style="background-color: var(--bidon-border-default);" />
        </div>
      </div>
      <div class="card-body space-y-2 py-3">
        <div class="h-2.5 rounded w-2/3" style="background-color: var(--bidon-border-default);" />
        <div class="h-2.5 rounded w-1/3" style="background-color: var(--bidon-border-default);" />
      </div>
    </div>
  </div>

  <!-- ── Error ────────────────────────────────────────────────────────────── -->
  <Message v-else-if="error" severity="error" closable>{{ error.message }}</Message>

  <!-- ── Empty state ──────────────────────────────────────────────────────── -->
  <div v-else-if="filteredItems.length === 0" class="card">
    <div class="card-body flex flex-col items-center py-16 gap-3">
      <i class="pi pi-inbox text-3xl" style="color: var(--bidon-muted);" />
      <p class="text-sm" style="color: var(--bidon-muted);">
        {{ hasFilters ? 'No records match your filters.' : emptyMessage }}
      </p>
      <NuxtLink v-if="!hasFilters && newPath" :to="newPath" class="btn-new mt-1">
        <i class="pi pi-plus text-xs" /> {{ newLabel }}
      </NuxtLink>
    </div>
  </div>

  <!-- ── Card grid ────────────────────────────────────────────────────────── -->
  <div v-else class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
    <slot
      v-for="item in currentPageItems"
      :key="item.id"
      name="card"
      :item="item"
      :delete-handle="deleteHandle"
    />
  </div>

  <!-- ── Pagination ───────────────────────────────────────────────────────── -->
  <div v-if="totalPages > 1" class="flex items-center justify-between mt-6 flex-wrap gap-3">
    <span class="text-xs" style="color: var(--bidon-muted);">
      Showing {{ pageStart }}–{{ pageEnd }} of {{ filteredItems.length }}
    </span>
    <div class="flex items-center gap-1">
      <button class="rg-page-btn" :disabled="currentPage === 1" @click="currentPage--">
        <i class="pi pi-chevron-left text-xs" />
      </button>
      <template v-for="p in pageNumbers" :key="p">
        <span v-if="p === '...'" class="px-1 text-xs" style="color: var(--bidon-muted);">…</span>
        <button
          v-else
          class="rg-page-btn"
          :class="p === currentPage ? 'rg-page-btn--active' : ''"
          @click="currentPage = (p as number)"
        >{{ p }}</button>
      </template>
      <button class="rg-page-btn" :disabled="currentPage === totalPages" @click="currentPage++">
        <i class="pi pi-chevron-right text-xs" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.resources-grid-filter-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.resources-grid-filter-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--bidon-muted);
}

.resources-grid-filter-input {
  width: 100%;
  padding: 0.375rem 0.625rem;
  font-size: 0.8125rem;
  border-radius: 6px;
  background-color: var(--bidon-bg-card);
  border: 1px solid var(--bidon-border-default);
  color: var(--bidon-text-primary);
  outline: none;
  transition: border-color 0.15s ease;
  cursor: pointer;
}

.resources-grid-filter-input:focus {
  border-color: var(--bidon-border-focus);
}

.rg-page-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.75rem;
  height: 1.75rem;
  padding: 0 0.375rem;
  font-size: 0.8125rem;
  border-radius: 6px;
  border: 1px solid var(--bidon-border-default);
  background-color: var(--bidon-bg-card);
  color: var(--bidon-text-primary);
  cursor: pointer;
  transition: border-color 0.15s ease, background-color 0.15s ease;
}

.rg-page-btn:hover:not(:disabled) {
  border-color: var(--bidon-accent);
  color: var(--bidon-accent);
}

.rg-page-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.rg-page-btn--active {
  background-color: var(--bidon-accent);
  border-color: var(--bidon-accent);
  color: #fff;
  font-weight: 600;
}

.rg-page-btn--active:hover:not(:disabled) {
  background-color: var(--bidon-accent-hover);
  border-color: var(--bidon-accent-hover);
  color: #fff;
}
</style>
