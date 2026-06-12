import { Empty, Panel } from '../components/ui';
import { useI18n } from '../i18n';
import { RestoreWizard } from '../restore/RestoreWizard';
import type { Source, Target } from '../types';

export function RestorePage({ targets, sources, locale }: {
  targets: Target[];
  sources: Source[];
  locale: string;
}) {
  const { t } = useI18n();
  const restoreTargets = targets.filter((target) => target.enabled && ['s3', 'webdav', 'gcs', 'sftp'].includes(target.type));
  const restoreSources = sources.filter((source) => source.enabled && source.type === 'nowledgemem_api');
  if (restoreTargets.length === 0 || restoreSources.length === 0) {
    return (
      <Panel title={t('restore')}>
        <Empty text={restoreTargets.length === 0 ? t('restoreNoTargets') : t('restoreNoSources')} />
      </Panel>
    );
  }
  return <RestoreWizard targets={restoreTargets} sources={restoreSources} locale={locale} />;
}
