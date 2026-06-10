import { useEffect, useState } from 'react';
import { Button, Checkbox, Input, Modal, Radio } from 'animal-island-ui';
import { CheckCircle2, ServerCog, XCircle } from 'lucide-react';

import { api } from '../api';
import { defaultDirectory, defaultNowledge } from '../configDefaults';
import { useI18n } from '../i18n';
import { downloadSourceTest, sourceTestMessage } from '../sourceTest';
import { Field, FormGrid, ModalFooter, NativeSelect, SwitchField, Tip } from '../components/ui';
import type { Editor, Source, SourceRoot, SourceType, TestResult } from '../types';

export function SourceModal({ editor, roots, saving, onChange, onCancel, onSave }: {
  editor: Editor<Source> | null;
  roots: SourceRoot[];
  saving: boolean;
  onChange: (next: Editor<Source> | null) => void;
  onCancel: () => void;
  onSave: (editor: Editor<Source>) => void;
}) {
  const { t } = useI18n();
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [testing, setTesting] = useState(false);
  const [downloadDuringTest, setDownloadDuringTest] = useState(false);

  useEffect(() => {
    setTestResult(null);
    setTesting(false);
    setDownloadDuringTest(false);
  }, [editor?.index]);

  if (!editor) return null;
  const source = editor.value;
  const setSource = (value: Source) => onChange({ ...editor, value });
  const test = async () => {
    setTesting(true);
    setTestResult(null);
    try {
      const result = downloadDuringTest
        ? await downloadSourceTest(source, t)
        : await api<TestResult>('/api/sources/test', { method: 'POST', body: JSON.stringify(source) });
      setTestResult(result);
    } catch (err) {
      setTestResult({ ok: false, message: err instanceof Error ? err.message : t('testFailed') });
    } finally {
      setTesting(false);
    }
  };
  return (
    <Modal
      open
      title={editor.index < 0 ? t('addSourceTitle') : t('editSourceTitle')}
      typewriter={false}
      width={680}
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={() => onSave(editor)} />}
    >
      <SourceForm value={source} roots={roots} onChange={setSource} />
      <div className="test-strip">
        <Button type="default" icon={<ServerCog size={16} />} loading={testing} onClick={test}>{t('testSource')}</Button>
        <Checkbox
          className="test-download-choice"
          value={downloadDuringTest ? ['download'] : []}
          onChange={(values) => setDownloadDuringTest(values.map(String).includes('download'))}
          options={[{ label: t('downloadRealExport'), value: 'download' }]}
        />
        <Tip content={t('downloadRealExportTip')} />
        {testResult && (
          <span className={`test-result ${testResult.ok ? 'success' : 'danger'}`}>
            {testResult.ok ? <CheckCircle2 size={16} /> : <XCircle size={16} />}
            {sourceTestMessage(testResult, t)}
          </span>
        )}
      </div>
    </Modal>
  );
}

function SourceForm({ value, roots, onChange }: { value: Source; roots: SourceRoot[]; onChange: (value: Source) => void }) {
  const { t } = useI18n();
  const nowledge = defaultNowledge(value);
  const directory = defaultDirectory(value, roots);
  const set = (patch: Partial<Source>) => onChange({ ...value, ...patch });
  const setNowledge = (patch: Partial<NonNullable<Source['nowledge_mem']>>) => set({ nowledge_mem: { ...nowledge, ...patch } });
  const setDirectory = (patch: Partial<NonNullable<Source['directory']>>) => set({ directory: { ...directory, ...patch } });

  return (
    <div className="editor-form">
      <Field label={t('name')}>
        <Input value={value.name} onChange={(e) => set({ name: e.target.value })} allowClear />
      </Field>
      <SwitchField label={t('enabled')} checked={value.enabled} onChange={(enabled) => set({ enabled })} />
      <Field label={t('type')}>
        <Radio
          value={value.type}
          onChange={(next) => set({ type: String(next) as SourceType })}
          options={[
            { label: t('nowledgeMemApi'), value: 'nowledgemem_api' },
            { label: t('directorySource'), value: 'directory' }
          ]}
        />
      </Field>
      {value.type === 'nowledgemem_api' ? (
        <FormGrid>
          <Field label={t('apiUrl')}>
            <Input value={nowledge.api_url} onChange={(e) => setNowledge({ api_url: e.target.value })} allowClear />
          </Field>
          <Field label={t('apiKey')} help={t('secretPreserveHelp')}>
            <Input type="password" value={nowledge.api_key ?? ''} onChange={(e) => setNowledge({ api_key: e.target.value })} allowClear />
          </Field>
        </FormGrid>
      ) : (
        <FormGrid>
          <Field label={t('allowedRoot')}>
            <NativeSelect
              value={directory.root_key || roots[0]?.key || ''}
              onChange={(rootKey) => {
                const root = roots.find((item) => item.key === rootKey);
                setDirectory({ root_key: rootKey, path: root?.path ?? directory.path });
              }}
              options={roots.map((root) => ({ key: root.key, label: `${root.name} · ${root.path}` }))}
              placeholder={t('noAllowedRoots')}
              disabled={roots.length === 0}
            />
          </Field>
          <Field label={t('directoryPath')}>
            <Input value={directory.path} onChange={(e) => setDirectory({ path: e.target.value })} allowClear />
          </Field>
        </FormGrid>
      )}
      <Field label={t('remark')}>
        <Input value={value.remark ?? ''} onChange={(e) => set({ remark: e.target.value })} allowClear />
      </Field>
    </div>
  );
}
