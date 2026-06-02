<template>
  <div class="mt-10">
    <div
      class="mb-4 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"
    >
      <div>
        <h3
          class="text-base font-semibold"
          style="color: var(--bidon-text-primary)"
        >
          Demand Source Profiles
        </h3>
        <p class="text-sm mt-0.5" style="color: var(--bidon-muted)">
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
      class="mb-4 rounded-lg overflow-hidden"
      style="border: 1px solid rgba(16, 175, 108, 0.2)"
      @created="onProfileCreated"
      @cancel="showCreateForm = false"
    />

    <!-- Configured profiles — iterated directly from profiles, not filtered through NETWORK_DEFS -->
    <div
      v-if="profiles.length"
      class="grid grid-cols-1 gap-3 md:grid-cols-2 mb-6 items-stretch"
    >
      <div
        v-for="profile in profiles"
        :key="profile.id"
        :class="[
          'rounded-lg overflow-hidden flex flex-col',
          editingId === profile.id ? 'self-start w-full' : 'h-full',
          !profile.enabled && 'opacity-60',
        ]"
        style="
          background-color: var(--bidon-bg-card);
          border: 1px solid var(--bidon-border-default);
          box-shadow: var(--bidon-shadow-sm);
        "
      >
        <div class="card-header shrink-0">
          <div class="flex items-center gap-2 flex-wrap min-w-0">
            <span
              class="font-semibold"
              style="color: var(--bidon-text-primary)"
            >
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

        <div
          class="flex flex-col"
          :class="editingId !== profile.id && 'flex-1 min-h-0'"
        >
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

          <div
            v-else
            class="divide-y flex flex-col flex-1 min-h-0"
            style="border-color: var(--bidon-border-default)"
          >
            <div class="flex items-start gap-3 px-6 py-3">
              <span
                class="text-xs w-28 shrink-0 pt-px"
                style="color: var(--bidon-muted)"
              >
                Account
              </span>
              <span class="text-sm" style="color: var(--bidon-text-primary)">
                {{ accountLabel(profile.account) }}
              </span>
            </div>
            <div class="flex items-start gap-3 px-6 py-3">
              <span
                class="text-xs w-28 shrink-0 pt-px"
                style="color: var(--bidon-muted)"
              >
                Demand Source
              </span>
              <NuxtLink
                v-if="profile.demandSource"
                :to="`/demand_sources/${profile.demandSource.id}`"
                class="text-sm hover:underline"
                style="color: var(--bidon-primary)"
              >
                {{ profile.demandSource.humanName }}
              </NuxtLink>
              <span v-else class="text-sm" style="color: var(--bidon-muted)"
                >—</span
              >
            </div>
            <template v-if="profile.data && Object.keys(profile.data).length">
              <div
                v-for="(value, key) in profile.data"
                :key="key"
                class="flex items-start gap-3 px-6 py-3"
              >
                <span
                  class="text-xs w-28 shrink-0 pt-px capitalize"
                  style="color: var(--bidon-muted)"
                >
                  {{ key.replace(/_/g, " ") }}
                </span>
                <span
                  class="text-sm font-mono break-all"
                  style="color: var(--bidon-text-primary)"
                >
                  {{ value }}
                </span>
              </div>
            </template>
            <div v-else class="flex items-start gap-3 px-6 py-3">
              <span
                class="text-xs w-28 shrink-0 pt-px"
                style="color: var(--bidon-muted)"
                >Config Data</span
              >
              <span class="text-sm" style="color: var(--bidon-muted)"
                >No adapter data</span
              >
            </div>
          </div>
        </div>

        <div
          v-if="editingId !== profile.id && profile._permissions?.delete"
          class="card-footer flex shrink-0 items-center justify-end"
        >
          <button
            type="button"
            class="table-action-btn table-action-btn--delete"
            title="Delete profile"
            aria-label="Delete profile"
            @click="deleteHandle(Number(profile.id))"
          >
            <i class="pi pi-trash" />
          </button>
        </div>
      </div>
    </div>

    <p
      v-if="!profiles.length"
      class="text-sm py-4"
      style="color: var(--bidon-muted)"
    >
      No demand sources configured for this app yet.
    </p>
  </div>
</template>

<script setup>
import useDeleteResource from "@/composables/useDeleteResource";
import { NETWORK_DEFS } from "@/constants/Networks.js";
import { mergeResourcePermissions } from "@/utils/mergeResourcePermissions";

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
  if (idx !== -1) {
    profiles.value[idx] = mergeResourcePermissions(
      profiles.value[idx],
      updated,
    );
  }
  editingId.value = null;
}

function onProfileCreated(created) {
  profiles.value.push(mergeResourcePermissions(undefined, created));
  showCreateForm.value = false;
}

const deleteHandle = useDeleteResource({
  path: "/app_demand_profiles",
  hook: (deletedId) => {
    const id = Number(deletedId);
    profiles.value = profiles.value.filter((p) => p.id !== id);
    if (editingId.value === id) {
      editingId.value = null;
    }
  },
  successDetail: "Demand source profile deleted.",
});
</script>
