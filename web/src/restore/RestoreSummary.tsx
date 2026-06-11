import { Card } from 'animal-island-ui';
import { AlertTriangle, CheckCircle2 } from 'lucide-react';

import { formatBytes } from '../format';
import { useI18n } from '../i18n';
import { restoreImportFlags, restoreModeOptions } from './restoreDefaults';
import type { RestoreDraft } from './types';
import type { RestoreObject, Source, Target } from '../types';

export function RestoreSummary({ draft, targets, sources, selectedObject }: {
  draft: RestoreDraft;
  targets: Target[];
  sources: Source[];
  selectedObject?: RestoreObject;
}) {
  const { t } = useI18n();
  const target = targets.find((item) => item.key === draft.targetKey);
  const source = sources.find((item) => item.key === draft.destinationSourceKey);
  const objectName = draft.objectName.trim();
  const encrypted = selectedObject?.encrypted ?? objectName.endsWith('.zip.aes.json');
  const selectedFlags = restoreImportFlags.filter((key) => draft.importOptions[key] !== false);
  const mode = restoreModeOptions.find((item) => item.key === (draft.importOptions.mode ?? ''));
  const dangerousMode = draft.importOptions.mode === 'overwrite' || draft.importOptions.mode === 'clear';
  const warnings = [
    !target ? t('restoreSummaryMissingTarget') : '',
    !objectName ? t('restoreSummaryMissingObject') : '',
    encrypted && !draft.encryptionPassword ? t('restoreSummaryMissingPassword') : '',
    !source ? t('restoreSummaryMissingDestination') : '',
    dangerousMode && !draft.dangerousModeConfirmed ? t('restoreSummaryDangerousMode') : ''
  ].filter(Boolean);

  return (
    <Card color="app-yellow" pattern="app-yellow" className="restore-summary">
      <div className="restore-summary-head">
        <h3>{t('restoreSummary')}</h3>
        {warnings.length === 0 ? <CheckCircle2 size={18} /> : <AlertTriangle size={18} />}
      </div>
      <SummaryRow label={t('target')} value={target ? `${target.name} · ${target.type.toUpperCase()}` : t('notSet')} />
      <SummaryRow label={t('restoreObject')} value={objectName || t('notSet')} code />
      <SummaryRow
        label={t('restorePackage')}
        value={encrypted ? t('encryptedPackage') : t('plainPortableZip')}
      />
      {selectedObject && <SummaryRow label={t('restoreObjectSize')} value={formatBytes(selectedObject.size_bytes)} />}
      <SummaryRow label={t('restoreDestination')} value={source ? `${source.name} · ${source.nowledge_mem?.api_url ?? ''}` : t('notSet')} />
      <SummaryRow label={t('restoreImportMode')} value={mode ? t(mode.labelKey) : draft.importOptions.mode || t('restoreModeApiDefault')} />
      <SummaryRow label={t('restoreSelectedContents')} value={`${selectedFlags.length}/${restoreImportFlags.length}`} />
      {warnings.length > 0 && (
        <div className="restore-warnings">
          {warnings.map((warning) => <span key={warning}>{warning}</span>)}
        </div>
      )}
    </Card>
  );
}

function SummaryRow({ label, value, code }: { label: string; value: string; code?: boolean }) {
  return (
    <div className="restore-summary-row">
      <span>{label}</span>
      {code ? <code>{value}</code> : <strong>{value}</strong>}
    </div>
  );
}
