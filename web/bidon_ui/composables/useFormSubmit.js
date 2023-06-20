import { useToast } from "primevue/usetoast";
import axios from "@/services/ApiService.js";
import { useToastStore } from "@/store/toast";

export default function ({
  path,
  method,
  message,
  showToastLater = false,
  hook = async () => {
    /* no operation function */
  },
}) {
  const toast = useToast();
  const { addToast } = useToastStore();
  const toastMessage = {
    severity: "success",
    summary: "Success",
    detail: message,
  };

  const showSuccessMessage = showToastLater ? () => addToast(toastMessage) : () => toast.add(toastMessage);
  const handleSubmit = (event) => {
    axios[method](path, event)
      .then(async (response) => {
        const id = response.data.id;
        await hook(id);
        showSuccessMessage();
      })
      .catch((error) => {
        console.error(error);
        toast.add({
          severity: "error",
          summary: "Error",
          detail: error.error.message,
        });
      });
  };
  return handleSubmit;
}
