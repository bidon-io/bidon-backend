<template>
  <button
    class="inline-flex items-center gap-1 text-xs mb-3 transition-colors"
    style="color: var(--bidon-muted);"
    @mouseenter="$event.currentTarget.style.color = 'var(--bidon-primary)'"
    @mouseleave="$event.currentTarget.style.color = 'var(--bidon-muted)'"
    @click.prevent="goBack"
  >
    <i class="pi pi-arrow-left text-xs" />
    Back
  </button>
</template>

<script setup>
const props = defineProps({ path: { type: String, default: "/" } });
const router = useRouter();

const goBack = () => {
  if (history.state?.back !== null) {
    router.back();
  } else {
    router.push(props.path);
  }
};
</script>
