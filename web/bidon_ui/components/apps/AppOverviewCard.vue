<template>
  <div class="card mb-8">
    <div class="card-header">
      <div class="flex items-center gap-2 flex-wrap">
        <span
          class="font-semibold text-lg"
          style="color: var(--bidon-text-primary)"
        >
          {{ mergedResource.humanName }}
        </span>
        <span class="badge badge-platform">{{
          mergedResource.platformId
        }}</span>
        <span class="text-sm" style="color: var(--bidon-muted)">{{
          mergedResource.packageName
        }}</span>
      </div>
      <div class="flex gap-2 shrink-0">
        <NuxtLink
          v-if="mergedResource._permissions?.update"
          :to="`/apps/${id}/edit`"
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
          {{
            field.value
              ? field.value(mergedResource)
              : mergedResource[field.key]
          }}
          <button
            v-if="
              field.copyable &&
              (field.value
                ? field.value(mergedResource)
                : mergedResource[field.key])
            "
            :aria-label="`Copy ${field.label}`"
            class="table-action-btn shrink-0"
            @click="
              copyField(
                field.value
                  ? field.value(mergedResource)
                  : mergedResource[field.key],
              )
            "
          >
            <i class="pi pi-copy" aria-hidden="true" />
          </button>
        </span>
      </div>
    </div>

    <div
      v-if="mergedResource._permissions?.delete"
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
import { mergeResourcePermissions } from "@/utils/mergeResourcePermissions";

const props = defineProps({
  resource: { type: Object, required: true },
  id: { type: [Number, String], required: true },
  resourcesPath: { type: String, required: true },
});

const mergedResource = ref(mergeResourcePermissions(undefined, props.resource));

watch(
  () => [props.id, props.resource],
  ([id, next], prevTuple) => {
    const prevId = prevTuple?.[0];
    if (prevTuple !== undefined && id !== prevId) {
      mergedResource.value = mergeResourcePermissions(undefined, next);
      return;
    }
    mergedResource.value = mergeResourcePermissions(mergedResource.value, next);
  },
);

const deleteHandle = useDeleteResource({
  path: props.resourcesPath,
  hook: async () => await navigateTo(props.resourcesPath),
});

const OVERVIEW_FIELDS = [
  { label: "Package Name", key: "packageName", copyable: true },
  { label: "Platform", key: "platformId" },
  { label: "Owner", value: (r) => r.user?.email },
  { label: "Public UID", key: "publicUid", copyable: true },
  { label: "App Key", key: "appKey", copyable: true },
  { label: "Store ID", key: "storeId" },
  { label: "Store URL", key: "storeUrl" },
  { label: "Categories", value: (r) => r.categories?.join(", ") },
  { label: "Blocked Advertiser Domains", key: "badv" },
  { label: "Blocked Categories", key: "bcat" },
  { label: "Blocked Apps", key: "bapp" },
];

function copyField(value) {
  navigator.clipboard.writeText(String(value));
}
</script>
