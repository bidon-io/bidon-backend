export default function ({ path, message }) {
  return useFormSubmit({
    path,
    message,
    method: "patch",
    showToastLater: false,
  });
}
