import React from 'react';
import { Button, Card, Switch, Title, Tooltip } from 'animal-island-ui';
import { Pencil, Plus, Trash2 } from 'lucide-react';

import { useI18n } from '../i18n';
import type { Profile } from '../types';

export function Avatar({ profile }: { profile: Pick<Profile, 'display_name' | 'username' | 'avatar_url'> }) {
  const label = (profile.display_name || profile.username || 'U').trim();
  if (profile.avatar_url) {
    return <img src={profile.avatar_url} className="avatar" alt={label} />;
  }
  return <div className="avatar fallback">{label.slice(0, 1).toUpperCase()}</div>;
}

export function Panel({ title, actionLabel, onAdd, children }: { title: string; actionLabel?: string; onAdd?: () => void; children: React.ReactNode }) {
  return (
    <section className="panel">
      <div className="panel-head">
        <Title size="small" color="app-teal">{title}</Title>
        {actionLabel && <Button type="primary" icon={<Plus size={16} />} onClick={onAdd}>{actionLabel}</Button>}
      </div>
      {children}
    </section>
  );
}

export function CardActions({ onEdit, onDelete }: { onEdit: () => void; onDelete: () => void }) {
  const { t } = useI18n();
  return (
    <div className="card-actions">
      <Button type="default" icon={<Pencil size={16} />} onClick={onEdit}>{t('edit')}</Button>
      <Button type="default" danger icon={<Trash2 size={16} />} onClick={onDelete}>{t('delete')}</Button>
    </div>
  );
}

export function ModalFooter({ saving, onCancel, onSave }: { saving: boolean; onCancel: () => void; onSave: () => void }) {
  const { t } = useI18n();
  return (
    <div className="modal-footer">
      <Button type="default" onClick={onCancel}>{t('cancel')}</Button>
      <Button type="primary" loading={saving} onClick={onSave}>{t('save')}</Button>
    </div>
  );
}

export function NativeSelect({ value, onChange, options, disabled, placeholder }: {
  value: string;
  onChange: (value: string) => void;
  options: Array<{ key: string; label: string }>;
  disabled?: boolean;
  placeholder?: string;
}) {
  return (
    <select className="native-select" value={value} disabled={disabled} onChange={(event) => onChange(event.target.value)}>
      {placeholder && <option value="">{placeholder}</option>}
      {options.map((option) => (
        <option key={option.key} value={option.key}>{option.label}</option>
      ))}
    </select>
  );
}

export function Field({ label, help, children }: { label: string; help?: string; children: React.ReactNode }) {
  return (
    <label className="field">
      <span className="label-with-tip">{label}{help && <Tip content={help} />}</span>
      {children}
    </label>
  );
}

export function Tip({ content }: { content: React.ReactNode }) {
  return (
    <Tooltip title={content} placement="top" trigger="click" variant="island" bordered>
      <button type="button" className="tip-button" aria-label="Info">?</button>
    </Tooltip>
  );
}

export function SwitchField({ label, checked, onChange }: { label: string; checked: boolean; onChange: (checked: boolean) => void }) {
  const { t } = useI18n();
  return (
    <div className="switch-field">
      <span>{label}</span>
      <Switch checked={checked} onChange={onChange} checkedChildren={t('on')} unCheckedChildren={t('off')} />
    </div>
  );
}

export function FormGrid({ children }: { children: React.ReactNode }) {
  return <div className="form-grid">{children}</div>;
}

export function Empty({ text }: { text: string }) {
  return <Card color="app-yellow" pattern="app-yellow" className="empty">{text}</Card>;
}
