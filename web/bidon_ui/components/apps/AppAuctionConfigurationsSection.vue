<template>
  <div class="mt-10 mb-10">
    <div class="mb-4 flex flex-col gap-3">
      <div
        class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between"
      >
        <div class="min-w-0">
          <h3
            class="text-base font-semibold"
            style="color: var(--bidon-text-primary)"
          >
            Auction Configurations
          </h3>
          <p class="text-sm mt-0.5" style="color: var(--bidon-muted)">
            Per ad-type auction settings with CPM and bidding networks.
          </p>
        </div>
        <button
          type="button"
          :class="[
            'btn-sm shrink-0',
            showInlineConfigForm ? 'btn-cancel' : 'btn-new',
            'hidden md:inline-flex',
          ]"
          @click="toggleConfigForm"
        >
          <i
            :class="['pi', showInlineConfigForm ? 'pi-times' : 'pi-plus']"
            aria-hidden="true"
          />
          {{ showInlineConfigForm ? "Cancel" : "New Auction Config" }}
        </button>
      </div>

      <!-- Filters: full width on small screens -->
      <div
        v-if="auctionConfigs.length"
        class="flex w-full min-w-0 flex-col gap-2 md:flex-row md:flex-wrap md:items-center md:gap-3"
      >
        <div class="relative w-full md:w-auto">
          <svg
            class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400 pointer-events-none"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
            aria-hidden="true"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M21 21l-4.35-4.35M17 11A6 6 0 1 1 5 11a6 6 0 0 1 12 0z"
            />
          </svg>
          <label for="auction-config-search" class="sr-only">
            Search configurations by name
          </label>
          <input
            id="auction-config-search"
            v-model="configSearch"
            type="text"
            placeholder="Search by name…"
            class="w-full max-w-none rounded-lg py-1.5 pr-3 pl-8 text-sm md:w-52 md:max-w-full focus:outline-none"
            style="
              background-color: var(--bidon-bg-card);
              border: 1px solid var(--bidon-border-default);
              color: var(--bidon-text-primary);
              box-sizing: border-box;
            "
            @focus="
              $event.target.style.borderColor = 'var(--bidon-primary)';
              $event.target.style.boxShadow = 'var(--bidon-shadow-focus)';
            "
            @blur="
              $event.target.style.borderColor = 'var(--bidon-border-default)';
              $event.target.style.boxShadow = 'none';
            "
          />
        </div>
        <div
          class="flex w-full min-w-0 flex-wrap gap-1.5 md:w-auto md:justify-start"
        >
          <button
            :class="[
              'text-xs px-3 py-1 rounded-md border font-medium transition-colors',
            ]"
            :style="
              configAdTypeFilter === null
                ? 'background: var(--bidon-primary); border-color: var(--bidon-primary); color: #fff;'
                : 'background: var(--bidon-bg-card); border-color: var(--bidon-border-default); color: var(--bidon-muted);'
            "
            @click="configAdTypeFilter = null"
          >
            All
          </button>
          <button
            v-for="adTypeOption in configAdTypes"
            :key="adTypeOption"
            :class="[
              'text-xs px-3 py-1 rounded-md border font-medium capitalize transition-colors',
            ]"
            :style="
              configAdTypeFilter === adTypeOption
                ? 'background: var(--bidon-primary); border-color: var(--bidon-primary); color: #fff;'
                : 'background: var(--bidon-bg-card); border-color: var(--bidon-border-default); color: var(--bidon-muted);'
            "
            @click="configAdTypeFilter = adTypeOption"
          >
            {{ adTypeOption }}
          </button>
        </div>
      </div>

      <button
        type="button"
        :class="[
          'btn-sm w-full justify-center md:hidden',
          showInlineConfigForm ? 'btn-cancel' : 'btn-new',
        ]"
        @click="toggleConfigForm"
      >
        <i
          :class="['pi', showInlineConfigForm ? 'pi-times' : 'pi-plus']"
          aria-hidden="true"
        />
        {{ showInlineConfigForm ? "Cancel" : "New Auction Config" }}
      </button>
    </div>

    <InlineAuctionConfigForm
      v-if="showInlineConfigForm"
      :app-id="appId"
      :clone-from="cloneSourceConfig"
      @created="handleAuctionConfigCreated"
      @cancel="showInlineConfigForm = false"
      @line-item-created="registerLineItemFromConfigForm"
      @line-item-updated="handleLineItemUpdated"
      @line-item-deleted="handleLineItemDeleted"
    />

    <p
      v-if="!auctionConfigs.length"
      class="text-sm py-4"
      style="color: var(--bidon-muted)"
    >
      No auction configurations for this app yet.
    </p>
    <p
      v-else-if="!filteredConfigGroups.length"
      class="text-sm py-4"
      style="color: var(--bidon-muted)"
    >
      No configurations match the current filters.
    </p>

    <div v-for="group in filteredConfigGroups" :key="group.adType" class="mb-6">
      <div class="flex items-center gap-2 mb-3">
        <span class="badge badge-ad-type">{{ group.adType }}</span>
        <span class="text-sm" style="color: var(--bidon-muted)">
          {{ group.items.length }} config{{
            group.items.length !== 1 ? "s" : ""
          }}
        </span>
      </div>

      <div class="grid grid-cols-1 gap-4">
        <div
          v-for="config in group.items"
          :key="config.id"
          class="rounded-lg overflow-hidden flex flex-col"
          style="
            background-color: var(--bidon-bg-card);
            border: 1px solid var(--bidon-border-default);
            box-shadow: var(--bidon-shadow-sm);
          "
        >
          <div
            class="card-header cursor-pointer select-none flex flex-col gap-3 !items-stretch md:!flex-row md:items-start md:justify-between"
            @click="toggleCollapse(config.id)"
          >
            <div class="flex min-w-0 w-full flex-col gap-1 md:flex-1">
              <div class="flex items-center gap-2 flex-wrap">
                <span
                  class="font-semibold"
                  style="color: var(--bidon-text-primary)"
                >
                  {{ config.name }}
                </span>
                <span class="badge badge-ad-type">{{ config.adType }}</span>
                <span class="badge badge-segment">
                  {{
                    config.isDefault
                      ? "Default"
                      : (config.segment?.name ?? "Segment")
                  }}
                </span>
              </div>
              <div class="flex items-center gap-3 flex-wrap">
                <span
                  class="flex items-center gap-1 text-xs font-mono"
                  style="color: var(--bidon-muted)"
                >
                  <span style="color: var(--bidon-border-strong, #8c8c9c)"
                    >floor</span
                  >
                  ${{ config.pricefloor }}
                </span>
                <span
                  class="flex items-center gap-1 text-xs font-mono"
                  style="color: var(--bidon-muted)"
                >
                  <span style="color: var(--bidon-border-strong, #8c8c9c)"
                    >timeout</span
                  >
                  {{ config.timeout }}ms
                </span>
                <span
                  v-if="config.auctionKey"
                  class="flex items-center gap-1 text-xs font-mono"
                  style="color: var(--bidon-muted)"
                >
                  <span style="color: var(--bidon-border-strong, #8c8c9c)"
                    >key</span
                  >
                  {{ config.auctionKey }}
                  <button
                    :aria-label="`Copy auction key ${config.auctionKey}`"
                    class="table-action-btn"
                    @click.stop="copyField(config.auctionKey)"
                  >
                    <i class="pi pi-copy" aria-hidden="true" />
                  </button>
                </span>
                <span
                  v-if="config.publicUid"
                  class="flex items-center gap-1 text-xs font-mono"
                  style="color: var(--bidon-muted)"
                >
                  <span style="color: var(--bidon-border-strong, #8c8c9c)"
                    >uid</span
                  >
                  {{ config.publicUid }}
                  <button
                    :aria-label="`Copy public UID ${config.publicUid}`"
                    class="table-action-btn"
                    @click.stop="copyField(config.publicUid)"
                  >
                    <i class="pi pi-copy" aria-hidden="true" />
                  </button>
                </span>
              </div>
            </div>
            <div
              class="flex w-full shrink-0 items-center justify-between gap-2 md:w-auto md:justify-end"
            >
              <div class="flex flex-wrap gap-2" @click.stop>
                <button
                  type="button"
                  class="btn-new btn-sm"
                  @click="cloneAuctionConfig(config)"
                >
                  <i class="pi pi-clone" /> Clone
                </button>
                <button
                  type="button"
                  :class="[
                    'btn-sm',
                    editingConfigId === config.id ? 'btn-cancel' : 'btn-edit',
                  ]"
                  @click="toggleEditingAuctionConfig(config.id)"
                >
                  <i
                    :class="[
                      'pi',
                      editingConfigId === config.id ? 'pi-times' : 'pi-pencil',
                    ]"
                    aria-hidden="true"
                  />
                  {{ editingConfigId === config.id ? "Cancel" : "Edit" }}
                </button>
              </div>
              <i
                class="pi pi-chevron-down shrink-0 transition-transform duration-300"
                style="color: var(--bidon-muted)"
                :style="
                  collapsedConfigIds.has(config.id)
                    ? 'transform: rotate(-90deg)'
                    : ''
                "
                aria-hidden="true"
              />
            </div>
          </div>

          <Transition name="collapse">
            <div v-if="!collapsedConfigIds.has(config.id)" class="flex-1">
              <InlineAuctionConfigForm
                v-if="editingConfigId === config.id"
                :app-id="appId"
                :edit-config="config"
                class="!mb-0 !border-0 !rounded-none"
                @updated="handleAuctionConfigUpdated"
                @cancel="editingConfigId = null"
                @line-item-created="registerLineItemFromConfigForm"
                @line-item-updated="handleLineItemUpdated"
                @line-item-deleted="handleLineItemDeleted"
              />
              <div
                v-else
                class="divide-y"
                style="border-color: var(--bidon-border-default)"
              >
                <AuctionConfigNetworkSection
                  :config="config"
                  :is-bidding="true"
                  :all-line-items="allLineItems"
                  :show-inline-form="showInlineForm"
                  :app-id="appId"
                  @toggle-inline-form="toggleInlineForm"
                  @line-item-created="handleLineItemCreated"
                  @line-item-updated="handleLineItemUpdated"
                  @line-item-deleted="handleLineItemDeleted"
                  @dismiss-inline-create="showInlineForm = null"
                  @inline-form-cancel="showInlineForm = null"
                />
                <AuctionConfigNetworkSection
                  :config="config"
                  :is-bidding="false"
                  :all-line-items="allLineItems"
                  :show-inline-form="showInlineForm"
                  :app-id="appId"
                  @toggle-inline-form="toggleInlineForm"
                  @line-item-created="handleLineItemCreated"
                  @line-item-updated="handleLineItemUpdated"
                  @line-item-deleted="handleLineItemDeleted"
                  @dismiss-inline-create="showInlineForm = null"
                  @inline-form-cancel="showInlineForm = null"
                />
              </div>
            </div>
          </Transition>

          <div
            v-if="editingConfigId !== config.id && config._permissions?.delete"
            class="card-footer flex items-center justify-end"
          >
            <button
              type="button"
              class="table-action-btn table-action-btn--delete"
              title="Delete auction configuration"
              aria-label="Delete"
              @click="deleteAuctionConfigHandle(Number(config.id))"
            >
              <i class="pi pi-trash" />
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import useDeleteResource from "@/composables/useDeleteResource";
import { useToast } from "primevue/usetoast";

