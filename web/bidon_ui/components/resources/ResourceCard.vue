<template>
  <FormCard :title="title">
    <FormField v-for="field in fields" :key="field.key" :label="field.label">
      <div v-if="!field.type" style="color: var(--bidon-text-primary);">
        <button
          v-if="field.copyable"
          @click="copyField(localResource[field.key])"
        >
          <i class="pi pi-copy table-action-btn"></i>
        </button>
        {{ localResource[field.key] }}
      </div>
      <div v-if="field.type === 'static'" style="color: var(--bidon-text-primary);">
        <button v-if="field.copyable" @click="copyField(field.value)">
          <i class="pi pi-copy table-action-btn"></i>
        </button>
        {{ field.value }}
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
  </FormCard>
</template>

<script setup>
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

const copyField = (field) => {
  navigator.clipboard.writeText(field);
};
</script>
