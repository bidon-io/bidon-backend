<template>
  <div class="card mb-8">
    <div class="card-header">
      <div class="flex items-center gap-2 flex-wrap">
        <span class="font-semibold text-gray-800 text-lg">
          {{ resource.humanName }}
        </span>
        <span class="badge badge-platform">{{ resource.platformId }}</span>
        <span class="text-sm text-gray-400">{{ resource.packageName }}</span>
      </div>
      <div class="flex gap-2 shrink-0">
        <NuxtLink
          v-if="resource._permissions?.update"
          :to="`/apps/${id}/edit`"
          class="btn-edit btn-sm"
        >
          <i class="pi pi-pencil" /> Edit
        </NuxtLink>
        <DestroyButton
          v-if="resource._permissions?.delete"
          :id="id"
          :path="resourcesPath"
        />
      </div>
    </div>

    <div class="divide-y divide-gray-100">
      <div
        v-for="field in OVERVIEW_FIELDS"
        :key="field.key ?? field.label"
        class="flex items-start gap-4 px-6 py-3"
      >
        <span class="text-sm text-gray-400 w-44 shrink-0 pt-px">
          {{ field.label }}
        </span>
        <span
          class="text-sm text-gray-700 font-mono break-all flex items-center gap-1.5 min-w-0"
        >
          {{ field.value ? field.value(resource) : resource[field.key] }}
          <button
            v-if="
              field.copyable &&
              (field.value ? field.value(resource) : resource[field.key])
            "
            :aria-label="`Copy ${field.label}`"
            class="text-gray-300 hover:text-blue-400 transition-colors shrink-0"
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
  </div>
</template>

<script setup>
defineProps({
  resource: { type: Object, required: true },
  id: { type: [Number, String], required: true },
  resourcesPath: { type: String, required: true },
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
