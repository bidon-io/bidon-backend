<template>
  <div
    class="flex h-screen overflow-hidden"
    style="background-color: var(--bidon-bg-layout)"
  >
    <!-- Mobile backdrop -->
    <Transition name="fade">
      <button
        v-if="sidebarOpen && !isLarge"
        type="button"
        class="fixed inset-0 z-40 bg-black/40"
        aria-label="Close menu"
        @click="closeSidebar"
      />
    </Transition>

    <aside
      :class="[
        'fixed inset-y-0 left-0 z-50 flex h-full shrink-0 flex-col',
        'shadow-lg transition-transform duration-300 ease-out',
        'lg:relative lg:inset-auto lg:z-auto lg:shadow-none',
        sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
      ]"
    >
      <Sidebar />
    </aside>

    <div class="relative z-[30] flex min-h-0 min-w-0 flex-1 flex-col">
      <Header />
      <Toast />
      <ConfirmDialog />
      <div
        class="flex min-h-0 flex-1 flex-col justify-between overflow-x-hidden overflow-y-auto"
        style="background-color: var(--bidon-bg-layout)"
      >
        <main
          class="container mx-auto max-w-none px-4 py-6 sm:px-6 sm:py-8 lg:px-8 lg:py-10"
        >
          <slot></slot>
        </main>
        <Footer />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const { sidebarOpen, closeSidebar, isLarge } = useLayoutNav();
useLayoutNavSetup();
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
