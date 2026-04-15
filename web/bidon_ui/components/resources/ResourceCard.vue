<template>
  <FormCard :title="title">
    <template v-if="hasHeaderActionsSlot" #headerActions>
      <slot name="headerActions" />
    </template>
    <FormField
      v-for="field in fields"
      :key="field.key ?? field.label"
      :label="field.label"
    >
      <div
        v-if="!field.type"
        class="flex items-start gap-2 min-w-0"
        style="color: var(--bidon-text-primary)"
      >
        <button
          v-if="field.copyable"
          type="button"
          class="table-action-btn shrink-0 mt-0.5"
          aria-label="Copy"
          @click="copyField(localResource[field.key])"
        >
          <i class="pi pi-copy text-xs" />
        </button>
        <span class="min-w-0 break-words">{{ localResource[field.key] }}</span>
      </div>
      <div
        v-if="field.type === 'static'"
        class="flex items-start gap-2 min-w-0"
        style="color: var(--bidon-text-primary)"
      >
        <button
          v-if="field.copyable"
          type="button"
          class="table-action-btn shrink-0 mt-0.5"
          aria-label="Copy"
          @click="copyField(field.value)"
        >
          <i class="pi pi-copy text-xs" />
        </button>
        <span class="min-w-0 break-words">{{ field.value }}</span>
      </div>
      <ResourceLink
        v-if="field.type === 'link'"
        :link="field.link"
        :data="localResource"
      />
      <AssociatedResourcesLink
        v-if="field.type === 'associatedResourcesLink'"
        :link="field.associatedResourcesLink"
        :data="localResource"
      />
      <Textarea
        v-if="field.type === 'textarea'"
        :value="JSON.stringify(localResource[field.key])"
        rows="5"
        cols="50"
        disabled
      />
    </FormField>

    <!-- Slot for additional content -->
    <div v-if="$slots.default" class="pt-4">
      <slot></slot>
    </div>

    <template v-if="hasFooterSlot" #footer>
      <slot name="footer" />
    </template>
  </FormCard>
</template>

<script setup>
import { computed, useSlots } from "vue";

const slots = useSlots();
const hasFooterSlot = computed(() => !!slots.footer);
const hasHeaderActionsSlot = computed(() => !!slots.headerActions);

const props = defineProps({
  title: {
    type: String,
    required: true,
  },
  resource: {
    type: Object,
    required: true,
  },
  fields: {
    type: Array,
    required: true,
  },
});

const localResource = ref(props.resource);

const copyField = (value) => {
  navigator.clipboard.writeText(value == null ? "" : String(value));
};
</script>
