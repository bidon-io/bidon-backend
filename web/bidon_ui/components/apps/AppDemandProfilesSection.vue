<template>
  <div class="mt-10">
    <div class="flex items-center justify-between mb-4">
      <div>
        <h3 class="text-base font-semibold text-gray-700">
          Demand Source Profiles
        </h3>
        <p class="text-sm text-gray-400 mt-0.5">
          App-level configuration for each connected demand source.
        </p>
      </div>
      <button
        :class="['btn-sm', showCreateForm ? 'btn-cancel' : 'btn-new']"
        @click="
          showCreateForm = !showCreateForm;
          if (showCreateForm) editingId = null;
        "
      >
        <i :class="['pi', showCreateForm ? 'pi-times' : 'pi-plus']" />
        {{ showCreateForm ? "Cancel" : "Create Profile" }}
      </button>
    </div>

    <InlineDemandProfileForm
      v-if="showCreateForm"
      :app-id="appId"
      class="mb-4 rounded-lg overflow-hidden border border-blue-100"
      @created="onProfileCreated"
      @cancel="showCreateForm = false"
    />

    <!-- Configured profiles — iterated directly from profiles, not filtered through NETWORK_DEFS -->
    <div
      v-if="profiles.length"
      class="grid grid-cols-1 gap-3 md:grid-cols-2 mb-6"
    >
      <div
        v-for="profile in profiles"
        :key="profile.id"
        :class="[
          'border border-gray-200 rounded-lg bg-white overflow-hidden',
          !profile.enabled && 'opacity-60',
        ]"
      >
        <div class="card-header">
          <div class="flex items-center gap-2 flex-wrap min-w-0">
            <span class="font-semibold text-gray-800">
              {{ profileLabel(profile) }}
            </span>
            <span
              :class="
                profile.enabled ? 'badge badge-active' : 'badge badge-disabled'
              "
            >
              <span
                :class="[
                  'w-1.5 h-1.5 rounded-full',
                  profile.enabled ? 'bg-green-500' : 'bg-red-400',
                ]"
              />
              {{ profile.enabled ? "Active" : "Disabled" }}
            </span>
          </div>
          <div class="flex gap-2 shrink-0">
            <button
              :class="[
                'btn-sm',
                editingId === profile.id ? 'btn-cancel' : 'btn-edit',
              ]"
              @click="toggleEdit(profile.id)"
            >
              <i
                :class="[
                  'pi',
                  editingId === profile.id ? 'pi-times' : 'pi-pencil',
                ]"
              />
              {{ editingId === profile.id ? "Cancel" : "Edit" }}
            </button>
          </div>
        </div>

        <InlineDemandProfileForm
          v-if="editingId === profile.id"
          :app-id="appId"
          :network-key="
            accountTypeToNetworkKey(
              profile.accountType ?? profile.account?.type,
            )
          "
          :initial-profile="profile"
          @updated="onProfileUpdated"
          @cancel="editingId = null"
        />

        <div v-else class="divide-y divide-gray-100">
          <div class="flex items-start gap-3 px-6 py-3">
            <span class="text-xs text-gray-400 w-28 shrink-0 pt-px">
              Account
            </span>
            <span class="text-sm text-gray-700">
              {{ accountLabel(profile.account) }}
            </span>
          </div>
          <div class="flex items-start gap-3 px-6 py-3">
            <span class="text-xs text-gray-400 w-28 shrink-0 pt-px">
              Demand Source
            </span>
            <NuxtLink
              v-if="profile.demandSource"
              :to="`/demand_sources/${profile.demandSource.id}`"
              class="text-sm text-blue-600 hover:underline"
            >
              {{ profile.demandSource.humanName }}
            </NuxtLink>
            <span v-else class="text-sm text-gray-400">—</span>
          </div>
          <template v-if="profile.data && Object.keys(profile.data).length">
            <div
              v-for="(value, key) in profile.data"
              :key="key"
              class="flex items-start gap-3 px-6 py-3"
            >
              <span
                class="text-xs text-gray-400 w-28 shrink-0 pt-px capitalize"
              >
                {{ key.replace(/_/g, " ") }}
              </span>
              <span class="text-sm text-gray-700 font-mono break-all">
                {{ value }}
              </span>
            </div>
          </template>
          <div v-else class="flex items-start gap-3 px-6 py-3">
            <span class="text-xs text-gray-400 w-28 shrink-0 pt-px"
              >Config Data</span
            >
            <span class="text-sm text-gray-400">No adapter data</span>
          </div>
        </div>
      </div>
    </div>

    <p v-if="!profiles.length" class="text-sm text-gray-400 py-4">
      No demand sources configured for this app yet.
    </p>
  </div>
</template>

<script setup>
import { NETWORK_DEFS } from "@/constants/Networks.js";

const props = defineProps({
  demandProfiles: { type: Array, required: true },
  appId: { type: [Number, String], required: true },
});

const profiles = ref([...props.demandProfiles]);
watch(
  () => props.demandProfiles,
  (val) => {
    profiles.value = [...val];
  },
);
const editingId = ref(null);
const showCreateForm = ref(false);

const accountTypeToNetworkKey = (accountType) =>
  NETWORK_DEFS.find((n) => n.accountType === accountType)?.key ?? accountType;

function profileLabel(profile) {
  return (
    profile.demandSource?.humanName ??
    NETWORK_DEFS.find(
      (n) => n.accountType === (profile.accountType ?? profile.account?.type),
    )?.label ??
    profile.accountType ??
    profile.account?.type ??
    "Unknown"
  );
}

function accountLabel(account) {
  if (!account) return "";
  const type = account.type?.split("::")?.[1] ?? account.type;
  return account.label
    ? `${type} · ${account.label}`
    : `${type} (#${account.id})`;
}

function toggleEdit(profileId) {
  editingId.value = editingId.value === profileId ? null : profileId;
  showCreateForm.value = false;
}

function onProfileUpdated(updated) {
  const idx = profiles.value.findIndex((p) => p.id === updated.id);
  if (idx !== -1) profiles.value[idx] = updated;
  editingId.value = null;
}

function onProfileCreated(created) {
  profiles.value.push(created);
  showCreateForm.value = false;
}
</script>
