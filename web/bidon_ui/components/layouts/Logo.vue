<template>
  <div class="px-3 py-2 logo-container">
    <NuxtLink to="/apps" class="block">
      <img
        class="w-full h-auto rounded-lg bg-transparent"
        :src="isDark ? '/logo-dark.png' : '/logo-light.png'"
        alt="BidOn"
      />
    </NuxtLink>
  </div>
</template>

<script setup lang="ts">
const { isDark } = useColorMode();
</script>
