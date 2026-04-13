<template>
  <div class="flex h-screen" style="background-color: var(--bidon-bg-layout);">
    <Sidebar />
    <div class="flex flex-col flex-1 min-w-0">
      <Header></Header>
      <Toast />
      <ConfirmDialog />
      <div
        class="flex-1 flex flex-col justify-between overflow-x-hidden overflow-y-auto"
        style="background-color: var(--bidon-bg-layout);"
      >
        <main class="container mx-auto px-8 py-10">
          <slot></slot>
        </main>
        <Footer />
      </div>
    </div>
  </div>
</template>
