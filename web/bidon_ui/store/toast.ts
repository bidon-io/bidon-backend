import { defineStore } from "pinia";
import { ToastServiceMethods } from "primevue/toastservice";
import { ToastMessageOptions } from "primevue/toast";

export const useToastStore = defineStore("toastStore", () => {
  const toasts = ref<ToastMessageOptions[]>([]);
  function addToast(toast: ToastMessageOptions) {
    toasts.value.push(toast);
  }
  function showToasts(toastService: ToastServiceMethods) {
    toasts.value.forEach((toastMessage) => toastService.add(toastMessage));
    toasts.value = [];
  }
  return { toasts, addToast, showToasts };
});
