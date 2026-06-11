import { Button, Checkbox } from 'animal-island-ui';
import { AlertTriangle } from 'lucide-react';

import { useI18n } from '../i18n';
import {
  defaultRestoreImportOptions,
  restoreImportOptionsFromSelected,
  restoreImportSelectedValues,
  restoreModeOptions,
  restoreImportFlags
} from './restoreDefaults';
import type { RestoreDraft } from './types';

export function RestoreOptionsStep({ draft, onDraft }: {
  draft: RestoreDraft;
  onDraft: (patch: Partial<RestoreDraft>) => void;
}) {
  const { t } = useI18n();
  const selectedValues = restoreImportSelectedValues(draft.importOptions);
  const mode = draft.importOptions.mode ?? '';
  const dangerousMode = mode === 'overwrite' || mode === 'clear';
  const setMode = (nextMode: string) => {
    onDraft({ importOptions: { ...draft.importOptions, mode: nextMode }, dangerousModeConfirmed: false });
  };
  return (
    <div className="restore-options-step">
      <div className="export-box">
        <div className="export-box-head">
          <span>{t('restoreImportMode')}</span>
          <Button type="default" size="small" onClick={() => onDraft({ importOptions: defaultRestoreImportOptions() })}>
            {t('restoreRecommendedDefaults')}
          </Button>
        </div>
        <div className="restore-mode-grid" role="radiogroup" aria-label={t('restoreImportMode')}>
          {restoreModeOptions.map((option) => (
            <button
              key={option.key || 'default'}
              type="button"
              role="radio"
              aria-checked={mode === option.key}
              className={`${mode === option.key ? 'selected' : ''} ${option.danger ? 'danger' : ''}`}
              onClick={() => setMode(option.key)}
            >
              <strong>{t(option.labelKey)}</strong>
              {option.danger && <span>{t('restoreDangerousMode')}</span>}
            </button>
          ))}
        </div>
        {dangerousMode && (
          <div className="restore-danger-confirm" role="alert">
            <AlertTriangle size={18} />
            <div>
              <strong>{t('restoreDangerousModeTitle')}</strong>
              <p>{t('restoreDangerousModeMessage')}</p>
              <Checkbox
                value={draft.dangerousModeConfirmed ? ['confirm'] : []}
                onChange={(values) => onDraft({ dangerousModeConfirmed: values.map(String).includes('confirm') })}
                options={[{ label: t('restoreDangerousModeConfirm'), value: 'confirm' }]}
              />
            </div>
          </div>
        )}
      </div>
      <div className="export-box">
        <div className="export-box-head">
          <span>{t('restoreImportContents')}</span>
          <span className="muted">{selectedValues.length}/{restoreImportFlags.length}</span>
        </div>
        <Checkbox
          className="choice-grid export-choice-grid"
          value={selectedValues}
          onChange={(values) => onDraft({ importOptions: restoreImportOptionsFromSelected(values.map(String), mode) })}
          options={restoreImportFlags.map((key) => ({ label: t(`export.${key}`), value: key }))}
        />
      </div>
    </div>
  );
}
