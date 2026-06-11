import type { RestoreImportOptions, RestoreJob, RestoreObject, Source, Target } from '../types';

export type RestoreStepKey = 'target' | 'object' | 'destination' | 'options' | 'progress';

export type RestoreDraft = {
  targetKey: string;
  prefix: string;
  objectName: string;
  encryptionPassword: string;
  destinationSourceKey: string;
  importOptions: RestoreImportOptions;
  dangerousModeConfirmed: boolean;
};

export type RestoreWizardProps = {
  targets: Target[];
  sources: Source[];
  locale: string;
};

export type RestoreStepProps = {
  draft: RestoreDraft;
  targets: Target[];
  sources: Source[];
  objects: RestoreObject[];
  selectedObject?: RestoreObject;
  locale: string;
  scanning: boolean;
  scanError: string;
  starting: boolean;
  job: RestoreJob | null;
  jobError: string;
  onDraft: (patch: Partial<RestoreDraft>) => void;
  onScan: () => void;
  onStart: () => void;
  onReset: () => void;
};
