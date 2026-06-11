import { exportFlags } from '../configDefaults';
import type { RestoreImportOptions } from '../types';

export const restoreImportFlags = exportFlags;

export const restoreModeOptions = [
  { key: '', labelKey: 'restoreModeApiDefault', danger: false },
  { key: 'append', labelKey: 'restoreModeAppend', danger: false },
  { key: 'merge', labelKey: 'restoreModeMerge', danger: false },
  { key: 'overwrite', labelKey: 'restoreModeOverwrite', danger: true },
  { key: 'clear', labelKey: 'restoreModeClear', danger: true }
];

export function defaultRestoreImportOptions(): RestoreImportOptions {
  return {
    mode: '',
    ...Object.fromEntries(restoreImportFlags.map((key) => [key, true]))
  };
}

export function restoreImportSelectedValues(value: RestoreImportOptions) {
  return restoreImportFlags.filter((key) => value[key] !== false);
}

export function restoreImportOptionsFromSelected(values: string[], mode: string): RestoreImportOptions {
  const selected = new Set(values);
  return {
    mode,
    ...Object.fromEntries(restoreImportFlags.map((key) => [key, selected.has(key)]))
  };
}
