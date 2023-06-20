import { useConfirm } from "primevue/useconfirm";
import { useToast } from "primevue/usetoast";
import axios from "@/services/ApiService.js";
import { useToastStore } from "@/store/toast";

export default function ({ path, hook, showToastLater = false }) {
  const confirm = useConfirm();

  let showToast;
  const toastMessage = { severity: "info", summary: "Success", detail: "Record deleted", life: 3000 };
  if (showToastLater) {
    const { addToast } = useToastStore();
    showToast = () => addToast(toastMessage);
  } else {
    const toastService = useToast();
    showToast = () => toastService.add(toastMessage);
  }

  async function deleteResource(id, callback) {
    await axios.delete(`${path}/${id}`);
    await callback();
  }

  const deleteHandle = (id) => {
    confirm.require({
      message: "Do you want to delete this record?",
      header: "Delete Confirmation",
      icon: "pi pi-info-circle",
      acceptClass: "p-button-danger",
      accept: () => {
        deleteResource(id, () => {
          hook(id);
          showToast();
        });
      },
    });
  };
  return deleteHandle;
}
