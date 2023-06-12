<template>
  <form @submit.prevent="handleSubmit">
    <Card class="p-6">
      <template #title>Auction Config</template>
      <template #content>
        <div class="divide-y">
          <div class="flex flex-row py-2">
            <div class="w-1/4 px-6">
              <div class="font-semibold text-gray-500">Name</div>
            </div>
            <div class="px-6">
              <InputText v-model="resource.name" type="text" placeholder="Name" />
            </div>
          </div>

          <div class="flex flex-row py-2">
            <div class="w-1/4 px-6">
              <div class="font-semibold text-gray-500">App</div>
            </div>
            <div class="px-6">
              <Dropdown
                v-model="resource.app_id"
                :options="apps"
                option-label="package_name"
                option-value="id"
                class="w-full md:w-14rem"
                placeholder="Select App"
              />
            </div>
          </div>

          <div class="flex flex-row py-2">
            <div class="w-1/4 px-6">
              <div class="font-semibold text-gray-500">Ad Type</div>
            </div>
            <div class="px-6">
              <Dropdown
                v-model="resource.ad_type"
                :options="adTypes"
                class="w-full md:w-14rem"
                placeholder="Select Ad Type"
              />
            </div>
          </div>

          <div class="flex flex-row py-2">
            <div class="w-1/4 px-6">
              <div class="font-semibold text-gray-500">Price floor</div>
            </div>
            <div class="px-6">
              <InputNumber
                v-model="resource.pricefloor"
                input-id="pricefloor"
                :min-fraction-digits="2"
                :max-fraction-digits="5"
                placeholder="Price floor"
              />
            </div>
          </div>

          <div class="flex flex-row py-2">
            <div class="w-1/4 px-6">
              <div class="font-semibold text-gray-500">Rounds</div>
            </div>
            <div class="px-6">
              <Textarea v-model="rounds" rows="10" cols="80" />
            </div>
          </div>
        </div>
        <div class="my-4">
          <Button type="submit" label="Save" icon="pi pi-save" class="p-button-success" />
        </div>
      </template>
    </Card>
  </form>
</template>

<script setup>
import { defineProps, defineEmits } from "vue";
import axios from "@/services/ApiService.js";

const props = defineProps({
  value: {
    type: Object,
    required: true,
  },
});
const emit = defineEmits(["submit"]);

const resource = ref(props.value);

const apps = ref([]);
axios
  .get("/apps")
  .then((response) => {
    apps.value = response.data;
  })
  .catch((error) => {
    console.error(error);
  });

const adTypes = ref(["banner", "interstitial", "rewarded_video"]);

const rounds = computed({
  get: () => JSON.stringify(resource.value.rounds, null, 2),
  set: (newValue) => {
    try {
      resource.value.rounds = JSON.parse(newValue);
    } catch (error) {
      console.error("Error parsing JSON:", error);
    }
  },
});

function handleSubmit() {
  emit("submit", resource.value);
}
</script>