const props = defineProps({
  initialAuctionConfigs: { type: Array, required: true },
  initialLineItems: { type: Array, required: true },
  appId: { type: [Number, String], required: true },
});

const toast = useToast();
const auctionConfigs = ref([...props.initialAuctionConfigs]);
const allLineItems = ref([...props.initialLineItems]);

const configSearch = ref("");
const configAdTypeFilter = ref(null);

const configAdTypes = computed(() =>
  [...new Set(auctionConfigs.value.map((c) => c.adType))].sort(),
);

const filteredConfigs = computed(() => {
  const search = configSearch.value.trim().toLowerCase();
  return auctionConfigs.value.filter((c) => {
    if (configAdTypeFilter.value && c.adType !== configAdTypeFilter.value)
      return false;
    if (search && !c.name?.toLowerCase().includes(search)) return false;
    return true;
  });
});

const filteredConfigGroups = computed(() => {
  const byAdType = {};
  for (const config of filteredConfigs.value) {
    if (!byAdType[config.adType]) byAdType[config.adType] = [];
    byAdType[config.adType].push(config);
  }
  return Object.entries(byAdType)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([adType, items]) => ({ adType, items }));
});

function copyField(value) {
  navigator.clipboard.writeText(String(value));
}

const showInlineConfigForm = ref(false);
const cloneSourceConfig = ref(null);
const editingConfigId = ref(null);
const collapsedConfigIds = ref(new Set());

