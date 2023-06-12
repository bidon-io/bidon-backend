<template>
  <Toast/>
  <form class="flex-1 p-6 mx-auto w-full" @submit.prevent="handleSubmit">
    <Card class="p-6">
      <template #title>Auction Config {{ resource.id }}</template>
      <template #content>
        <div class="divide-y">
          <div class="my-4">
          <Button type="submit" label="Save" icon="pi pi-save" class="p-button-success"/>
          </div>
          <div class="flex flex-row py-2">
            <div class="w-1/4 px-6">
              <div class="font-semibold text-gray-500">Name</div>
            </div>
            <div class="px-6">
              <InputText type="text" v-model="resource.name" />
            </div>
          </div>

          <div class="flex flex-row py-2">
            <div class="w-1/4 px-6">
              <div class="font-semibold text-gray-500">App</div>
            </div>
            <div class="px-6">
              <Dropdown v-model="selectedApp" :options="apps" optionLabel="package_name" optionValue="id" class="w-full md:w-14rem" placeholder="Select App" />
            </div>
          </div>

          <div class="flex flex-row py-2">
            <div class="w-1/4 px-6">
              <div class="font-semibold text-gray-500">Ad Type</div>
            </div>
            <div class="px-6">
              <Dropdown v-model="resource.ad_type" :options="adTypes" class="w-full md:w-14rem" placeholder="Select Ad Type" />
            </div>
          </div>

          <div class="flex flex-row py-2">
            <div class="w-1/4 px-6">
              <div class="font-semibold text-gray-500">Pricefloor</div>
            </div>
            <div class="px-6">
              <InputNumber v-model="resource.pricefloor" inputId="pricefloor" :minFractionDigits="2" :maxFractionDigits="5" placeholder="PriceFloor"/>
            </div>
          </div>

          <div class="flex flex-row py-2">
            <div class="w-1/4 px-6">
              <div class="font-semibold text-gray-500">Rounds</div>
            </div>
            <div class="px-6">
              <Textarea v-model="rounds" rows="10" cols="80"/>
            </div>
          </div>
        </div>
      </template>
    </Card>
  </form>
</template>

<script setup>
import { ref } from "vue";
import axios from "@/services/ApiService.js";
import { useToast } from "primevue/usetoast";

const route = useRoute();
const id = route.params.id
const resource = ref({});

axios.get(`/auction_configurations/${id}`)
.then((response) => {
  resource.value = response.data;
}).catch(error => { 
  console.error(error);
  resource.value = { error: error.message };
});

const apps = ref([]);
const selectedApp = ref();
axios.get('/apps').then((response) => {
  apps.value = response.data;
  const app = apps.value.find(app => app.id === resource.value.app_id);
  selectedApp.value = app.id;
}).catch(error => { 
  console.error(error);
});

const adTypes = ref(['banner', 'interstitial', 'rewarded_video']);

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

const toast = useToast();
const handleSubmit = () => {
  axios.patch(`/auction_configurations/${id}`, resource.value).then((response) => {
    console.log(response);
    toast.add({
      severity: "success",
      summary: "Success",
      detail: "Auction configuration updated",
    });
  }).catch(error => { 
    console.error(error);
    toast.add({
      severity: "error",
      summary: "Error",
      detail: error.message,
    });
  });
};
</script>
