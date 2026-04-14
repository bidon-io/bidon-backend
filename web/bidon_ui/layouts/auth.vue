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
/* Force light-mode appearance on the auth layout regardless of the user's dark-mode
   preference. CSS custom properties set on html.dark cascade into every descendant,
   so we override only the variables actually consumed by auth-page components here.
   No need to repeat these on .auth-layout * — custom properties inherit automatically. */
.auth-layout {
  color-scheme: light;
  --bidon-bg-layout:      #E1E7E5;
  --bidon-bg-card:        #FFFFFF;
  --bidon-bg-card-header: #F4F6F8;
  --bidon-bg-sidebar:     #F0F1F4;
  --bidon-text-primary:   #23283B;
  --bidon-text-secondary: #8C8C9C;
  --bidon-muted:          #8C8C9C;
  --bidon-border-default: #E1E7E5;
  --bidon-accent:         #23283B;
  --bidon-accent-text:    #23283B;
  --bidon-accent-subtle:  rgba(35, 40, 59, 0.07);
  background-color: #E1E7E5;
}
</style>
