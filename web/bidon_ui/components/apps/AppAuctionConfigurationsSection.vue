<template>
  <div class="mt-10 mb-10">
    <div class="flex items-center justify-between mb-4">
      <div>
        <h3 class="text-base font-semibold text-gray-700">
          Auction Configurations
        </h3>
        <p class="text-sm text-gray-400 mt-0.5">
          Per ad-type auction settings with CPM and bidding networks.
        </p>
      </div>
      <button
        :class="['btn-sm', showInlineConfigForm ? 'btn-cancel' : 'btn-new']"
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
    />

    <!-- Filters -->
    <div
      v-if="auctionConfigs.length"
      class="flex flex-wrap items-center gap-3 mb-4"
    >
      <div class="relative">
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
          class="pl-8 pr-3 py-1.5 text-sm border border-gray-200 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-blue-300 w-52"
        />
      </div>
      <div class="flex flex-wrap gap-1.5">
        <button
          :class="[
            'text-xs px-3 py-1 rounded-md border font-medium transition-colors',
            configAdTypeFilter === null
              ? 'bg-violet-600 border-violet-600 text-white'
              : 'bg-white border-gray-200 text-gray-600 hover:border-violet-300 hover:text-violet-700',
          ]"
          @click="configAdTypeFilter = null"
        >
          All
        </button>
        <button
          v-for="adTypeOption in configAdTypes"
          :key="adTypeOption"
          :class="[
            'text-xs px-3 py-1 rounded-md border font-medium capitalize transition-colors',
            configAdTypeFilter === adTypeOption
              ? 'bg-violet-600 border-violet-600 text-white'
              : 'bg-white border-gray-200 text-gray-600 hover:border-violet-300 hover:text-violet-700',
          ]"
          @click="configAdTypeFilter = adTypeOption"
        >
          {{ adTypeOption }}
        </button>
      </div>
    </div>

    <p v-if="!auctionConfigs.length" class="text-sm text-gray-400 py-4">
      No auction configurations for this app yet.
    </p>
    <p
      v-else-if="!filteredConfigGroups.length"
      class="text-sm text-gray-400 py-4"
    >
      No configurations match the current filters.
    </p>

    <div v-for="group in filteredConfigGroups" :key="group.adType" class="mb-6">
      <div class="flex items-center gap-2 mb-3">
        <span class="badge badge-ad-type">{{ group.adType }}</span>
        <span class="text-sm text-gray-400">
          {{ group.items.length }} config{{
            group.items.length !== 1 ? "s" : ""
          }}
        </span>
      </div>

      <div class="grid grid-cols-1 gap-4">
        <div
          v-for="config in group.items"
          :key="config.id"
          class="border border-gray-200 rounded-lg bg-white overflow-hidden flex flex-col"
        >
          <div
            class="card-header cursor-pointer select-none"
            @click="toggleCollapse(config.id)"
          >
            <div class="flex flex-col gap-1 min-w-0">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-semibold text-gray-800">
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
                  class="flex items-center gap-1 text-xs text-gray-400 font-mono"
                >
                  <span class="text-gray-300">floor</span>
                  ${{ config.pricefloor }}
                </span>
                <span
                  class="flex items-center gap-1 text-xs text-gray-400 font-mono"
                >
                  <span class="text-gray-300">timeout</span>
                  {{ config.timeout }}ms
                </span>
                <span
                  v-if="config.auctionKey"
                  class="flex items-center gap-1 text-xs text-gray-400 font-mono"
                >
                  <span class="text-gray-300">key</span>
                  {{ config.auctionKey }}
                  <button
                    :aria-label="`Copy auction key ${config.auctionKey}`"
                    class="text-gray-300 hover:text-blue-400 transition-colors"
                    @click.stop="copyField(config.auctionKey)"
                  >
                    <i class="pi pi-copy" aria-hidden="true" />
                  </button>
                </span>
                <span
                  v-if="config.publicUid"
                  class="flex items-center gap-1 text-xs text-gray-400 font-mono"
                >
                  <span class="text-gray-300">uid</span>
                  {{ config.publicUid }}
                  <button
                    :aria-label="`Copy public UID ${config.publicUid}`"
                    class="text-gray-300 hover:text-blue-400 transition-colors"
                    @click.stop="copyField(config.publicUid)"
                  >
                    <i class="pi pi-copy" aria-hidden="true" />
                  </button>
                </span>
              </div>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <div class="flex gap-2" @click.stop>
                <button
                  class="btn-new btn-sm"
                  @click="cloneAuctionConfig(config)"
                >
                  <i class="pi pi-clone" /> Clone
                </button>
                <button
                  :class="[
                    'btn-sm',
                    editingConfigId === config.id ? 'btn-cancel' : 'btn-edit',
                  ]"
                  @click="
                    editingConfigId =
                      editingConfigId === config.id ? null : config.id
                  "
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
                class="pi pi-chevron-down text-gray-400 transition-transform duration-300"
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
              />
              <div v-else class="divide-y divide-gray-100">
                <AuctionConfigNetworkSection
                  :config="config"
                  :is-bidding="true"
                  :all-line-items="allLineItems"
                  :show-inline-form="showInlineForm"
                  :app-id="appId"
                  @toggle-inline-form="toggleInlineForm"
                  @line-item-created="handleLineItemCreated"
                  @line-item-updated="handleLineItemUpdated"
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
                  @inline-form-cancel="showInlineForm = null"
                />
              </div>
            </div>
          </Transition>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
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

const showInlineForm = ref(null);

function inlineFormKey(configId, networkKey, isBidding) {
  return `${configId}-${networkKey}-${isBidding ? "b" : "w"}`;
}

function toggleInlineForm(configId, networkKey, isBidding) {
  const key = inlineFormKey(configId, networkKey, isBidding);
  showInlineForm.value = showInlineForm.value === key ? null : key;
}

async function handleLineItemCreated(newItem, configId) {
  allLineItems.value.push(newItem);

  const configIdx = auctionConfigs.value.findIndex((c) => c.id === configId);
  if (configIdx === -1) return;

  const config = auctionConfigs.value[configIdx];
  const updatedAdUnitIds = [...(config.adUnitIds ?? []), Number(newItem.id)];

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
    auctionConfigs.value[configIdx] = {
      ...config,
      adUnitIds: updatedAdUnitIds,
    };
  }

  showInlineForm.value = null;
}

function handleLineItemUpdated(updatedItem) {
  const idx = allLineItems.value.findIndex((i) => i.id === updatedItem.id);
  if (idx !== -1) {
    allLineItems.value[idx] = updatedItem;
  }
}
</script>

<style scoped>
.collapse-enter-active,
.collapse-leave-active {
  transition: max-height 0.3s ease, opacity 0.25s ease;
  overflow: hidden;
  max-height: 2000px;
}

.collapse-enter-from,
.collapse-leave-to {
  max-height: 0;
  opacity: 0;
}
</style>
