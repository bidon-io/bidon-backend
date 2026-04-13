<template>
  <nav class="mt-4 flex-1 px-2 pb-4">
    <div class="mt-2">
      <button
        class="flex items-center justify-between w-full px-4 py-2 text-xs font-semibold uppercase tracking-wider transition-colors" style="color: var(--bidon-accent);"
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
              'flex items-center mt-0.5 px-4 py-2 rounded-md text-sm transition-colors',
              isActive(resourcePath(resource.key))
                ? 'text-white font-medium'
                : 'nav-link-inactive',
            ]"
            :style="isActive(resourcePath(resource.key)) ? 'background-color: var(--bidon-accent-subtle); color: var(--bidon-accent-text); border-left: 3px solid var(--bidon-accent); padding-left: calc(1rem - 3px); margin-left: -0.5rem; margin-right: -0.5rem; padding-right: 1rem; border-radius: 0;' : ''"
          >
            <span>{{ title(resource.key) }}</span>
          </NuxtLink>
        </div>
      </Transition>
    </div>

    <div class="mt-5">
      <button
        class="flex items-center justify-between w-full px-4 py-2 text-xs font-semibold uppercase tracking-wider transition-colors" style="color: var(--bidon-accent);"
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
              'flex items-center mt-0.5 px-4 py-2 rounded-md text-sm transition-colors',
              isActive(resourcePath(resource.key))
                ? 'font-medium'
                : 'nav-link-inactive',
            ]"
            :style="isActive(resourcePath(resource.key)) ? 'background-color: var(--bidon-accent-subtle); color: var(--bidon-accent-text); border-left: 3px solid var(--bidon-accent); padding-left: calc(1rem - 3px); margin-left: -0.5rem; margin-right: -0.5rem; padding-right: 1rem; border-radius: 0;' : ''"
          >
            <span>{{ title(resource.key) }}</span>
          </NuxtLink>
        </div>
      </Transition>
    </div>

    <div class="mt-5">
      <button
        class="flex items-center justify-between w-full px-4 py-2 text-xs font-semibold uppercase tracking-wider transition-colors" style="color: var(--bidon-accent);"
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
              'flex items-center mt-0.5 px-4 py-2 rounded-md text-sm transition-colors',
              isActive(resourcePath(resource.key))
                ? 'font-medium'
                : 'nav-link-inactive',
            ]"
            :style="isActive(resourcePath(resource.key)) ? 'background-color: var(--bidon-accent-subtle); color: var(--bidon-accent-text); border-left: 3px solid var(--bidon-accent); padding-left: calc(1rem - 3px); margin-left: -0.5rem; margin-right: -0.5rem; padding-right: 1rem; border-radius: 0;' : ''"
          >
            <span>{{ title(resource.key) }}</span>
          </NuxtLink>
          <NuxtLink
            to="/settings/security"
            :class="[
              'flex items-center mt-0.5 px-4 py-2 rounded-md text-sm transition-colors',
              isActive('/settings/security')
                ? 'font-medium'
                : 'nav-link-inactive',
            ]"
            :style="isActive('/settings/security') ? 'background-color: var(--bidon-accent-subtle); color: var(--bidon-accent-text); border-left: 3px solid var(--bidon-accent); padding-left: calc(1rem - 3px); margin-left: -0.5rem; margin-right: -0.5rem; padding-right: 1rem; border-radius: 0;' : ''"
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
          'flex items-center mt-0.5 px-4 py-2 rounded-md text-sm transition-colors',
          isActive('/copilot')
            ? 'font-medium'
            : 'nav-link-inactive',
        ]"
        :style="isActive('/copilot') ? 'background-color: var(--bidon-accent-subtle); color: var(--bidon-accent-text); border-left: 3px solid var(--bidon-accent); padding-left: calc(1rem - 3px); margin-left: -0.5rem; margin-right: -0.5rem; padding-right: 1rem; border-radius: 0;' : ''"
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

const APP_RESOURCE_KEYS = [
  "app",
];

const GLOBAL_RESOURCE_KEYS = ["demandSource", "demandSourceAccount", "segment"];

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
  return route.path === path || route.path.startsWith(path + "/");
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
