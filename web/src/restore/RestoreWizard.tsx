import { useEffect, useMemo, useState } from 'react';
import { Button } from 'animal-island-ui';
import { ChevronLeft, ChevronRight, UploadCloud } from 'lucide-react';

import { api } from '../api';
import { Panel } from '../components/ui';
import { useI18n } from '../i18n';
import type { RestoreDirectory, RestoreJob, RestoreObject } from '../types';
import { defaultRestoreImportOptions } from './restoreDefaults';
import { RestoreDestinationStep } from './RestoreDestinationStep';
import { RestoreObjectStep } from './RestoreObjectStep';
import { RestoreOptionsStep } from './RestoreOptionsStep';
import { RestoreProgressStep } from './RestoreProgressStep';
import { RestoreStepper, restoreSteps } from './RestoreStepper';
import { RestoreSummary } from './RestoreSummary';
import { RestoreTargetStep } from './RestoreTargetStep';
import type { RestoreDraft, RestoreStepKey, RestoreWizardProps } from './types';

export function RestoreWizard({ targets, sources, locale }: RestoreWizardProps) {
  const { t } = useI18n();
  const [activeIndex, setActiveIndex] = useState(0);
  const [maxStep, setMaxStep] = useState(0);
  const [directories, setDirectories] = useState<RestoreDirectory[]>([]);
  const [objects, setObjects] = useState<RestoreObject[]>([]);
  const [scanning, setScanning] = useState(false);
  const [scanError, setScanError] = useState('');
  const [starting, setStarting] = useState(false);
  const [job, setJob] = useState<RestoreJob | null>(null);
  const [jobError, setJobError] = useState('');
  const [draft, setDraft] = useState<RestoreDraft>({
    targetKey: targets[0]?.key ?? '',
    prefix: '',
    directory: '',
    objectName: '',
    encryptionPassword: '',
    destinationSourceKey: sources[0]?.key ?? '',
    importOptions: defaultRestoreImportOptions(),
    dangerousModeConfirmed: false
  });

  useEffect(() => {
    setDraft((current) => ({
      ...current,
      targetKey: current.targetKey || targets[0]?.key || '',
      destinationSourceKey: current.destinationSourceKey || sources[0]?.key || ''
    }));
  }, [targets, sources]);

  const activeStep = restoreSteps[activeIndex] as RestoreStepKey;
  const selectedObject = useMemo(() => objects.find((object) => object.name === draft.objectName), [objects, draft.objectName]);
  const objectEncrypted = selectedObject?.encrypted ?? draft.objectName.trim().endsWith('.zip.aes.json');
  const canMoveNext = canLeaveStep(activeStep, draft, objectEncrypted);
  const terminal = job && ['completed', 'failed', 'cancelled'].includes(job.state);

  const updateDraft = (patch: Partial<RestoreDraft>) => {
    const targetChanged = patch.targetKey !== undefined && patch.targetKey !== draft.targetKey;
    const prefixChanged = patch.prefix !== undefined && patch.prefix !== draft.prefix;
    if (targetChanged || prefixChanged) {
      setDirectories([]);
      setObjects([]);
      setScanError('');
    }
    setDraft((current) => {
      const next = { ...current, ...patch };
      if (targetChanged) next.prefix = '';
      if (targetChanged || prefixChanged) next.directory = '';
      if (targetChanged || prefixChanged) next.objectName = '';
      if (targetChanged || prefixChanged) next.encryptionPassword = '';
      if (patch.importOptions && !isDangerousMode(patch.importOptions.mode ?? '')) {
        next.dangerousModeConfirmed = false;
      }
      if (patch.objectName !== undefined && !patch.objectName.endsWith('.zip.aes.json')) {
        next.encryptionPassword = patch.encryptionPassword ?? '';
      }
      return next;
    });
  };

  const goTo = (index: number) => {
    const next = Math.max(0, Math.min(index, restoreSteps.length - 1));
    setActiveIndex(next);
    setMaxStep((value) => Math.max(value, next));
  };

  const scanDirectories = async () => {
    setScanError('');
    setScanning(true);
    try {
      const response = await api<{ directories: RestoreDirectory[]; objects: RestoreObject[] }>('/api/restore/browse', {
        method: 'POST',
        body: JSON.stringify({ target_key: draft.targetKey, prefix: draft.prefix })
      });
      const nextDirectories = response.directories ?? [];
      const rootObjects = response.objects ?? [];
      setDirectories(nextDirectories);
      if (nextDirectories.length > 0) {
        setObjects([]);
        updateDraft({ directory: '', objectName: '', encryptionPassword: '' });
      } else {
        setObjects(rootObjects);
        updateDraft({ directory: '', objectName: rootObjects[0]?.name ?? '' });
        if (rootObjects.length === 0) {
          setScanError(t('restoreNoDirectoriesFound'));
        }
      }
    } catch (err) {
      setScanError(err instanceof Error ? err.message : t('restoreScanFailed'));
    } finally {
      setScanning(false);
    }
  };

  const selectDirectory = async (directory: string) => {
    updateDraft({ directory, objectName: '', encryptionPassword: '' });
    if (!directory) {
      setObjects([]);
      return;
    }
    setScanError('');
    setScanning(true);
    try {
      const response = await api<{ objects: RestoreObject[] }>('/api/restore/objects', {
        method: 'POST',
        body: JSON.stringify({ target_key: draft.targetKey, prefix: directory })
      });
      const nextObjects = response.objects ?? [];
      setObjects(nextObjects);
      updateDraft({ objectName: nextObjects[0]?.name ?? '' });
      if (nextObjects.length === 0) {
        setScanError(t('restoreNoObjectsFound'));
      }
    } catch (err) {
      setScanError(err instanceof Error ? err.message : t('restoreScanFailed'));
    } finally {
      setScanning(false);
    }
  };

  useEffect(() => {
    if (activeStep !== 'object' || !draft.targetKey || directories.length > 0 || objects.length > 0 || scanning) {
      return;
    }
    scanDirectories();
  }, [activeStep, draft.targetKey]);

  const fetchJob = async (id: string) => {
    const next = await api<RestoreJob>(`/api/restore/jobs/${encodeURIComponent(id)}`);
    setJob(next);
    return next;
  };

  const startRestore = async () => {
    setStarting(true);
    setJobError('');
    try {
      const response = await api<{ job_id: string }>('/api/restore/jobs', {
        method: 'POST',
        body: JSON.stringify({
          target_key: draft.targetKey,
          object_name: draft.objectName,
          destination_source_key: draft.destinationSourceKey,
          encryption_password: draft.encryptionPassword,
          import_options: draft.importOptions
        })
      });
      goTo(4);
      await fetchJob(response.job_id);
    } catch (err) {
      setJobError(err instanceof Error ? err.message : t('restoreStartFailed'));
      goTo(4);
    } finally {
      setStarting(false);
    }
  };

  useEffect(() => {
    if (!job || terminal) return;
    const timer = window.setInterval(() => {
      fetchJob(job.id).catch((err) => setJobError(err instanceof Error ? err.message : t('restoreStatusFailed')));
    }, 1500);
    return () => window.clearInterval(timer);
  }, [job?.id, terminal, t]);

  const reset = () => {
    setActiveIndex(0);
    setMaxStep(0);
    setObjects([]);
    setDirectories([]);
    setScanError('');
    setJob(null);
    setJobError('');
    setStarting(false);
    setDraft({
      targetKey: targets[0]?.key ?? '',
      prefix: '',
      directory: '',
      objectName: '',
      encryptionPassword: '',
      destinationSourceKey: sources[0]?.key ?? '',
      importOptions: defaultRestoreImportOptions(),
      dangerousModeConfirmed: false
    });
  };

  return (
    <Panel title={t('restore')}>
      <div className="restore-shell">
        <RestoreStepper activeStep={activeStep} maxStep={maxStep} />
        <div className="restore-layout">
          <div className="restore-main">
            {activeStep === 'target' && <RestoreTargetStep draft={draft} targets={targets} onDraft={updateDraft} />}
            {activeStep === 'object' && (
              <RestoreObjectStep
                draft={draft}
                directories={directories}
                objects={objects}
                selectedObject={selectedObject}
                locale={locale}
                scanning={scanning}
                scanError={scanError}
                onDraft={updateDraft}
                onScan={scanDirectories}
                onSelectDirectory={selectDirectory}
              />
            )}
            {activeStep === 'destination' && <RestoreDestinationStep draft={draft} sources={sources} onDraft={updateDraft} />}
            {activeStep === 'options' && <RestoreOptionsStep draft={draft} onDraft={updateDraft} />}
            {activeStep === 'progress' && (
              <RestoreProgressStep
                job={job}
                jobError={jobError}
                starting={starting}
                onStart={startRestore}
                onReset={reset}
              />
            )}
            <div className="restore-footer">
              {activeIndex > 0 && activeStep !== 'progress' && (
                <Button type="default" icon={<ChevronLeft size={16} />} onClick={() => goTo(activeIndex - 1)}>
                  {t('previous')}
                </Button>
              )}
              {activeIndex < 3 && (
                <Button type="primary" icon={<ChevronRight size={16} />} disabled={!canMoveNext} onClick={() => goTo(activeIndex + 1)}>
                  {t('next')}
                </Button>
              )}
              {activeStep === 'options' && (
                <Button type="primary" icon={<UploadCloud size={16} />} loading={starting} disabled={!canMoveNext || starting} onClick={startRestore}>
                  {t('restoreStart')}
                </Button>
              )}
            </div>
          </div>
          <aside className="restore-aside">
            <RestoreSummary draft={draft} targets={targets} sources={sources} selectedObject={selectedObject} />
          </aside>
        </div>
      </div>
    </Panel>
  );
}

function canLeaveStep(step: RestoreStepKey, draft: RestoreDraft, objectEncrypted: boolean) {
  switch (step) {
    case 'target':
      return Boolean(draft.targetKey);
    case 'object':
      return isImportableObject(draft.objectName) && (!objectEncrypted || Boolean(draft.encryptionPassword.trim()));
    case 'destination':
      return Boolean(draft.destinationSourceKey);
    case 'options':
      return Boolean(
        draft.targetKey &&
        draft.destinationSourceKey &&
        isImportableObject(draft.objectName) &&
        (!objectEncrypted || draft.encryptionPassword.trim()) &&
        (!isDangerousMode(draft.importOptions.mode ?? '') || draft.dangerousModeConfirmed)
      );
    default:
      return false;
  }
}

function isImportableObject(value: string) {
  const name = value.trim();
  return name.endsWith('.zip') || name.endsWith('.zip.aes.json');
}

function isDangerousMode(mode: string) {
  return mode === 'overwrite' || mode === 'clear';
}
