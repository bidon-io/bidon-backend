<template>
  <div class="px-6 py-4">
    <p
      class="text-sm font-semibold uppercase tracking-wide mb-2"
      style="color: var(--bidon-muted)"
    >
      {{ isBidding ? "Bidding Networks" : "Waterfall Networks" }}
    </p>

    <div
      v-if="loading"
      class="flex items-center gap-2 text-sm py-2"
      style="color: var(--bidon-muted)"
    >
      <i class="pi pi-spin pi-spinner" />
      Loading…
    </div>

    <div v-else class="flex flex-col gap-2">
      <details
        v-for="network in allNetworks"
        :key="network.key"
        open
        class="group rounded-lg overflow-hidden"
        style="border: 1px solid var(--bidon-border-default)"
      >
        <summary
          class="flex items-center justify-between px-3 py-2 cursor-pointer list-none select-none transition-colors"
          style="background-color: var(--bidon-bg-card-header)"
        >
          <div class="flex items-center gap-2">
            <svg
              class="w-3.5 h-3.5 transition-transform group-open:rotate-90 shrink-0"
              style="color: var(--bidon-muted)"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
              aria-hidden="true"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M9 5l7 7-7 7"
              />
            </svg>
            <span
              class="text-sm font-medium"
              style="color: var(--bidon-text-primary)"
            >
              {{ network.label }}
            </span>
            <span
              :class="[
                'badge',
                enabledNetworkKeys.includes(network.key)
                  ? isBidding
                    ? 'badge-count-rtb'
                    : 'badge-count-cpm'
                  : '',
              ]"
              :style="
                !enabledNetworkKeys.includes(network.key)
                  ? 'background-color: var(--bidon-accent-subtle); color: var(--bidon-muted); font-size: 0.75rem; font-weight: 500; padding: 0.125rem 0.375rem; border-radius: 4px;'
                  : ''
              "
            >
              {{ networkLineItems(network.key).length }}
            </span>
            <!-- No demand profile warning -->
            <span
              v-if="!networkHasProfile(network.key)"
              class="network-no-profile-warning flex items-center gap-1 text-xs px-1.5 py-0.5 rounded font-medium"
              title="No demand source profile configured for this network"
            >
              <i class="pi pi-exclamation-triangle text-xs" />
              No profile
            </span>
          </div>
          <div class="flex items-center gap-2" @click.stop>
            <button
              class="text-xs px-2.5 py-1 rounded border font-medium transition-colors"
              :style="
                !networkHasProfile(network.key)
                  ? 'background-color: var(--bidon-accent-subtle); border-color: var(--bidon-border-default); color: var(--bidon-muted); cursor: not-allowed; opacity: 0.5;'
                  : enabledNetworkKeys.includes(network.key)
                    ? 'background-color: rgba(16,175,108,0.12); border-color: rgba(16,175,108,0.4); color: var(--bidon-primary);'
                    : 'background-color: var(--bidon-bg-card); border-color: var(--bidon-border-default); color: var(--bidon-muted);'
              "
              type="button"
              :disabled="!networkHasProfile(network.key)"
              @click="toggleNetworkEnabled(network.key)"
            >
              <i
                :class="[
                  'pi mr-1',
                  enabledNetworkKeys.includes(network.key)
                    ? 'pi-check-circle'
                    : 'pi-circle',
                ]"
              />
              {{
                enabledNetworkKeys.includes(network.key)
                  ? "Enabled"
                  : "Disabled"
              }}
            </button>
            <button
              :class="[
                'btn-sm',
                !networkHasProfile(network.key)
                  ? 'btn cursor-not-allowed opacity-40'
                  : showCreateFor === network.key
                    ? 'btn-cancel'
                    : 'btn-new',
              ]"
              type="button"
              :disabled="!networkHasProfile(network.key)"
              :title="
                !networkHasProfile(network.key)
                  ? 'Add a demand profile first'
                  : undefined
              "
              @click="toggleCreateForm(network.key)"
            >
              <i
                :class="[
                  'pi',
                  showCreateFor === network.key ? 'pi-times' : 'pi-plus',
                ]"
              />
              {{ showCreateFor === network.key ? "Cancel" : "New Line Item" }}
            </button>
          </div>
        </summary>

        <div
          class="divide-y"
          style="
            background-color: var(--bidon-bg-card);
            border-color: var(--bidon-border-default);
          "
        >
          <template
            v-for="item in networkLineItems(network.key)"
            :key="item.id"
          >
            <div
              class="flex min-w-0 items-center gap-2 px-4 py-2 text-sm sm:gap-3"
            >
              <Checkbox
                :model-value="selectedAdUnitIds"
                :value="item.id"
                @update:model-value="$emit('update:selectedAdUnitIds', $event)"
              />
              <span
                class="min-w-0 flex-1 truncate"
                style="color: var(--bidon-text-primary)"
              >
                {{ item.label }}
              </span>
              <span
                v-if="!isBidding"
                class="text-sm shrink-0 font-mono"
                style="color: var(--bidon-muted)"
              >
                ${{ item.pricefloor.toFixed(2) }}
              </span>
              <button
                type="button"
                :class="[
                  'btn-sm shrink-0',
                  editingItemId === item.id ? 'btn-cancel' : 'btn-edit',
                ]"
                :aria-label="
                  editingItemId === item.id
                    ? 'Cancel editing line item'
                    : 'Edit line item'
                "
                :title="editingItemId === item.id ? 'Cancel' : 'Edit'"
                @click.stop="toggleEditForm(item.id)"
              >
                <i
                  :class="[
                    'pi',
                    editingItemId === item.id ? 'pi-times' : 'pi-pencil',
                  ]"
                  aria-hidden="true"
                />
                <span class="hidden sm:inline">{{
                  editingItemId === item.id ? "Cancel" : "Edit"
                }}</span>
              </button>
              <button
                type="button"
                class="btn-sm btn-delete shrink-0"
                aria-label="Delete line item"
                title="Delete"
                @click.stop="deleteItem(item.id)"
              >
                <i class="pi pi-trash" aria-hidden="true" />
                <span class="hidden sm:inline">Delete</span>
              </button>
            </div>
            <InlineLineItemForm
              v-if="editingItemId === item.id"
              :app-id="appId"
              :ad-type="adType"
              :network-key="network.key"
              :is-bidding="isBidding"
              :initial-item="{
                id: item.id,
                humanName: item.label,
                accountId: item.accountId,
                bidFloor: item.pricefloor,
                extra: item.extra,
              }"
              @updated="onItemUpdated"
              @cancel="editingItemId = null"
            />
          </template>

          <div
            v-if="
              !networkLineItems(network.key).length &&
              showCreateFor !== network.key
            "
            class="px-4 py-2 text-sm text-gray-400"
          >
            No line items yet.
          </div>

          <InlineLineItemForm
            v-if="showCreateFor === network.key"
            :app-id="appId"
            :ad-type="adType"
            :network-key="network.key"
            :is-bidding="isBidding"
            @created="onItemCreated($event, network.key)"
            @cancel="showCreateFor = null"
          />
        </div>
      </details>
    </div>
  </div>