const deleteAuctionConfigHandle = useDeleteResource({
  path: "/v2/auction_configurations",
  hook: (deletedId) => {
    auctionConfigs.value = auctionConfigs.value.filter(
      (c) => Number(c.id) !== Number(deletedId),
    );
    if (editingConfigId.value === Number(deletedId)) {
      editingConfigId.value = null;
    }
    collapsedConfigIds.value = new Set(
      [...collapsedConfigIds.value].filter(
        (id) => Number(id) !== Number(deletedId),
      ),
    );
  },
  successDetail: "Auction configuration deleted.",
});

function toggleCollapse(configId) {
  const next = new Set(collapsedConfigIds.value);
  if (next.has(configId)) {
    next.delete(configId);
  } else {
    next.add(configId);
  }
  collapsedConfigIds.value = next;
}

function toggleConfigForm() {
  showInlineConfigForm.value = !showInlineConfigForm.value;
  cloneSourceConfig.value = null;
  editingConfigId.value = null;
}

function cloneAuctionConfig(config) {
  cloneSourceConfig.value = config;
  showInlineConfigForm.value = true;
  editingConfigId.value = null;
  showInlineForm.value = null;
  nextTick(() => {
    document
      .querySelector(".inline-auction-config-form")
      ?.scrollIntoView({ behavior: "smooth", block: "start" });
  });
}

