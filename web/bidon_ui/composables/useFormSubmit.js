import { useToast } from "primevue/usetoast";
import axios from "@/services/ApiService.js";

export default function (resourcesPath, message) {
  const toast = useToast();
  const handleSubmit = (event) => {
    axios
      .post(resourcesPath, event)
      .then(async (response) => {
        const id = response.data.id;
        await navigateTo(`${resourcesPath}/${id}`);

        toast.add({
          severity: "success",
          summary: "Success",
          detail: message,
        });
      })
      .catch((error) => {
        console.error(error);
        toast.add({
          severity: "error",
          summary: "Error",
          detail: error.message,
        });
      });
  };
  return handleSubmit;
}