</template>

<script setup>
import { useConfirm } from "primevue/useconfirm";
import { useToast } from "primevue/usetoast";
import {
  AUCTION_NETWORKS,
  NETWORK_LABEL_BY_KEY,
  NETWORK_ACCOUNT_TYPE_BY_KEY,
} from "@/constants/Networks.js";

const props = defineProps({
  isBidding: { type: Boolean, required: true },
  adUnits: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  appId: { type: [Number, String], required: true },
  adType: { type: String, required: true },
  enabledNetworkKeys: { type: Array, default: () => [] },
  selectedAdUnitIds: { type: Array, default: () => [] },
  demandProfiles: { type: Array, default: () => [] },
});

const emit = defineEmits([
  "update:enabledNetworkKeys",
  "update:selectedAdUnitIds",
  "network-enabled",
  "network-disabled",
  "line-item-created",
  "line-item-updated",
  "line-item-deleted",
]);

const localProfiles = ref([...props.demandProfiles]);
watch(
  () => props.demandProfiles,
  (val) => {
    localProfiles.value = [...val];
  },
);

const profileAccountTypes = computed(
  () =>
    new Set(
      localProfiles.value
        .map((p) => p.accountType ?? p.account?.type)
        .filter(Boolean),
    ),
);

function networkHasProfile(networkKey) {
  return profileAccountTypes.value.has(NETWORK_ACCOUNT_TYPE_BY_KEY[networkKey]);
}

const allNetworks = computed(() =>
  AUCTION_NETWORKS.filter((n) => n.isBidding === props.isBidding).map((n) => ({
    key: n.key,
    label: NETWORK_LABEL_BY_KEY[n.key] ?? n.key,
  })),
);

function networkLineItems(networkKey) {
  return props.adUnits.filter((u) => u.networkKey === networkKey);
}

function toggleNetworkEnabled(networkKey) {
  if (!networkHasProfile(networkKey)) return;
  const isEnabled = props.enabledNetworkKeys.includes(networkKey);
  const keys = isEnabled
    ? props.enabledNetworkKeys.filter((k) => k !== networkKey)
    : [...props.enabledNetworkKeys, networkKey];
  emit("update:enabledNetworkKeys", keys);
  emit(isEnabled ? "network-disabled" : "network-enabled", networkKey);
}

const showCreateFor = ref(null);
const editingItemId = ref(null);

function toggleCreateForm(networkKey) {
  showCreateFor.value = showCreateFor.value === networkKey ? null : networkKey;
  editingItemId.value = null;
}

function toggleEditForm(itemId) {
  editingItemId.value = editingItemId.value === itemId ? null : itemId;
  showCreateFor.value = null;
}

function onItemCreated(item, networkKey) {
  showCreateFor.value = null;
  emit("line-item-created", item, networkKey);
}

function onItemUpdated(item) {
  editingItemId.value = null;
  emit("line-item-updated", item);
}

const confirm = useConfirm();
const toast = useToast();

function deleteItem(itemId) {
  confirm.require({
    message: "Delete this line item?",
    header: "Confirm Delete",
    icon: "pi pi-exclamation-triangle",
    acceptClass: "p-button-danger",
    accept: async () => {
      try {
        await $apiFetch(`/line_items/${itemId}`, { method: "DELETE" });
        toast.add({
          severity: "success",
          summary: "Success",
          detail: "Line item deleted.",
          life: 3000,
        });
        emit("line-item-deleted", itemId);
      } catch (err) {
        toast.add({
          severity: "error",
          summary: "Delete failed",
          detail: err.data?.error?.message ?? "Could not delete the line item.",
        });
      }
    },
  });
}
</script>
