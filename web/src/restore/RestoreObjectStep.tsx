import { Button, Card, Input } from 'animal-island-ui';
import { FileArchive, FolderOpen, Lock, RefreshCw } from 'lucide-react';

import { Field, NativeSelect } from '../components/ui';
import { formatBytes } from '../format';
import { useI18n } from '../i18n';
import type { RestoreDraft } from './types';
import type { RestoreDirectory, RestoreObject } from '../types';

export function RestoreObjectStep({ draft, directories, objects, selectedObject, locale, scanning, scanError, onDraft, onScan, onSelectDirectory }: {
  draft: RestoreDraft;
  directories: RestoreDirectory[];
  objects: RestoreObject[];
  selectedObject?: RestoreObject;
  locale: string;
  scanning: boolean;
  scanError: string;
  onDraft: (patch: Partial<RestoreDraft>) => void;
  onScan: () => void;
  onSelectDirectory: (directory: string) => void;
}) {
  const { t } = useI18n();
  const encrypted = selectedObject?.encrypted ?? draft.objectName.trim().endsWith('.zip.aes.json');
  const objectName = draft.objectName.trim();
  const invalidObject = objectName !== '' && !objectName.endsWith('.zip') && !objectName.endsWith('.zip.aes.json');
  return (
    <div className="restore-object-step">
      <div className="restore-scan-row">
        <Field label={t('restoreBrowsePrefix')} help={t('restoreBrowsePrefixTip')}>
          <Input value={draft.prefix} placeholder={t('restoreBrowsePrefixPlaceholder')} onChange={(event) => onDraft({ prefix: event.target.value, directory: '', objectName: '' })} allowClear />
        </Field>
        <Button type="primary" icon={<RefreshCw size={16} />} loading={scanning} onClick={onScan} disabled={scanning || !draft.targetKey}>
          {t('restoreScanDirectories')}
        </Button>
      </div>
      {scanError && <p className="error" role="alert">{scanError}</p>}
      {directories.length > 0 && (
        <Card color="app-blue" pattern="app-blue" className="restore-directory-card">
          <div className="settings-title"><FolderOpen size={18} />{t('restoreDirectory')}</div>
          <Field label={t('restoreDirectory')} help={t('restoreDirectoryTip')}>
            <NativeSelect
              value={draft.directory}
              onChange={onSelectDirectory}
              placeholder={t('restoreSelectDirectory')}
              options={directories.map((directory) => ({
                key: directory.name,
                label: `${directory.name} (${directory.object_count})`
              }))}
            />
          </Field>
        </Card>
      )}
      {objects.length > 0 && (
        <div className="restore-object-list" role="list">
          {objects.map((object) => (
            <button
              key={object.name}
              type="button"
              className={`restore-object-row ${draft.objectName === object.name ? 'selected' : ''}`}
              aria-pressed={draft.objectName === object.name}
              onClick={() => onDraft({ objectName: object.name })}
            >
              <FileArchive size={18} />
              <span>
                <strong>{object.name}</strong>
                <small>{formatBytes(object.size_bytes)} · {new Date(object.modified_at).toLocaleString(locale)}</small>
              </span>
              <em>{object.encrypted ? t('encrypted') : t('zip')}</em>
            </button>
          ))}
        </div>
      )}
      <Card color="app-yellow" pattern="app-yellow" className="restore-manual-object">
        <Field label={t('restoreManualObject')} help={t('restoreManualObjectTip')}>
          <Input value={draft.objectName} onChange={(event) => onDraft({ objectName: event.target.value })} allowClear aria-invalid={invalidObject} />
        </Field>
        {invalidObject && <p className="field-error" role="alert">{t('restoreObjectInvalid')}</p>}
      </Card>
      {encrypted && (
        <Card color="app-pink" pattern="app-pink" className="restore-password-box">
          <div className="settings-title"><Lock size={18} />{t('restoreEncryptedObject')}</div>
          <Field label={t('encryptionPassword')}>
            <Input
              type="password"
              value={draft.encryptionPassword}
              onChange={(event) => onDraft({ encryptionPassword: event.target.value })}
              autoComplete="current-password"
              allowClear
            />
          </Field>
        </Card>
      )}
    </div>
  );
}
