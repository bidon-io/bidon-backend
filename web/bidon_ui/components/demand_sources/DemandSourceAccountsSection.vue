<template>
  <div class="mt-10 mb-10">
    <div class="flex items-center justify-between mb-4">
      <div>
        <h3
          class="text-base font-semibold"
          style="color: var(--bidon-text-primary)"
        >
          Accounts
        </h3>
        <p class="text-sm mt-0.5" style="color: var(--bidon-muted)">
          Credentials and API keys for this demand source.
        </p>
      </div>
      <button
        v-if="canCreate"
        :class="['btn-sm', showCreateForm ? 'btn-cancel' : 'btn-new']"
        type="button"
        @click="toggleCreateForm"
      >
        <i :class="['pi', showCreateForm ? 'pi-times' : 'pi-plus']" />
        {{ showCreateForm ? "Cancel" : "New Account" }}
      </button>
    </div>

    <InlineDemandSourceAccountForm
      v-if="showCreateForm"
      :demand-source-id="dsId"
      :demand-source-api-key="demandSourceApiKey"
      class="mb-4 rounded-lg overflow-hidden"
      style="border: 1px solid rgba(16, 175, 108, 0.2)"
      @created="onCreated"
      @cancel="showCreateForm = false"
    />

    <div
      v-if="!visibleAccounts.length && !showCreateForm"
      class="text-sm py-6 px-4 rounded-lg"
      style="
        color: var(--bidon-muted);
        background: var(--bidon-bg-card);
        border: 1px solid var(--bidon-border-default);
      "
    >
      No accounts yet.
    </div>

    <div
      v-if="visibleAccounts.length"
      class="grid grid-cols-1 gap-3 md:grid-cols-2"
    >
      <div
        v-for="item in visibleAccounts"
        :key="item.id"
        class="card flex flex-col rounded-lg overflow-hidden"
        style="
          background-color: var(--bidon-bg-card);
          border: 1px solid var(--bidon-border-default);
        "
      >
        <div class="card-header gap-3">
          <div class="min-w-0">
            <span
              class="font-semibold text-sm block truncate"
              style="color: var(--bidon-text-primary)"
            >
              {{ item.label || `Account #${item.id}` }}
            </span>
          </div>
          <span v-if="item.type" class="badge badge-platform shrink-0">
            {{ String(item.type).split("::")[1] }}
          </span>
          <button
            v-if="item._permissions?.update"
            :class="[
              'btn-sm',
              editingId === Number(item.id) ? 'btn-cancel' : 'btn-edit',
            ]"
            type="button"
            @click="toggleEdit(Number(item.id))"
          >
            <i
              :class="[
                'pi',
                editingId === Number(item.id) ? 'pi-times' : 'pi-pencil',
              ]"
            />
            {{ editingId === Number(item.id) ? "Cancel" : "Edit" }}
          </button>
        </div>

        <InlineDemandSourceAccountForm
          v-if="editingId === Number(item.id)"
          :demand-source-id="dsId"
          :demand-source-api-key="demandSourceApiKey"
          :initial-account="item"
          @updated="onUpdated"
          @cancel="editingId = null"
        />

        <div v-else class="card-body flex-1 py-3 space-y-1.5">
          <div v-if="item.user?.email" class="flex items-center gap-2">
            <i
              class="pi pi-user text-xs shrink-0 w-3.5"
              style="color: var(--bidon-muted)"
            />
            <NuxtLink
              :to="`/users/${item.user.id}`"
              class="text-xs truncate table-resource-link"
            >
              {{ item.user.email }}
            </NuxtLink>
          </div>
        </div>

        <div
          v-if="editingId !== Number(item.id) && item._permissions?.delete"
          class="card-footer flex items-center justify-end"
        >
          <button
            type="button"
            class="table-action-btn table-action-btn--delete"
            title="Delete"
            aria-label="Delete"
            @click="deleteHandle(Number(item.id))"
          >
            <i class="pi pi-trash" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import useDeleteResource from "@/composables/useDeleteResource";

const props = defineProps<{
  accounts: Record<string, unknown>[];
  demandSourceId: string | string[];
  demandSourceApiKey: string;
}>();

const emit = defineEmits<{
  refresh: [];
}>();

const resources = useResources();
const dsId = computed(() =>
  Array.isArray(props.demandSourceId)
    ? props.demandSourceId[0]
    : props.demandSourceId,
);

const visibleAccounts = computed(() =>
  props.accounts.filter(
    (account) => Number(account.demandSourceId) === Number(dsId.value),
  ),
);

const canCreate = computed(
  () => resources.state?.demandSourceAccount?.permissions?.create === true,
);

const showCreateForm = ref(false);
const editingId = ref<number | null>(null);

function toggleCreateForm() {
  showCreateForm.value = !showCreateForm.value;
  if (showCreateForm.value) {
    editingId.value = null;
  }
}

function toggleEdit(id: number) {
  editingId.value = editingId.value === id ? null : id;
  if (editingId.value != null) {
    showCreateForm.value = false;
  }
}

function onCreated() {
  showCreateForm.value = false;
  emit("refresh");
}

function onUpdated() {
  editingId.value = null;
  emit("refresh");
}

const deleteHandle = useDeleteResource({
  path: "/demand_source_accounts",
  hook: () => emit("refresh"),
  successDetail: "Account deleted.",
});
</script>
