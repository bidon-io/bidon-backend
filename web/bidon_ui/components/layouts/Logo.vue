<template>
  <div class="px-3 py-2" style="border-bottom: 1px solid var(--bidon-border-default); background-color: var(--bidon-bg-sidebar);">
    <NuxtLink to="/" class="block">
      <img class="w-full h-auto rounded-lg bg-transparent" :src="isDark ? '/logo-dark.png' : '/logo-light.png'" alt="BidOn" />
    </NuxtLink>
  </div>
</template>

<script setup lang="ts">
const { isDark } = useColorMode();
</script>