function handleAuctionConfigCreated(newConfig) {
  auctionConfigs.value.push(newConfig);
  showInlineConfigForm.value = false;
  cloneSourceConfig.value = null;
}

function handleAuctionConfigUpdated(updatedConfig) {
  const idx = auctionConfigs.value.findIndex((c) => c.id === updatedConfig.id);
  if (idx !== -1) {
    auctionConfigs.value[idx] = updatedConfig;
  }
  editingConfigId.value = null;
}

function toggleEditingAuctionConfig(configId) {
  const next = editingConfigId.value === configId ? null : configId;
  editingConfigId.value = next;
  if (next !== null) {
    showInlineConfigForm.value = false;
    cloneSourceConfig.value = null;
    showInlineForm.value = null;
  }
}

const showInlineForm = ref(null);

function inlineFormKey(configId, networkKey, isBidding) {
  return `${configId}-${networkKey}-${isBidding ? "b" : "w"}`;
}

function toggleInlineForm(configId, networkKey, isBidding) {
  const key = inlineFormKey(configId, networkKey, isBidding);
  showInlineForm.value = showInlineForm.value === key ? null : key;
}

/** Keeps app-level line item list in sync when line items are created from the full auction config form (edit / new / clone). Inline network UI uses this list to render linked line items. */
function registerLineItemFromConfigForm(newItem) {
  const id = Number(newItem.id);
  if (Number.isNaN(id)) return;
  const alreadyLoaded = allLineItems.value.some((i) => Number(i.id) === id);
  if (!alreadyLoaded) {
    allLineItems.value.push(newItem);
  }
}

async function handleLineItemCreated(newItem, configId) {
  const id = Number(newItem.id);

  const configIdx = auctionConfigs.value.findIndex((c) => c.id === configId);
  if (configIdx === -1) return;

  const config = auctionConfigs.value[configIdx];
  const adUnitIds = (config.adUnitIds ?? []).map(Number);

  // Server may return an existing row (deduped) with the same id as a line item
  // already linked here, or the client may not see HTTP 200 vs 201 — avoid
  // duplicate rows in the list and duplicate ids in ad_unit_ids.
  if (adUnitIds.includes(id)) {
    toast.add({
      severity: "warn",
      summary: "Line item already linked",
      detail:
        "A line item with the same app, account, ad type, and settings already exists and is already part of this auction configuration.",
      life: 6000,
    });
    return;
  }

  const alreadyLoaded = allLineItems.value.some((i) => i.id === id);
  if (!alreadyLoaded) {
    allLineItems.value.push(newItem);
  }

  const updatedAdUnitIds = [...adUnitIds, id];

  try {
    const data = await $apiFetch(`/v2/auction_configurations/${configId}`, {
      method: "PATCH",
      body: { adUnitIds: updatedAdUnitIds },
    });
    auctionConfigs.value[configIdx] = data;
  } catch (err) {
    toast.add({
      severity: "error",
      summary: "Link failed",
      detail:
        err.data?.error?.message ??
        "Line item was created but could not be linked to the auction configuration. Please edit the configuration manually.",
    });
  }

  showInlineForm.value = null;
}

function handleLineItemUpdated(updatedItem) {
  const idx = allLineItems.value.findIndex((i) => i.id === updatedItem.id);
  if (idx !== -1) {
    allLineItems.value[idx] = updatedItem;
  }
}

async function handleLineItemDeleted(itemId) {
  const idNum = Number(itemId);
  allLineItems.value = allLineItems.value.filter((i) => Number(i.id) !== idNum);

  for (const config of auctionConfigs.value) {
    const unitIds = config.adUnitIds ?? [];
    if (!unitIds.some((id) => Number(id) === idNum)) continue;

    const updatedAdUnitIds = unitIds.filter((id) => Number(id) !== idNum);
    try {
      const data = await $apiFetch(`/v2/auction_configurations/${config.id}`, {
        method: "PATCH",
        body: { adUnitIds: updatedAdUnitIds },
      });
      const configIdx = auctionConfigs.value.findIndex(
        (c) => c.id === config.id,
      );
      if (configIdx !== -1) {
        auctionConfigs.value[configIdx] = data;
      }
    } catch (err) {
      toast.add({
        severity: "error",
        summary: "Unlink failed",
        detail:
          err.data?.error?.message ??
          "Line item was deleted but could not be unlinked from auction configuration. Please refresh the page.",
      });
    }
  }
}
</script>

<style scoped>
.collapse-enter-active,
.collapse-leave-active {
  transition:
    max-height 0.3s ease,
    opacity 0.25s ease;
  overflow: hidden;
  max-height: 2000px;
}

.collapse-enter-from,
.collapse-leave-to {
  max-height: 0;
  opacity: 0;
}
</style>
