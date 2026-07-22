import type React from "react";

// Generic types for flexible filter system
export interface FilterOption<T = unknown> {
  value: T;
  label: string;
  icon?: React.ReactNode;
  metadata?: Record<string, unknown>;
  className?: string;
}

export interface FilterOperator {
  value: string;
  label: string;
  supportsMultiple?: boolean;
}

// Custom renderer props interface
export interface CustomRendererProps<T = unknown> {
  field: FilterFieldConfig<T>;
  values: T[];
  onChange: (values: T[]) => void;
  operator: string;
}

// Grouped field configuration interface
export interface FilterFieldGroup<T = unknown> {
  group?: string;
  fields: FilterFieldConfig<T>[];
}

// Union type for both flat and grouped field configurations
export type FilterFieldsConfig<T = unknown> = FilterFieldConfig<T>[] | FilterFieldGroup<T>[];

export interface FilterFieldConfig<T = unknown> {
  key?: string;
  label?: string;
  icon?: React.ReactNode;
  type?: "select" | "multiselect" | "text" | "custom" | "separator" | "toggle";
  // Group-level configuration
  group?: string;
  fields?: FilterFieldConfig<T>[];
  // Field-specific options
  options?: FilterOption<T>[];
  operators?: FilterOperator[];
  customRenderer?: (props: CustomRendererProps<T>) => React.ReactNode;
  customValueRenderer?: (values: T[], options: FilterOption<T>[]) => React.ReactNode;
  placeholder?: string;
  searchable?: boolean;
  maxSelections?: number;
  min?: number;
  max?: number;
  step?: number;
  prefix?: string | React.ReactNode;
  suffix?: string | React.ReactNode;
  pattern?: string;
  validation?: (value: unknown) => boolean | { valid: boolean; message?: string };
  allowCustomValues?: boolean;
  className?: string;
  menuPopupClassName?: string;
  // Grouping options (legacy support)
  groupLabel?: string;
  // Boolean field options
  onLabel?: string;
  offLabel?: string;
  // Input event handlers
  onInputChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
  // Default operator to use when creating a filter for this field
  defaultOperator?: string;
  // Controlled values support for this field
  value?: T[];
  onValueChange?: (values: T[]) => void;
}

export interface Filter<T = unknown> {
  id: string;
  field: string;
  operator: string;
  values: T[];
}

export interface FilterGroup<T = unknown> {
  id: string;
  label?: string;
  filters: Filter<T>[];
  fields: FilterFieldConfig<T>[];
}
