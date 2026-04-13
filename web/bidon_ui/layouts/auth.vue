<template>
  <div class="auth-layout flex h-screen">
    <div class="flex flex-col flex-1">
      <div class="flex-1 flex flex-col justify-between overflow-x-hidden overflow-y-auto">
        <div>
          <!-- Brand header for auth pages -->
          <div class="flex items-center justify-center py-10">
            <img class="w-48 h-auto" :src="'/logo-light.png'" alt="BidOn" />
          </div>
          <Toast />
          <ConfirmDialog />
          <main class="container mx-auto px-8 pb-10">
            <slot></slot>
          </main>
        </div>
        <footer class="py-4 text-center">
          <p class="text-xs" style="color: #8C8C9C;">&copy; BidOn. All rights reserved.</p>
        </footer>
      </div>
    </div>
  </div>
</template>

<style>
/* Force light mode on the auth layout regardless of user preference */
.auth-layout,
.auth-layout * {
  --bidon-accent:         #23283B;
  --bidon-accent-hover:   #1a1f2e;
  --bidon-accent-subtle:  rgba(35, 40, 59, 0.07);
  --bidon-accent-border:  rgba(35, 40, 59, 0.2);
  --bidon-accent-text:    #23283B;
  --bidon-primary:        #10AF6C;
  --bidon-primary-hover:  #0E9C62;
  --bidon-muted:          #8C8C9C;
  --bidon-text-primary:   #23283B;
  --bidon-text-secondary: #8C8C9C;
  --bidon-bg-layout:      #E1E7E5;
  --bidon-bg-sidebar:     #F0F1F4;
  --bidon-bg-card:        #FFFFFF;
  --bidon-bg-card-header: #F4F6F8;
  --bidon-border-default: #E1E7E5;
  --bidon-shadow-sm:      0 1px 3px rgba(10, 6, 48, 0.08);
  --bidon-shadow-focus:   0 0 0 3px rgba(35, 40, 59, 0.25);
  color-scheme: light;
}

.auth-layout {
  background-color: #E1E7E5;
}
</style>
