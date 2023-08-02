<template>
  <div class="flex h-screen bg-gray-100">
    <Sidebar />
    <div class="flex flex-col flex-1">
      <Header></Header>
      <Toast />
      <ConfirmDialog />
      <main class="flex-1 overflow-x-hidden overflow-y-auto bg-gray-200">
        <div class="container mx-auto px-8 py-10">
          <slot></slot>
        </div>
      </main>
      <Footer />
    </div>
  </div>
</template>
