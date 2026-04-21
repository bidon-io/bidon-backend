<template>
  <ResourcesGrid
    resources-path="/api_keys"
    :filters="filters"
    :sort-fn="sortApiKeys"
    empty-message="No API credentials yet."
    delete-success-detail="API credential deleted."
  >
    <template v-if="resource?.permissions?.create" #actions>
      <button
        type="button"
        class="btn-new"
        :disabled="buttonDisabled"
        @click="createApiToken"
      >
        <i class="pi pi-plus text-xs" /> Generate Token
      </button>
    </template>
    <template #card="{ item, deleteHandle }">
      <div class="card flex flex-col">
        <div class="card-header">
          <div class="min-w-0">
            <NuxtLink
              :to="`/api_keys/${item.id}`"
              class="font-semibold text-sm block truncate table-resource-link font-mono"
            >
              {{ item.id }}
            </NuxtLink>
          </div>
        </div>

        <div class="card-body flex-1 py-3 space-y-1.5">
          <div class="flex items-center gap-2">
            <i
              class="pi pi-clock text-xs shrink-0 w-3.5"
              style="color: var(--bidon-muted)"
            />
            <span class="text-xs" style="color: var(--bidon-text-secondary)">
              {{ lastAccessedLabel(item.lastAccessedAt) }}
            </span>
          </div>
        </div>

        <div class="card-footer flex items-center justify-between">
          <div class="flex items-center gap-2">
            <NuxtLink :to="`/api_keys/${item.id}`" class="btn-view btn-sm">
              <i class="pi pi-eye text-xs" /> View
            </NuxtLink>
            <button
              type="button"
              class="btn-view btn-sm"
              title="Copy API key"
              :disabled="copyingId === item.id"
              @click="copyKey(item.id)"
            >
              <i
                :class="copiedId === item.id ? 'pi pi-check text-xs' : 'pi pi-copy text-xs'"
              />
              {{ copiedId === item.id ? "Copied" : "Copy Key" }}
            </button>
          </div>
          <button
            v-if="item._permissions?.delete"
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
import { $apiFetch } from "~/utils/$apiFetch";
import { useToast } from "primevue/usetoast";
import type { FilterConfig } from "~/components/resources/ResourcesGrid.vue";

const toast = useToast();

const resources = useResources();
const resource = computed(() => resources.state?.apiKey ?? {});

const filters: FilterConfig[] = [
  {
    key: "search",
    label: "Credential ID",
    type: "text",
    placeholder: "Search…",
    match: (item, value) =>
      String(item.id ?? "")
        .toLowerCase()
        .includes(value.trim().toLowerCase()),
  },
];

function lastAccessedLabel(value: unknown): string {
  if (value == null || value === "") {
    return "Never used";
  }
  const d = new Date(String(value));
  if (Number.isNaN(d.getTime())) {
    return "Never used";
  }
  return d.toLocaleString();
}

function sortApiKeys(
  a: Record<string, unknown>,
  b: Record<string, unknown>,
): number {
  const ta = a.lastAccessedAt
    ? new Date(String(a.lastAccessedAt)).getTime()
    : 0;
  const tb = b.lastAccessedAt
    ? new Date(String(b.lastAccessedAt)).getTime()
    : 0;
  if (tb !== ta) {
    return tb - ta;
  }
  return String(a.id ?? "").localeCompare(String(b.id ?? ""));
}

const copyingId = ref<string | null>(null);
const copiedId = ref<string | null>(null);

const copyKey = async (id: string) => {
  copyingId.value = id;
  try {
    const key = await $apiFetch(`/api_keys/${id}`);
    await navigator.clipboard.writeText(key.value ?? "");
    copiedId.value = id;
    setTimeout(() => {
      if (copiedId.value === id) copiedId.value = null;
    }, 2000);
  } catch (error) {
    console.error(error);
    toast.add({ severity: "error", summary: "Failed to copy API key" });
  } finally {
    copyingId.value = null;
  }
};

const buttonDisabled = ref(false);
const createApiToken = async () => {
  buttonDisabled.value = true;

  const apiKey = await $apiFetch("/api_keys", {
    method: "POST",
  }).catch((error) => {
    console.error(error);
    const status = error?.response?.status ?? error?.status;
    const statusText = error?.response?.statusText ?? error?.statusText;
    const message =
      error?.response?.data?.error?.message ??
      error?.data?.error?.message ??
      "";
    toast.add({
      severity: "error",
      summary: `${status ?? ""} ${statusText ?? ""}`.trim(),
      detail: message,
    });
  });

  buttonDisabled.value = false;

  if (!apiKey) {
    return;
  }

  navigateTo(`/api_keys/${apiKey.id}`);
};
</script>
