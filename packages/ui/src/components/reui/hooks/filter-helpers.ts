import type {
  Filter,
  FilterFieldConfig,
  FilterFieldGroup,
  FilterFieldsConfig,
  FilterGroup,
} from "../filters";

export const isFieldGroup = <T = unknown>(
  item: FilterFieldConfig<T> | FilterFieldGroup<T>
): item is FilterFieldGroup<T> => {
  return "fields" in item && Array.isArray(item.fields);
};

export const isGroupLevelField = <T = unknown>(field: FilterFieldConfig<T>): boolean => {
  return Boolean(field.group && field.fields);
};

export const flattenFields = <T = unknown>(
  fields: FilterFieldsConfig<T>
): FilterFieldConfig<T>[] => {
  return fields.reduce<FilterFieldConfig<T>[]>((acc, item) => {
    if (isFieldGroup(item)) {
      return [...acc, ...item.fields];
    }
    if (isGroupLevelField(item)) {
      return [...acc, ...item.fields!];
    }
    return [...acc, item];
  }, []);
};

export const getFieldsMap = <T = unknown>(
  fields: FilterFieldsConfig<T>
): Record<string, FilterFieldConfig<T>> => {
  const flatFields = flattenFields(fields);
  return flatFields.reduce(
    (acc, field) => {
      if (field.key) {
        acc[field.key] = field;
      }
      return acc;
    },
    {} as Record<string, FilterFieldConfig<T>>
  );
};

export const createFilter = <T = unknown>(
  field: string,
  operator?: string,
  values: T[] = []
): Filter<T> => ({
  id: `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
  field,
  operator: operator || "is",
  values,
});

export const createFilterGroup = <T = unknown>(
  id: string,
  label: string,
  fields: FilterFieldConfig<T>[],
  initialFilters: Filter<T>[] = []
): FilterGroup<T> => ({
  id,
  label,
  filters: initialFilters,
  fields,
});
