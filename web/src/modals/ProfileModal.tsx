import { useEffect, useState } from 'react';
import { Button, Input, Modal } from 'animal-island-ui';

import { useI18n } from '../i18n';
import { Avatar, Field, ModalFooter } from '../components/ui';
import type { Profile } from '../types';

export function ProfileModal({ open, profile, saving, onCancel, onSave }: {
  open: boolean;
  profile: Profile;
  saving: boolean;
  onCancel: () => void;
  onSave: (profile: Profile) => Promise<boolean>;
}) {
  const { t } = useI18n();
  const [draftProfile, setDraftProfile] = useState(profile);

  useEffect(() => {
    setDraftProfile(profile);
  }, [profile, open]);

  const uploadAvatar = async (file?: File) => {
    if (!file) return;
    if (!file.type.startsWith('image/')) return;
    const dataURL = await readFileAsDataURL(file);
    setDraftProfile({ ...draftProfile, avatar_url: dataURL });
  };

  const save = async () => {
    const ok = await onSave(draftProfile);
    if (ok) onCancel();
  };

  if (!open) return null;
  return (
    <Modal
      open
      title={t('profile')}
      typewriter={false}
      width={560}
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={() => { void save(); }} />}
    >
      <div className="editor-form">
        <div className="avatar-edit">
          <Avatar profile={draftProfile} />
          <div>
            <input id="profile-avatar-upload" className="file-input" type="file" accept="image/*" onChange={(event) => uploadAvatar(event.target.files?.[0])} />
            <Button type="default" onClick={() => document.getElementById('profile-avatar-upload')?.click()}>{t('uploadImage')}</Button>
          </div>
        </div>
        <Field label={t('nickname')}>
          <Input value={draftProfile.display_name} onChange={(e) => setDraftProfile({ ...draftProfile, display_name: e.target.value })} allowClear />
        </Field>
        <Field label={t('avatarUrlOrBase64')}>
          <Input value={draftProfile.avatar_url} onChange={(e) => setDraftProfile({ ...draftProfile, avatar_url: e.target.value })} allowClear />
        </Field>
      </div>
    </Modal>
  );
}

function readFileAsDataURL(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || ''));
    reader.onerror = () => reject(reader.error ?? new Error('failed to read file'));
    reader.readAsDataURL(file);
  });
}
