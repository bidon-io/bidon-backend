<template>
  <div class="px-6 py-4">
    <p class="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-2">
      {{ isBidding ? "Bidding Networks" : "Waterfall Networks" }}
    </p>
    <div v-if="groups.length" class="flex flex-col gap-2">
      <details
        v-for="group in groups"
        :key="group.key"
        open
        class="group border border-gray-100 rounded-lg overflow-hidden"
      >
        <summary
          class="flex items-center justify-between px-3 py-2 bg-gray-50 cursor-pointer list-none select-none hover:bg-gray-100 transition-colors"
        >
          <div class="flex items-center gap-2">
            <svg
              class="w-3.5 h-3.5 text-gray-400 transition-transform group-open:rotate-90"
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
            <span class="text-sm font-medium text-gray-700">{{
              group.label
            }}</span>
            <span
              :class="[
                'badge',
                isBidding ? 'badge-count-rtb' : 'badge-count-cpm',
              ]"
            >
              {{ group.items.length }}
            </span>
          </div>
          <button
            class="btn-new btn-sm"
            @click.stop="
              $emit('toggleInlineForm', config.id, group.key, isBidding)
            "
          >
            <i class="pi pi-plus" /> New Line Item
          </button>
        </summary>
        <div class="divide-y divide-gray-50 bg-white">
          <template v-for="item in group.items" :key="item.id">
            <div class="flex items-center gap-3 px-4 py-2 text-sm">
              <span class="flex-1 text-gray-700 truncate">
                {{ item.humanName }}
              </span>
              <span v-if="!isBidding" class="text-sm text-gray-400 shrink-0">
                ${{ item.bidFloor }}
              </span>
              <button
                :class="[
                  'btn-sm shrink-0',
                  editingItemId === item.id ? 'btn-cancel' : 'btn-edit',
                ]"
                @click.stop="toggleEditForm(item.id)"
              >
                <i
                  :class="[
                    'pi',
                    editingItemId === item.id ? 'pi-times' : 'pi-pencil',
                  ]"
                />
                {{ editingItemId === item.id ? "Cancel" : "Edit" }}
              </button>
              <button
                class="btn-sm btn-delete shrink-0"
                @click.stop="deleteItem(item.id)"
              >
                <i class="pi pi-trash" />
                Delete
              </button>
            </div>
            <InlineLineItemForm
              v-if="editingItemId === item.id"
              :app-id="appId"
              :ad-type="config.adType"
              :network-key="group.key"
              :is-bidding="isBidding"
              :initial-item="item"
              @updated="(updatedItem) => onItemUpdated(updatedItem)"
              @cancel="editingItemId = null"
            />
          </template>
          <div
            v-if="
              !group.items.length &&
              showInlineForm !== inlineFormKey(config.id, group.key, isBidding)
            "
            class="px-4 py-2 text-sm text-gray-400"
          >
            No line items linked yet.
          </div>
          <InlineLineItemForm
            v-if="
              showInlineForm === inlineFormKey(config.id, group.key, isBidding)
            "
            :app-id="appId"
            :ad-type="config.adType"
            :network-key="group.key"
            :is-bidding="isBidding"
            @created="(item) => $emit('lineItemCreated', item, config.id)"
            @cancel="$emit('inlineFormCancel')"
          />
        </div>
      </details>
    </div>
    <p v-else class="text-sm text-gray-400">None configured</p>
  </div>
</template>

<script setup>
import { useConfirm } from "primevue/useconfirm";
import { useToast } from "primevue/usetoast";
import { NETWORK_LABEL_BY_KEY } from "@/constants/Networks.js";

const props = defineProps({
  config: { type: Object, required: true },
  isBidding: { type: Boolean, required: true },
  allLineItems: { type: Array, required: true },
  showInlineForm: { type: String, default: null },
  appId: { type: [Number, String], required: true },
});

const emit = defineEmits([
  "toggleInlineForm",
  "lineItemCreated",
  "lineItemUpdated",
  "lineItemDeleted",
  "inlineFormCancel",
  "dismissInlineCreate",
]);

const editingItemId = ref(null);

// Only dismiss inline *edit* when opening inline *create* — not when create
// finishes (showInlineForm becomes null), so a stray edit is not cleared.
watch(
  () => props.showInlineForm,
  (val) => {
    if (val) {
      editingItemId.value = null;
    }
  },
);

function inlineFormKey(configId, networkKey, isBidding) {
  return `${configId}-${networkKey}-${isBidding ? "b" : "w"}`;
}

function toggleEditForm(itemId) {
  const next = editingItemId.value === itemId ? null : itemId;
  if (next !== null) {
    emit("dismissInlineCreate");
  }
  editingItemId.value = next;
}

function onItemUpdated(updatedItem) {
  editingItemId.value = null;
  emit("lineItemUpdated", updatedItem);
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
        emit("lineItemDeleted", itemId);
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

const groups = computed(() => {
  const enabledKeys = props.isBidding
    ? (props.config.bidding ?? [])
    : (props.config.demands ?? []);
  if (!enabledKeys.length) return [];

  const configIds = new Set(props.config.adUnitIds ?? []);
  const items = props.allLineItems.filter(
    (item) =>
      configIds.has(Number(item.id)) && item.isBidding === props.isBidding,
  );

  return enabledKeys.map((key) => ({
    key,
    label: NETWORK_LABEL_BY_KEY[key] ?? key,
    items: items.filter(
      (item) => item.account?.type?.split("::")?.[1]?.toLowerCase() === key,
    ),
  }));
});
</script>
