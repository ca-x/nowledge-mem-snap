import { useEffect, useState } from 'react';
import { Button, Card, Input } from 'animal-island-ui';
import { Settings } from 'lucide-react';

import { useI18n } from '../i18n';
import { Field, Panel } from '../components/ui';
import type { Config } from '../types';

export function SettingsPage({ cfg, saving, onSaveConfig }: {
  cfg: Config;
  saving: boolean;
  onSaveConfig: (cfg: Config) => void;
}) {
  const { t } = useI18n();
  const [historyLimit, setHistoryLimit] = useState(String(cfg.history_limit));
  const [historyDays, setHistoryDays] = useState(String(cfg.history_retention_days));

  useEffect(() => {
    setHistoryLimit(String(cfg.history_limit));
    setHistoryDays(String(cfg.history_retention_days));
  }, [cfg]);

  return (
    <Panel title={t('settings')}>
      <div className="settings-grid">
        <Card color="app-green" pattern="app-green" className="settings-card wide">
          <div className="settings-title"><Settings size={20} /> {t('history')}</div>
          <div className="editor-form">
            <Field label={t('keepLatestRuns')}>
              <Input type="number" min={1} value={historyLimit} onChange={(e) => setHistoryLimit(e.target.value)} />
            </Field>
            <Field label={t('keepRunHistoryDays')}>
              <Input type="number" min={1} value={historyDays} onChange={(e) => setHistoryDays(e.target.value)} />
            </Field>
            <Button
              type="primary"
              loading={saving}
              onClick={() => onSaveConfig({
                ...cfg,
                history_limit: Math.max(1, Number(historyLimit) || 100),
                history_retention_days: Math.max(1, Number(historyDays) || 180)
              })}
            >
              {t('saveHistorySettings')}
            </Button>
          </div>
        </Card>
      </div>
    </Panel>
  );
}
