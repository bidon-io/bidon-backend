<template>
  <div class="flex flex-col p-4 border rounded shadow-sm">
    <FileUpload
      mode="basic"
      :name="label"
      accept=".csv"
      :auto="true"
      :choose-label="label"
      custom-upload
      @uploader="onUpload"
    />
    <small class="text-gray-500 mt-2">
      CSV format: name, price, price_point (e.g., "T1_BANNER", 0.50, "pp_123")
    </small>
    <Message v-if="error" severity="error" class="mt-2">{{ error }}</Message>
    <ScrollPanel
      v-if="content.length"
      style="width: 100%; height: 200px"
      class="mt-4"
    >
      <table>
        <thead>
          <tr>
            <th v-for="header in headers" :key="header">{{ header }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, rowIndex) in content" :key="rowIndex">
            <td v-for="cell in row" :key="cell">{{ cell }}</td>
          </tr>
        </tbody>
      </table>
    </ScrollPanel>
  </div>
</template>

<script setup>
const props = defineProps({
  label: {
    type: String,
    required: true,
  },
  valueModel: {
    type: Array,
    default: () => [],
  },
});
const emit = defineEmits(["update:modelValue"]);
const value = computed({
  get() {
    return props.valueModel;
  },
  set(value) {
    emit("update:modelValue", value);
  },
});

const headers = ref([]);
const content = ref([]);
const error = ref("");

const validateRow = (row, rowIndex) => {
  if (!row[0] || row[0].trim() === "") {
    return `Row ${rowIndex + 1}: name is required`;
  }
  if (row[1] === undefined || row[1].trim() === "") {
    return `Row ${rowIndex + 1}: price is required`;
  }
  const price = parseFloat(row[1]);
  if (isNaN(price)) {
    return `Row ${rowIndex + 1}: price must be a number (got "${row[1]}")`;
  }
  if (!row[2] || row[2].trim() === "") {
    return `Row ${rowIndex + 1}: price_point is required`;
  }
  return null;
};

const contentToJson = (content) =>
  content.map((row) => ({
    name: row[0]?.trim(),
    price: parseFloat(row[1]),
    pricePoint: row[2]?.trim(),
  }));

const onUpload = (event) => {
  const file = event.files[0];
  const reader = new FileReader();

  reader.onload = function (e) {
    error.value = "";
    const data = e.target.result;
    const rows = data.split("\n").filter((row) => row.trim() !== "");
    if (rows.length < 2) {
      error.value = "CSV must have a header row and at least one data row";
      return;
    }
    const [headersValue, ...contentValue] = rows.map((row) => row.split(","));

    if (headersValue.length < 3) {
      error.value = "CSV must have 3 columns: name, price, price_point";
      return;
    }

    for (let i = 0; i < contentValue.length; i++) {
      const rowError = validateRow(contentValue[i], i);
      if (rowError) {
        error.value = rowError;
        return;
      }
    }

    headers.value = headersValue;
    content.value = contentValue;

    value.value = contentToJson(contentValue);
  };
  reader.readAsText(file);
};
</script>
