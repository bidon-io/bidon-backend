<template>
  <div class="card mb-8">
    <div class="card-header">
      <div class="flex items-center gap-2 flex-wrap min-w-0">
        <span
          class="font-semibold text-lg break-all"
          style="color: var(--bidon-text-primary)"
        >
          {{ resource.email }}
        </span>
        <span
          class="badge shrink-0"
          :class="resource.isAdmin === true ? 'badge-active' : 'badge-disabled'"
        >
          {{ resource.isAdmin === true ? "Admin" : "User" }}
        </span>
        <span
          v-if="resource.publicUid"
          class="text-sm font-mono truncate max-w-full sm:max-w-md"
          style="color: var(--bidon-muted)"
        >
          {{ resource.publicUid }}
        </span>
      </div>
      <div class="flex gap-2 shrink-0">
        <NuxtLink
          v-if="resource._permissions?.update"
          :to="`/users/${id}/edit`"
          class="btn-edit btn-sm"
        >
          <i class="pi pi-pencil" /> Edit
        </NuxtLink>
      </div>
    </div>

    <div class="divide-y" style="border-color: var(--bidon-border-default)">
      <div
        v-for="field in OVERVIEW_FIELDS"
        :key="field.key ?? field.label"
        class="flex items-start gap-4 px-6 py-3"
      >
        <span
          class="text-sm w-44 shrink-0 pt-px"
          style="color: var(--bidon-muted)"
        >
          {{ field.label }}
        </span>
        <span
          class="text-sm font-mono break-all flex items-center gap-1.5 min-w-0"
          style="color: var(--bidon-text-primary)"
        >
          {{ field.value ? field.value(resource) : resource[field.key] }}
          <button
            v-if="
              field.copyable &&
              (field.value ? field.value(resource) : resource[field.key])
            "
            :aria-label="`Copy ${field.label}`"
            class="table-action-btn shrink-0"
            type="button"
            @click="
              copyField(
                field.value ? field.value(resource) : resource[field.key],
              )
            "
          >
            <i class="pi pi-copy" aria-hidden="true" />
          </button>
        </span>
      </div>
    </div>

    <div
      v-if="resource._permissions?.delete"
      class="card-footer flex items-center justify-end"
    >
      <button
        type="button"
        class="table-action-btn table-action-btn--delete"
        title="Delete"
        aria-label="Delete"
        @click="deleteHandle(String(id))"
      >
        <i class="pi pi-trash" />
      </button>
    </div>
  </div>
</template>

<script setup>
import useDeleteResource from "@/composables/useDeleteResource";

const props = defineProps({
  resource: { type: Object, required: true },
  id: { type: [Number, String], required: true },
  resourcesPath: { type: String, required: true },
});

const deleteHandle = useDeleteResource({
  path: props.resourcesPath,
  hook: async () => await navigateTo(props.resourcesPath),
});

const OVERVIEW_FIELDS = [
  { label: "Public UID", key: "publicUid", copyable: true },
  { label: "Email", key: "email", copyable: true },
  {
    label: "Is Admin",
    value: (r) => (r.isAdmin === true ? "Yes" : "No"),
  },
];

function copyField(value) {
  navigator.clipboard.writeText(String(value));
}
</script>
