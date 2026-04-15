<template>
  <nav class="mt-4 flex-1 px-2 pb-4">
    <div class="mt-2">
      <button
        class="flex items-center justify-between w-full px-4 py-2 text-xs font-semibold uppercase tracking-wider transition-colors"
        style="color: var(--bidon-accent)"
        @click="openSections.apps = !openSections.apps"
      >
        <span>Apps</span>
        <svg
          :class="[
            'w-3 h-3 transition-transform',
            openSections.apps ? 'rotate-180' : '',
          ]"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M19 9l-7 7-7-7"
          />
        </svg>
      </button>
      <Transition name="section-slide">
        <div v-if="openSections.apps" class="mt-1">
          <NuxtLink
            v-for="resource in appResources"
            :key="resource.key"
            :to="resourcePath(resource.key)"
            :class="[
              'flex items-center mt-0.5 py-2 text-sm transition-colors',
              isActive(resourcePath(resource.key))
                ? 'nav-link-active'
                : 'nav-link-inactive rounded-md px-4',
            ]"
          >
            <span>{{ title(resource.key) }}</span>
          </NuxtLink>
        </div>
      </Transition>
    </div>

    <div class="mt-5">
      <button
        class="flex items-center justify-between w-full px-4 py-2 text-xs font-semibold uppercase tracking-wider transition-colors"
        style="color: var(--bidon-accent)"
        @click="openSections.global = !openSections.global"
      >
        <span>Global Configuration</span>
        <svg
          :class="[
            'w-3 h-3 transition-transform',
            openSections.global ? 'rotate-180' : '',
          ]"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M19 9l-7 7-7-7"
          />
        </svg>
      </button>
      <Transition name="section-slide">
        <div v-if="openSections.global" class="mt-1">
          <NuxtLink
            v-for="resource in globalResources"
            :key="resource.key"
            :to="resourcePath(resource.key)"
            :class="[
              'flex items-center mt-0.5 py-2 text-sm transition-colors',
              isActive(resourcePath(resource.key))
                ? 'nav-link-active'
                : 'nav-link-inactive rounded-md px-4',
            ]"
          >
            <span>{{ title(resource.key) }}</span>
          </NuxtLink>
        </div>
      </Transition>
    </div>

    <div class="mt-5">
      <button
        class="flex items-center justify-between w-full px-4 py-2 text-xs font-semibold uppercase tracking-wider transition-colors"
        style="color: var(--bidon-accent)"
        @click="openSections.settings = !openSections.settings"
      >
        <span>Settings</span>
        <svg
          :class="[
            'w-3 h-3 transition-transform',
            openSections.settings ? 'rotate-180' : '',
          ]"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M19 9l-7 7-7-7"
          />
        </svg>
      </button>
      <Transition name="section-slide">
        <div v-if="openSections.settings" class="mt-1">
          <NuxtLink
            v-for="resource in settingsResources"
            :key="resource.key"
            :to="resourcePath(resource.key)"
            :class="[
              'flex items-center mt-0.5 py-2 text-sm transition-colors',
              isActive(resourcePath(resource.key))
                ? 'nav-link-active'
                : 'nav-link-inactive rounded-md px-4',
            ]"
          >
            <span>{{ title(resource.key) }}</span>
          </NuxtLink>
          <NuxtLink
            to="/settings/security"
            :class="[
              'flex items-center mt-0.5 py-2 text-sm transition-colors',
              isActive('/settings/security')
                ? 'nav-link-active'
                : 'nav-link-inactive rounded-md px-4',
            ]"
          >
            <span>Passwords</span>
          </NuxtLink>
        </div>
      </Transition>
    </div>

    <div class="mt-4">
      <NuxtLink
        v-if="currentUser?.isAdmin === true"
        to="/copilot"
        :class="[
          'flex items-center mt-0.5 py-2 text-sm transition-colors',
          isActive('/copilot')
            ? 'nav-link-active'
            : 'nav-link-inactive rounded-md px-4',
        ]"
      >
        <span>AI Copilot</span>
      </NuxtLink>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { pluralize, titleize } from "inflection";

interface User {
  isAdmin?: boolean;
}

const APP_RESOURCE_KEYS = ["app"];

const GLOBAL_RESOURCE_KEYS = ["demandSource", "segment"];

const SETTINGS_RESOURCE_KEYS = ["user", "apiKey"];

const resources = useResources();
const route = useRoute();
const authStore = useAuthStore();
const currentUser = computed<User | null>(() => authStore.currentUser);

const openSections = reactive({ apps: true, global: true, settings: true });

function isCountry(key: string): boolean {
  return key === "country";
}

function filterItems(keys: string[]) {
  return computed(() => {
    if (!resources.state) return [];
    return keys
      .filter((key) => resources.state![key] && !isCountry(key))
      .map((key) => resources.state![key]);
  });
}

const appResources = filterItems(APP_RESOURCE_KEYS);
const globalResources = filterItems(GLOBAL_RESOURCE_KEYS);
const settingsResources = filterItems(SETTINGS_RESOURCE_KEYS);

function title(key: string) {
  if (key === "auction_configuration_v2") return "Auction Configurations";
  if (key === "api_key") return "API Credentials";
  return pluralize(titleize(key));
}

function resourcePath(key: string) {
  if (key === "auction_configuration_v2") return "/v2/auction_configurations";
  return `/${pluralize(key)}`;
}

function isActive(path: string): boolean {
  if (route.path === path || route.path.startsWith(path + "/")) {
    return true;
  }
  // Accounts are managed under each demand source; keep Demand Sources highlighted.
  if (
    path === "/demand_sources" &&
    route.path.startsWith("/demand_source_accounts")
  ) {
    return true;
  }
  return false;
}
</script>

<!--Used by the sidebar sections to animate the open / collapsed state-->
<style scoped>
.section-slide-enter-active,
.section-slide-leave-active {
  transition:
    max-height 0.2s ease-out,
    opacity 0.2s ease-out;
  max-height: 400px;
  overflow: hidden;
}

.section-slide-enter-from,
.section-slide-leave-to {
  max-height: 0;
  opacity: 0;
}
</style>
