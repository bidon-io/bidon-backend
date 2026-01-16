/**
 * Formats a camelCase or snake_case key into a readable label with spaces and capitalization
 * @param {string} key - The camelCase or snake_case key to format
 * @returns {string} - Formatted label with spaces and capitalization
 */
export const formatLabel = (key) => {
  return key
    .replace(/_/g, " ") // Replace underscores with spaces
    .replace(/([A-Z])/g, " $1") // Add space before capital letters
    .replace(/^./, (str) => str.toUpperCase()) // Capitalize first letter
    .replace(/\s+/g, " ") // Remove extra spaces
    .trim();
};

/**
 * Formats a value for display - arrays are joined with commas
 * @param {any} value - The value to format
 * @returns {any} - Formatted value
 */
const formatValue = (value) => {
  if (Array.isArray(value)) {
    return value.join(", ");
  }
  return value;
};

/**
 * Converts a JSON object to an array of field objects for ResourceCard
 * @param {Object} jsonData - The JSON data to convert
 * @param {string} prefix - Optional prefix for the key (e.g., 'data')
 * @param {string} type - The type of field to create (default: 'static')
 * @param {boolean} copyable - Whether the field is copyable (default: false)
 * @returns {Array} - Array of field objects
 */
export const jsonToFields = (
  jsonData,
  prefix = "",
  type = "",
  copyable = false,
) => {
  if (!jsonData || typeof jsonData !== "object") {
    return [];
  }

  return Object.keys(jsonData).map((key) => {
    const fieldKey = prefix ? `${prefix}.${key}` : key;

    return {
      label: formatLabel(key),
      key: fieldKey,
      value: formatValue(jsonData[key]),
      type: type || undefined,
      copyable,
    };
  });
};
