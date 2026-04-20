<template>
  <ResourcesGrid
    resources-path="/users"
    new-path="/users/new"
    new-label="New User"
    :filters="filters"
    :sort-fn="sortUsersByEmail"
    card-grid-class="grid-cols-1 md:grid-cols-2"
    empty-message="No users yet."
    delete-success-detail="User deleted."
  >
    <template #card="{ item, deleteHandle }">
      <div class="card flex flex-col">
        <div class="card-header gap-3">
          <div class="min-w-0">
            <NuxtLink
              :to="`/users/${item.id}`"
              class="font-semibold text-sm block break-all table-resource-link"
              >{{ item.email }}</NuxtLink
            >
          </div>
          <div class="flex items-center gap-2 shrink-0">
            <span
              class="badge shrink-0"
              :class="item.isAdmin === true ? 'badge-active' : 'badge-disabled'"
            >
              {{ item.isAdmin === true ? "Admin" : "User" }}
            </span>
            <button
              v-if="item._permissions?.update"
              type="button"
              :class="['btn-sm', isEditing(item) ? 'btn-cancel' : 'btn-edit']"
              @click="toggleInlineEdit(item)"
            >
              <i :class="['pi text-xs', isEditing(item) ? 'pi-times' : 'pi-pencil']" />
              {{ isEditing(item) ? "Cancel" : "Edit" }}
            </button>
          </div>
        </div>

        <InlineUserForm
          v-if="isEditing(item)"
          :user="item"
          @updated="applyInlineUpdate(item, $event)"
          @cancel="cancelInlineEdit"
        />

        <div v-else class="card-body flex-1 py-3 space-y-1.5">
          <div v-if="item.email" class="flex items-start gap-2">
            <i
              class="pi pi-envelope text-xs shrink-0 w-3.5 mt-0.5"
              style="color: var(--bidon-muted)"
            />
            <span
              class="text-xs break-all leading-snug"
              style="color: var(--bidon-text-secondary)"
              >{{ item.email }}</span
            >
          </div>
          <div v-if="item.publicUid" class="flex items-center gap-2">
            <i
              class="pi pi-hashtag text-xs shrink-0 w-3.5"
              style="color: var(--bidon-muted)"
            />
            <span
              class="text-xs font-mono truncate"
              style="color: var(--bidon-text-secondary)"
              >{{ item.publicUid }}</span
            >
          </div>
        </div>

        <div
          v-if="!isEditing(item) && item._permissions?.delete"
          class="card-footer flex items-center justify-end"
        >
          <button
            type="button"
            class="table-action-btn table-action-btn--delete"
            title="Delete"
            aria-label="Delete"
            @click="deleteHandle(String(item.id))"
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

type UserCardItem = Record<string, unknown> & {
  id: number | string;
  email?: string;
  isAdmin?: boolean;
  publicUid?: string;
};

const filters: FilterConfig[] = [
  {
    key: "search",
    label: "Email",
    type: "text",
    placeholder: "Search…",
    match: (item, value) =>
      String(item.email ?? "")
        .toLowerCase()
        .includes(value.trim().toLowerCase()),
  },
  {
    key: "role",
    label: "Role",
    type: "select",
    placeholder: "All",
    options: [
      { label: "Admin", value: "true" },
      { label: "User", value: "false" },
    ],
    match: (item, value) => String(item.isAdmin === true) === value,
  },
];

function sortUsersByEmail(
  a: Record<string, unknown>,
  b: Record<string, unknown>,
): number {
  return String(a.email ?? "").localeCompare(String(b.email ?? ""), undefined, {
    sensitivity: "base",
  });
}

const editingId = ref<string | null>(null);

function itemId(item: UserCardItem): string {
  return String(item.id);
}

function isEditing(item: UserCardItem): boolean {
  return editingId.value === itemId(item);
}

function toggleInlineEdit(item: UserCardItem): void {
  const id = itemId(item);
  editingId.value = editingId.value === id ? null : id;
}

function cancelInlineEdit(): void {
  editingId.value = null;
}

function applyInlineUpdate(item: UserCardItem, updated: Record<string, unknown>): void {
  item.email = String(updated.email ?? item.email ?? "");
  item.isAdmin = Boolean(updated.isAdmin ?? item.isAdmin);
  cancelInlineEdit();
}
</script>
