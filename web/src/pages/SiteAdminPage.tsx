import { useEffect, useState } from 'react';
import { Button, Card, Input, Modal } from 'animal-island-ui';
import { KeyRound, RefreshCw, Shield, Trash2, UserCog } from 'lucide-react';

import { api } from '../api';
import { Avatar, Field, ModalFooter, Panel, SwitchField } from '../components/ui';
import { useI18n } from '../i18n';
import type { AdminUser } from '../types';

type UserDraft = {
  username: string;
  email: string;
  display_name: string;
  password: string;
  is_admin: boolean;
};

export function SiteAdminPage({ currentTenant, locale, onCurrentUserRenamed }: { currentTenant: string; locale: string; onCurrentUserRenamed: () => void }) {
  const { t } = useI18n();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [editor, setEditor] = useState<{ tenant?: string; draft: UserDraft } | null>(null);
  const [passwordEditor, setPasswordEditor] = useState<{ tenant: string; username: string; password: string } | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AdminUser | null>(null);

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await api<{ users: AdminUser[] }>('/api/admin/users');
      setUsers(response.users ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('siteUsersLoadFailed'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const saveUser = async () => {
    if (!editor) return;
    setSaving(true);
    setError('');
    setMessage('');
    try {
      if (editor.tenant) {
        const saved = await api<AdminUser>(`/api/admin/users/${encodeURIComponent(editor.tenant)}`, {
          method: 'PUT',
          body: JSON.stringify({
            username: editor.draft.username,
            email: editor.draft.email,
            display_name: editor.draft.display_name,
            is_admin: editor.draft.is_admin
          })
        });
        const original = users.find((user) => user.tenant === editor.tenant);
        if (editor.tenant === currentTenant && (saved.tenant !== currentTenant || saved.username !== original?.username)) {
          onCurrentUserRenamed();
          return;
        }
        setMessage(t('siteUserSaved'));
      } else {
        await api<AdminUser>('/api/admin/users', {
          method: 'POST',
          body: JSON.stringify(editor.draft)
        });
        setMessage(t('siteUserCreated'));
      }
      setEditor(null);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t('siteUserSaveFailed'));
    } finally {
      setSaving(false);
    }
  };

  const resetPassword = async () => {
    if (!passwordEditor) return;
    setSaving(true);
    setError('');
    setMessage('');
    try {
      await api(`/api/admin/users/${encodeURIComponent(passwordEditor.tenant)}/password`, {
        method: 'PUT',
        body: JSON.stringify({ password: passwordEditor.password })
      });
      setPasswordEditor(null);
      setMessage(t('sitePasswordReset'));
    } catch (err) {
      setError(err instanceof Error ? err.message : t('sitePasswordResetFailed'));
    } finally {
      setSaving(false);
    }
  };

  const deleteUser = async () => {
    if (!deleteTarget) return;
    setSaving(true);
    setError('');
    setMessage('');
    try {
      await api(`/api/admin/users/${encodeURIComponent(deleteTarget.tenant)}`, { method: 'DELETE' });
      setDeleteTarget(null);
      setMessage(t('siteUserDeleted'));
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t('siteUserDeleteFailed'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Panel title={t('siteManagement')} actionLabel={t('addUser')} onAdd={() => setEditor({ draft: emptyUserDraft() })}>
      <div className="site-admin-toolbar">
        <Button type="default" icon={<RefreshCw size={16} />} loading={loading} onClick={() => { void load(); }}>{t('refresh')}</Button>
      </div>
      {(message || error) && <p className={`inline-notice ${error ? 'danger' : 'success'}`}>{error || message}</p>}
      {users.length === 0 ? (
        <Card color="app-yellow" pattern="app-yellow" className="empty">{t('noUsersYet')}</Card>
      ) : (
        <div className="grid-list">
          {users.map((user) => (
            <Card key={user.tenant} color={user.is_admin ? 'app-green' : 'app-blue'} pattern={user.is_admin ? 'app-green' : 'app-blue'} className="item user-card">
              <div className="user-card-head">
                <Avatar profile={user} />
                <div>
                  <h3>{user.display_name || user.username}</h3>
                  <code>{user.username}</code>
                  {user.email && <small>{user.email}</small>}
                </div>
              </div>
              <div className="user-card-meta">
                {user.is_admin && <span><Shield size={14} />{t('administrator')}</span>}
                <span>{t('createdAt')}: {new Date(user.created_at).toLocaleString(locale)}</span>
                <span>{user.oidc?.linked ? `${t('oidcLinked')}${user.oidc.email ? ` · ${user.oidc.email}` : ''}` : t('oidcNotLinked')}</span>
              </div>
              <div className="card-actions">
                <Button
                  type="default"
                  icon={<UserCog size={16} />}
                  onClick={() => setEditor({
                    tenant: user.tenant,
                    draft: {
                      username: user.username,
                      email: user.email ?? '',
                      display_name: user.display_name,
                      password: '',
                      is_admin: user.is_admin
                    }
                  })}
                >
                  {t('edit')}
                </Button>
                <Button type="default" icon={<KeyRound size={16} />} onClick={() => setPasswordEditor({ tenant: user.tenant, username: user.username, password: '' })}>
                  {t('resetPassword')}
                </Button>
                <Button type="default" danger icon={<Trash2 size={16} />} disabled={user.tenant === currentTenant} onClick={() => setDeleteTarget(user)}>
                  {t('delete')}
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}

      <UserEditorModal
        editor={editor}
        currentTenant={currentTenant}
        saving={saving}
        onChange={setEditor}
        onCancel={() => setEditor(null)}
        onSave={saveUser}
      />
      <PasswordModal
        editor={passwordEditor}
        saving={saving}
        onChange={setPasswordEditor}
        onCancel={() => setPasswordEditor(null)}
        onSave={resetPassword}
      />
      <DeleteUserModal
        user={deleteTarget}
        saving={saving}
        onCancel={() => setDeleteTarget(null)}
        onDelete={deleteUser}
      />
    </Panel>
  );
}

function UserEditorModal({ editor, currentTenant, saving, onChange, onCancel, onSave }: {
  editor: { tenant?: string; draft: UserDraft } | null;
  currentTenant: string;
  saving: boolean;
  onChange: (editor: { tenant?: string; draft: UserDraft } | null) => void;
  onCancel: () => void;
  onSave: () => void;
}) {
  const { t } = useI18n();
  if (!editor) return null;
  const isCurrentUser = editor.tenant === currentTenant;
  const set = (patch: Partial<UserDraft>) => onChange({ ...editor, draft: { ...editor.draft, ...patch } });
  return (
    <Modal
      open
      title={editor.tenant ? t('editUser') : t('addUser')}
      typewriter={false}
      width={560}
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={onSave} />}
    >
      <div className="editor-form">
        <Field label={t('username')}>
          <Input value={editor.draft.username} onChange={(event) => set({ username: event.target.value })} allowClear />
        </Field>
        <Field label={t('email')} help={t('emailAutoLinkTip')}>
          <Input value={editor.draft.email} onChange={(event) => set({ email: event.target.value })} allowClear />
        </Field>
        <Field label={t('nickname')}>
          <Input value={editor.draft.display_name} onChange={(event) => set({ display_name: event.target.value })} allowClear />
        </Field>
        {!editor.tenant && (
          <Field label={t('password')}>
            <Input type="password" value={editor.draft.password} onChange={(event) => set({ password: event.target.value })} autoComplete="new-password" allowClear />
          </Field>
        )}
        <SwitchField label={t('administrator')} checked={editor.draft.is_admin} onChange={(is_admin) => { if (!isCurrentUser) set({ is_admin }); }} />
        {isCurrentUser && <p className="muted">{t('currentUserAdminEditTip')}</p>}
      </div>
    </Modal>
  );
}

function PasswordModal({ editor, saving, onChange, onCancel, onSave }: {
  editor: { tenant: string; username: string; password: string } | null;
  saving: boolean;
  onChange: (editor: { tenant: string; username: string; password: string } | null) => void;
  onCancel: () => void;
  onSave: () => void;
}) {
  const { t } = useI18n();
  if (!editor) return null;
  return (
    <Modal
      open
      title={t('resetPassword')}
      typewriter={false}
      width={520}
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={onSave} />}
    >
      <div className="editor-form">
        <p className="muted">{t('resetPasswordFor').replace('{username}', editor.username)}</p>
        <Field label={t('newPassword')}>
          <Input type="password" value={editor.password} onChange={(event) => onChange({ ...editor, password: event.target.value })} autoComplete="new-password" allowClear />
        </Field>
      </div>
    </Modal>
  );
}

function DeleteUserModal({ user, saving, onCancel, onDelete }: {
  user: AdminUser | null;
  saving: boolean;
  onCancel: () => void;
  onDelete: () => void;
}) {
  const { t } = useI18n();
  if (!user) return null;
  return (
    <Modal
      open
      title={t('deleteUser')}
      typewriter={false}
      width={520}
      onClose={onCancel}
      footer={(
        <div className="modal-footer">
          <Button type="default" onClick={onCancel}>{t('cancel')}</Button>
          <Button type="default" danger loading={saving} onClick={onDelete}>{t('delete')}</Button>
        </div>
      )}
    >
      <div className="editor-form">
        <p>{t('deleteUserConfirm').replace('{username}', user.username)}</p>
        <p className="muted">{t('deleteUserDataTip')}</p>
      </div>
    </Modal>
  );
}

function emptyUserDraft(): UserDraft {
  return {
    username: '',
    email: '',
    display_name: '',
    password: '',
    is_admin: false
  };
}
