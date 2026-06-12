package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ca-x/nowledge-mem-snap/internal/persist"
	"github.com/ca-x/nowledge-mem-snap/internal/persist/ent"
	"github.com/ca-x/nowledge-mem-snap/internal/persist/ent/runrecord"
	"github.com/ca-x/nowledge-mem-snap/internal/persist/ent/systemconfig"
	"github.com/ca-x/nowledge-mem-snap/internal/persist/ent/tenantconfig"
	"github.com/ca-x/nowledge-mem-snap/internal/persist/ent/user"
)

const (
	DefaultPort = 14335

	DefaultSourceKey         = "7f9f4bc7-6245-4d20-9f45-2e1837e7a901"
	DefaultDailyScheduleKey  = "3dd785a4-3bfc-4be1-a412-60176b879f77"
	DefaultWeeklyScheduleKey = "6df2e7f1-8f64-4072-9823-0c249a422a81"
	DefaultExportOptionKey   = "9caaf0b1-a0f1-4ab7-8a28-2f5c0e2c45f9"
	DefaultBackupStrategyKey = "2b41d8a7-87c2-4b8a-913a-6b6a41b611b7"
	DefaultTaskKey           = "018ff3c8-a1ec-74f8-9381-fc7d6fb17f51"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

type Config struct {
	Listen               ListenConfig           `json:"listen"`
	Auth                 AuthConfig             `json:"auth"`
	Runtime              RuntimeConfig          `json:"runtime,omitempty"`
	TaskRuntime          map[string]TaskRuntime `json:"task_runtime,omitempty"`
	Sources              []SourceConfig         `json:"sources"`
	Schedules            []ScheduleConfig       `json:"schedules"`
	Targets              []TargetConfig         `json:"targets"`
	ExportOptions        []ExportOptionConfig   `json:"export_options"`
	BackupStrategies     []BackupStrategyConfig `json:"backup_strategies"`
	Tasks                []TaskConfig           `json:"tasks"`
	HistoryLimit         int                    `json:"history_limit"`
	HistoryRetentionDays int                    `json:"history_retention_days"`
}

type UserConfig struct {
	Sources              []SourceConfig         `json:"sources"`
	Schedules            []ScheduleConfig       `json:"schedules"`
	Targets              []TargetConfig         `json:"targets"`
	ExportOptions        []ExportOptionConfig   `json:"export_options"`
	BackupStrategies     []BackupStrategyConfig `json:"backup_strategies"`
	Tasks                []TaskConfig           `json:"tasks"`
	HistoryLimit         int                    `json:"history_limit"`
	HistoryRetentionDays int                    `json:"history_retention_days"`
}

type ListenConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	BasePath string `json:"base_path,omitempty"`
}

type AuthConfig struct {
	Username         string     `json:"username"`
	PasswordEnv      string     `json:"password_env"`
	SessionSecretEnv string     `json:"session_secret_env"`
	OIDC             OIDCConfig `json:"oidc"`
}

type RuntimeConfig struct {
	Timezone      string `json:"timezone,omitempty"`
	TimezoneLabel string `json:"timezone_label,omitempty"`
}

const (
	TaskRuntimeStatusScheduled        = "scheduled"
	TaskRuntimeStatusRunning          = "running"
	TaskRuntimeStatusDisabled         = "disabled"
	TaskRuntimeStatusScheduleDisabled = "schedule_disabled"
	TaskRuntimeStatusMissingSchedule  = "missing_schedule"
	TaskRuntimeStatusInvalidSchedule  = "invalid_schedule"
)

type TaskRuntime struct {
	Status    string     `json:"status"`
	NextRunAt *time.Time `json:"next_run_at,omitempty"`
}

type OIDCConfig struct {
	Enabled         bool     `json:"enabled"`
	IssuerURL       string   `json:"issuer_url"`
	ClientID        string   `json:"client_id"`
	ClientSecretEnv string   `json:"client_secret_env"`
	ClientSecret    string   `json:"-"`
	RedirectURL     string   `json:"redirect_url"`
	Scopes          []string `json:"scopes"`
	AllowedEmails   []string `json:"allowed_emails"`
	AllowedDomains  []string `json:"allowed_domains"`
}

type NowledgeConfig struct {
	APIURL    string        `json:"api_url"`
	APIKey    string        `json:"api_key,omitempty"`
	APIKeyEnv string        `json:"api_key_env,omitempty"`
	Timeout   time.Duration `json:"-"`
}

type ExportConfig struct {
	IncludeMemories             *bool `json:"include_memories,omitempty"`
	IncludeThreads              *bool `json:"include_threads,omitempty"`
	IncludeMessages             *bool `json:"include_messages,omitempty"`
	IncludeEntities             *bool `json:"include_entities,omitempty"`
	IncludeLabels               *bool `json:"include_labels,omitempty"`
	IncludeSources              *bool `json:"include_sources,omitempty"`
	IncludeCommunities          *bool `json:"include_communities,omitempty"`
	IncludeSkills               *bool `json:"include_skills,omitempty"`
	IncludeEdges                *bool `json:"include_edges,omitempty"`
	IncludeWorkingMemory        *bool `json:"include_working_memory,omitempty"`
	IncludeWorkingMemoryArchive *bool `json:"include_working_memory_archive,omitempty"`
	IncludeSourceFiles          *bool `json:"include_source_files,omitempty"`
}

type ExportOptionConfig struct {
	Key    string       `json:"key"`
	Name   string       `json:"name"`
	Export ExportConfig `json:"export"`
}

type BackupStrategyConfig struct {
	Key       string          `json:"key"`
	Name      string          `json:"name"`
	Retention RetentionConfig `json:"retention"`
}

type SourceConfig struct {
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Remark      string          `json:"remark,omitempty"`
	Enabled     bool            `json:"enabled"`
	Type        string          `json:"type"`
	NowledgeMem NowledgeConfig  `json:"nowledge_mem,omitempty"`
	Directory   DirectorySource `json:"directory,omitempty"`
}

type DirectorySource struct {
	Path    string `json:"path"`
	RootKey string `json:"root_key,omitempty"`
}

type DirectoryRoot struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Path string `json:"path"`
}

type ScheduleConfig struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"`
	Time    string `json:"time"`
	Weekday string `json:"weekday,omitempty"`
	RunAt   string `json:"run_at,omitempty"`
}

type TargetConfig struct {
	Key     string       `json:"key"`
	Name    string       `json:"name"`
	Enabled bool         `json:"enabled"`
	Type    string       `json:"type"`
	S3      S3Config     `json:"s3,omitempty"`
	WebDAV  WebDAVConfig `json:"webdav,omitempty"`
	GCS     GCSConfig    `json:"gcs,omitempty"`
	SFTP    SFTPConfig   `json:"sftp,omitempty"`
}

type S3Config struct {
	EndpointURL        string `json:"endpoint_url"`
	Region             string `json:"region"`
	PathStyle          bool   `json:"path_style"`
	BucketName         string `json:"bucket_name"`
	RootPrefix         string `json:"root_prefix"`
	AccessKeyID        string `json:"access_key_id"`
	SecretAccessKey    string `json:"secret_access_key,omitempty"`
	SecretAccessKeyEnv string `json:"secret_access_key_env,omitempty"`
}

type WebDAVConfig struct {
	URL         string `json:"url"`
	RootPrefix  string `json:"root_prefix"`
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	PasswordEnv string `json:"password_env,omitempty"`
}

type GCSConfig struct {
	BucketName         string `json:"bucket_name"`
	RootPrefix         string `json:"root_prefix"`
	CredentialsJSON    string `json:"credentials_json,omitempty"`
	CredentialsJSONEnv string `json:"credentials_json_env,omitempty"`
}

type SFTPConfig struct {
	Host                    string `json:"host"`
	Port                    int    `json:"port"`
	RootPrefix              string `json:"root_prefix"`
	Username                string `json:"username"`
	Password                string `json:"password,omitempty"`
	PasswordEnv             string `json:"password_env,omitempty"`
	PrivateKey              string `json:"private_key,omitempty"`
	PrivateKeyEnv           string `json:"private_key_env,omitempty"`
	PrivateKeyPassphrase    string `json:"private_key_passphrase,omitempty"`
	PrivateKeyPassphraseEnv string `json:"private_key_passphrase_env,omitempty"`
	HostKeySHA256           string `json:"host_key_sha256,omitempty"`
	InsecureIgnoreHostKey   bool   `json:"insecure_ignore_host_key,omitempty"`
}

type TaskConfig struct {
	Key               string           `json:"key"`
	Name              string           `json:"name"`
	Enabled           bool             `json:"enabled"`
	SourceKey         string           `json:"source_key"`
	ScheduleKey       string           `json:"schedule_key"`
	TargetKeys        []string         `json:"target_keys"`
	ExportOptionKey   string           `json:"export_option_key"`
	BackupStrategyKey string           `json:"backup_strategy_key"`
	ObjectPrefix      string           `json:"object_prefix"`
	Encryption        EncryptionConfig `json:"encryption"`
	Retention         RetentionConfig  `json:"-"`
}

type EncryptionConfig struct {
	Enabled     bool   `json:"enabled"`
	Password    string `json:"password,omitempty"`
	PasswordEnv string `json:"password_env,omitempty"`
}

type RetentionConfig struct {
	Mode       string `json:"mode"`
	KeepLast   int    `json:"keep_last,omitempty"`
	KeepDays   int    `json:"keep_days,omitempty"`
	KeepAfter  string `json:"keep_after,omitempty"`
	KeepBefore string `json:"keep_before,omitempty"`
}

type Store struct {
	client *ent.Client
	path   string
	err    error
}

type UserRecord struct {
	Tenant       string    `json:"tenant"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	AvatarURL    string    `json:"avatar_url"`
	OIDCIssuer   string    `json:"oidc_issuer,omitempty"`
	OIDCSubject  string    `json:"-"`
	OIDCEmail    string    `json:"oidc_email,omitempty"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
}

type Profile struct {
	Tenant      string       `json:"tenant"`
	Username    string       `json:"username"`
	Email       string       `json:"email,omitempty"`
	DisplayName string       `json:"display_name"`
	AvatarURL   string       `json:"avatar_url"`
	IsAdmin     bool         `json:"is_admin"`
	OIDC        OIDCIdentity `json:"oidc"`
}

type OIDCIdentity struct {
	Linked bool   `json:"linked"`
	Issuer string `json:"issuer,omitempty"`
	Email  string `json:"email,omitempty"`
}

type OIDCProfile struct {
	Issuer            string
	Subject           string
	Email             string
	EmailVerified     bool
	Username          string
	DisplayName       string
	AvatarURL         string
	PreferredUsername string
	Nickname          string
}

type UserUpdate struct {
	Username    string
	Email       string
	DisplayName string
	IsAdmin     bool
}

type AdminUser struct {
	Tenant      string       `json:"tenant"`
	Username    string       `json:"username"`
	Email       string       `json:"email,omitempty"`
	DisplayName string       `json:"display_name"`
	AvatarURL   string       `json:"avatar_url"`
	IsAdmin     bool         `json:"is_admin"`
	OIDC        OIDCIdentity `json:"oidc"`
	CreatedAt   time.Time    `json:"created_at"`
}

func NewStore(dataDir string) *Store {
	client, err := persist.OpenClient(dataDir)
	if err != nil {
		return &Store{path: persist.DBPath(dataDir), err: err}
	}
	return &Store{client: client, path: persist.DBPath(dataDir)}
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Client() *ent.Client {
	return s.client
}

func (s *Store) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}

func (s *Store) BootstrapFromEnv() error {
	if s.err != nil {
		return s.err
	}
	count, err := s.UserCount()
	if err != nil || count > 0 {
		return err
	}
	username := strings.TrimSpace(os.Getenv("NMEM_SNAP_ADMIN_USERNAME"))
	password := os.Getenv("NMEM_SNAP_ADMIN_PASSWORD")
	if username == "" {
		username = strings.TrimSpace(os.Getenv("NMEM_SNAP_AUTH_USERNAME"))
	}
	if password == "" {
		password = os.Getenv("NMEM_SNAP_PASSWORD")
	}
	if username == "" || password == "" {
		return nil
	}
	return s.CreateLocalUser(username, password, true)
}

func (s *Store) SetupRequired() (bool, error) {
	count, err := s.UserCount()
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func (s *Store) UserCount() (int, error) {
	return s.client.User.Query().Count(context.Background())
}

func (s *Store) CreateLocalUser(username, password string, admin bool) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}
	tenant := TenantKey(username)
	if tenant == "" {
		return fmt.Errorf("username cannot produce a valid tenant key")
	}
	email := NormalizeEmail(username)
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	ctx := context.Background()
	existing, err := s.client.User.Query().Where(user.Username(username)).Only(ctx)
	if ent.IsNotFound(err) {
		create := s.client.User.Create().
			SetTenant(tenant).
			SetUsername(username).
			SetPasswordHash(hash).
			SetDisplayName(username).
			SetIsAdmin(admin).
			SetCreatedAt(time.Now().UTC())
		if email != "" {
			create.SetEmail(email)
		}
		err = create.Exec(ctx)
	} else if err == nil {
		update := existing.Update().
			SetPasswordHash(hash).
			SetIsAdmin(admin)
		if email != "" {
			update.SetEmail(email)
		}
		err = update.Exec(ctx)
	}
	if err != nil {
		return err
	}
	_, _ = s.LoadUser(tenant)
	return nil
}

func (s *Store) CreateUser(username, password, displayName, email string, admin bool) (AdminUser, error) {
	username = strings.TrimSpace(username)
	displayName = strings.TrimSpace(displayName)
	email = strings.TrimSpace(email)
	if username == "" {
		return AdminUser{}, fmt.Errorf("username is required")
	}
	if password == "" {
		return AdminUser{}, fmt.Errorf("password is required")
	}
	tenant := TenantKey(username)
	if tenant == "" {
		return AdminUser{}, fmt.Errorf("username cannot produce a valid tenant key")
	}
	if displayName == "" {
		displayName = username
	}
	normalizedEmail, err := NormalizeOptionalEmail(email)
	if err != nil {
		return AdminUser{}, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return AdminUser{}, err
	}
	ctx := context.Background()
	if exists, err := s.client.User.Query().Where(user.Username(username)).Exist(ctx); err != nil {
		return AdminUser{}, err
	} else if exists {
		return AdminUser{}, fmt.Errorf("username %q already exists", username)
	}
	if exists, err := s.client.User.Query().Where(user.Tenant(tenant)).Exist(ctx); err != nil {
		return AdminUser{}, err
	} else if exists {
		return AdminUser{}, fmt.Errorf("tenant %q already exists", tenant)
	}
	if err := s.ensureUniqueEmail(ctx, normalizedEmail, ""); err != nil {
		return AdminUser{}, err
	}
	create := s.client.User.Create().
		SetTenant(tenant).
		SetUsername(username).
		SetPasswordHash(hash).
		SetDisplayName(displayName).
		SetIsAdmin(admin).
		SetCreatedAt(time.Now().UTC())
	if normalizedEmail != "" {
		create.SetEmail(normalizedEmail)
	}
	row, err := create.Save(ctx)
	if err != nil {
		return AdminUser{}, err
	}
	if _, err := s.LoadUser(tenant); err != nil {
		return AdminUser{}, err
	}
	return adminUserFromEnt(row), nil
}

func (s *Store) VerifyLocalUser(username, password string) (UserRecord, bool, error) {
	row, err := s.client.User.Query().Where(user.Username(strings.TrimSpace(username))).Only(context.Background())
	if ent.IsNotFound(err) {
		return UserRecord{}, false, nil
	}
	if err != nil {
		return UserRecord{}, false, err
	}
	if !strings.HasPrefix(row.PasswordHash, "$argon2id$") {
		return UserRecord{}, false, nil
	}
	ok, err := VerifyPassword(row.PasswordHash, password)
	if err != nil {
		return UserRecord{}, false, err
	}
	return userRecordFromEnt(row), ok, nil
}

func (s *Store) UpsertExternalUser(tenant, username, displayName, avatarURL string) (UserRecord, error) {
	return s.UpsertOIDCUser(OIDCProfile{
		Issuer:      "legacy",
		Subject:     tenant,
		Email:       username,
		Username:    username,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	})
}

func (s *Store) UpsertOIDCUser(profile OIDCProfile) (UserRecord, error) {
	profile = normalizeOIDCProfile(profile)
	if profile.Issuer == "" {
		return UserRecord{}, fmt.Errorf("oidc issuer is required")
	}
	if profile.Subject == "" {
		return UserRecord{}, fmt.Errorf("oidc subject is required")
	}
	ctx := context.Background()
	row, err := s.client.User.Query().Where(user.OidcIssuer(profile.Issuer), user.OidcSubject(profile.Subject)).Only(ctx)
	if ent.IsNotFound(err) {
		row, err = s.linkOIDCByVerifiedEmail(ctx, profile)
		if err == nil && row != nil {
			return userRecordFromEnt(row), nil
		}
		if err != nil {
			return UserRecord{}, err
		}
		tenant, username, err := s.uniqueOIDCUserIdentity(ctx, profile)
		if err != nil {
			return UserRecord{}, err
		}
		create := s.client.User.Create().
			SetTenant(tenant).
			SetUsername(username).
			SetPasswordHash("external:oidc").
			SetDisplayName(defaultString(profile.DisplayName, username)).
			SetAvatarURL(profile.AvatarURL).
			SetOidcIssuer(profile.Issuer).
			SetOidcSubject(profile.Subject).
			SetOidcEmail(profile.Email).
			SetIsAdmin(false).
			SetCreatedAt(time.Now().UTC())
		if ok, err := s.emailIsUnique(ctx, profile.Email, ""); err != nil {
			return UserRecord{}, err
		} else if profile.EmailVerified && profile.Email != "" && ok {
			create.SetEmail(profile.Email)
		}
		err = create.Exec(ctx)
		if err != nil {
			return UserRecord{}, err
		}
		row, err = s.client.User.Query().Where(user.OidcIssuer(profile.Issuer), user.OidcSubject(profile.Subject)).Only(ctx)
	} else if err == nil {
		update := row.Update().SetOidcEmail(profile.Email)
		if ok, err := s.emailIsUnique(ctx, profile.Email, row.Tenant); err != nil {
			return UserRecord{}, err
		} else if profile.EmailVerified && profile.Email != "" && ok {
			update.SetEmail(profile.Email)
		}
		if profile.DisplayName != "" {
			update.SetDisplayName(profile.DisplayName)
		}
		if profile.AvatarURL != "" {
			update.SetAvatarURL(profile.AvatarURL)
		}
		err = update.Exec(ctx)
		if err == nil {
			row, err = s.client.User.Query().Where(user.OidcIssuer(profile.Issuer), user.OidcSubject(profile.Subject)).Only(ctx)
		}
	}
	if err != nil {
		return UserRecord{}, err
	}
	return userRecordFromEnt(row), nil
}

func (s *Store) LinkOIDCIdentity(tenant string, profile OIDCProfile) (Profile, error) {
	profile = normalizeOIDCProfile(profile)
	if profile.Issuer == "" {
		return Profile{}, fmt.Errorf("oidc issuer is required")
	}
	if profile.Subject == "" {
		return Profile{}, fmt.Errorf("oidc subject is required")
	}
	ctx := context.Background()
	tenant = TenantKey(tenant)
	row, err := s.client.User.Query().Where(user.Tenant(tenant)).Only(ctx)
	if err != nil {
		return Profile{}, err
	}
	existing, err := s.client.User.Query().Where(user.OidcIssuer(profile.Issuer), user.OidcSubject(profile.Subject)).Only(ctx)
	if err == nil && existing.Tenant != row.Tenant {
		return Profile{}, fmt.Errorf("oidc account is already linked to another user")
	}
	if err != nil && !ent.IsNotFound(err) {
		return Profile{}, err
	}
	update := row.Update().
		SetOidcIssuer(profile.Issuer).
		SetOidcSubject(profile.Subject).
		SetOidcEmail(profile.Email)
	if profile.EmailVerified && profile.Email != "" {
		if err := s.ensureUniqueEmail(ctx, profile.Email, row.Tenant); err != nil {
			return Profile{}, err
		}
		update.SetEmail(profile.Email)
	}
	if profile.DisplayName != "" && strings.TrimSpace(row.DisplayName) == "" {
		update.SetDisplayName(profile.DisplayName)
	}
	if profile.AvatarURL != "" && strings.TrimSpace(row.AvatarURL) == "" {
		update.SetAvatarURL(profile.AvatarURL)
	}
	if err := update.Exec(ctx); err != nil {
		return Profile{}, err
	}
	row, err = s.client.User.Query().Where(user.Tenant(tenant)).Only(ctx)
	if err != nil {
		return Profile{}, err
	}
	return profileFromUser(row), nil
}

func (s *Store) OIDCUser(issuer, subject string) (UserRecord, bool, error) {
	issuer = strings.TrimSpace(issuer)
	subject = strings.TrimSpace(subject)
	if issuer == "" || subject == "" {
		return UserRecord{}, false, nil
	}
	row, err := s.client.User.Query().Where(user.OidcIssuer(issuer), user.OidcSubject(subject)).Only(context.Background())
	if ent.IsNotFound(err) {
		return UserRecord{}, false, nil
	}
	if err != nil {
		return UserRecord{}, false, err
	}
	return userRecordFromEnt(row), true, nil
}

func (s *Store) Profile(tenant string) (Profile, error) {
	row, err := s.client.User.Query().Where(user.Tenant(TenantKey(tenant))).Only(context.Background())
	if err != nil {
		return Profile{}, err
	}
	return profileFromUser(row), nil
}

func (s *Store) UpdateProfile(tenant, displayName, email, avatarURL string) (Profile, error) {
	displayName = strings.TrimSpace(displayName)
	email = strings.TrimSpace(email)
	avatarURL = strings.TrimSpace(avatarURL)
	if displayName == "" {
		return Profile{}, fmt.Errorf("display name is required")
	}
	normalizedEmail, err := NormalizeOptionalEmail(email)
	if err != nil {
		return Profile{}, err
	}
	if err := ValidateAvatarURL(avatarURL); err != nil {
		return Profile{}, err
	}
	ctx := context.Background()
	row, err := s.client.User.Query().Where(user.Tenant(TenantKey(tenant))).Only(ctx)
	if err != nil {
		return Profile{}, err
	}
	if err := s.ensureUniqueEmail(ctx, normalizedEmail, row.Tenant); err != nil {
		return Profile{}, err
	}
	update := row.Update().
		SetDisplayName(displayName).
		SetAvatarURL(avatarURL)
	if normalizedEmail == "" {
		update.ClearEmail()
	} else {
		update.SetEmail(normalizedEmail)
	}
	if err := update.Exec(ctx); err != nil {
		return Profile{}, err
	}
	row, err = s.client.User.Query().Where(user.Tenant(TenantKey(tenant))).Only(ctx)
	if err != nil {
		return Profile{}, err
	}
	return profileFromUser(row), nil
}

func (s *Store) ListAdminUsers() ([]AdminUser, error) {
	rows, err := s.client.User.Query().Order(ent.Asc(user.FieldUsername)).All(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]AdminUser, 0, len(rows))
	for _, row := range rows {
		out = append(out, adminUserFromEnt(row))
	}
	return out, nil
}

func (s *Store) UpdateUser(tenant string, update UserUpdate) (AdminUser, error) {
	tenant = TenantKey(tenant)
	update.Username = strings.TrimSpace(update.Username)
	update.Email = strings.TrimSpace(update.Email)
	update.DisplayName = strings.TrimSpace(update.DisplayName)
	if tenant == "" {
		return AdminUser{}, fmt.Errorf("tenant is required")
	}
	if update.Username == "" {
		return AdminUser{}, fmt.Errorf("username is required")
	}
	if update.DisplayName == "" {
		update.DisplayName = update.Username
	}
	normalizedEmail, err := NormalizeOptionalEmail(update.Email)
	if err != nil {
		return AdminUser{}, err
	}
	newTenant := TenantKey(update.Username)
	if newTenant == "" {
		return AdminUser{}, fmt.Errorf("username cannot produce a valid tenant key")
	}
	ctx := context.Background()
	row, err := s.client.User.Query().Where(user.Tenant(tenant)).Only(ctx)
	if err != nil {
		return AdminUser{}, err
	}
	if row.Username != update.Username {
		exists, err := s.client.User.Query().Where(user.Username(update.Username)).Exist(ctx)
		if err != nil {
			return AdminUser{}, err
		}
		if exists {
			return AdminUser{}, fmt.Errorf("username %q already exists", update.Username)
		}
	}
	if !update.IsAdmin && row.IsAdmin {
		admins, err := s.adminCountExcept(ctx, tenant)
		if err != nil {
			return AdminUser{}, err
		}
		if admins == 0 {
			return AdminUser{}, fmt.Errorf("cannot remove the last administrator")
		}
	}
	if newTenant != tenant {
		if exists, err := s.client.User.Query().Where(user.Tenant(newTenant)).Exist(ctx); err != nil {
			return AdminUser{}, err
		} else if exists {
			return AdminUser{}, fmt.Errorf("tenant %q already exists", newTenant)
		}
		if exists, err := s.client.TenantConfig.Query().Where(tenantconfig.Tenant(newTenant)).Exist(ctx); err != nil {
			return AdminUser{}, err
		} else if exists {
			return AdminUser{}, fmt.Errorf("tenant config %q already exists", newTenant)
		}
	}
	if err := s.ensureUniqueEmail(ctx, normalizedEmail, tenant); err != nil {
		return AdminUser{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return AdminUser{}, err
	}
	userUpdate := tx.User.UpdateOneID(row.ID).
		SetTenant(newTenant).
		SetUsername(update.Username).
		SetDisplayName(update.DisplayName).
		SetIsAdmin(update.IsAdmin)
	if normalizedEmail == "" {
		userUpdate.ClearEmail()
	} else {
		userUpdate.SetEmail(normalizedEmail)
	}
	updated, err := userUpdate.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return AdminUser{}, err
	}
	if newTenant != tenant {
		if err := renameTenantConfig(ctx, tx, tenant, newTenant); err != nil {
			_ = tx.Rollback()
			return AdminUser{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return AdminUser{}, err
	}
	updated, err = s.client.User.Query().Where(user.Tenant(updated.Tenant)).Only(ctx)
	if err != nil {
		return AdminUser{}, err
	}
	return adminUserFromEnt(updated), nil
}

func (s *Store) ResetUserPassword(tenant, password string) error {
	tenant = TenantKey(tenant)
	if tenant == "" {
		return fmt.Errorf("tenant is required")
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	row, err := s.client.User.Query().Where(user.Tenant(tenant)).Only(context.Background())
	if err != nil {
		return err
	}
	return row.Update().SetPasswordHash(hash).Exec(context.Background())
}

func (s *Store) DeleteUser(tenant string) error {
	tenant = TenantKey(tenant)
	if tenant == "" {
		return fmt.Errorf("tenant is required")
	}
	ctx := context.Background()
	row, err := s.client.User.Query().Where(user.Tenant(tenant)).Only(ctx)
	if err != nil {
		return err
	}
	if row.IsAdmin {
		admins, err := s.adminCountExcept(ctx, tenant)
		if err != nil {
			return err
		}
		if admins == 0 {
			return fmt.Errorf("cannot delete the last administrator")
		}
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.User.DeleteOneID(row.ID).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.TenantConfig.Delete().Where(tenantconfig.Tenant(tenant)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.RunRecord.Delete().Where(runrecord.Tenant(tenant)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Store) adminCountExcept(ctx context.Context, tenant string) (int, error) {
	query := s.client.User.Query().Where(user.IsAdmin(true))
	if tenant = TenantKey(tenant); tenant != "" {
		query = query.Where(user.TenantNEQ(tenant))
	}
	return query.Count(ctx)
}

func (s *Store) ensureUniqueEmail(ctx context.Context, email, exceptTenant string) error {
	ok, err := s.emailIsUnique(ctx, email, exceptTenant)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("email %q already exists", NormalizeEmail(email))
	}
	return nil
}

func (s *Store) emailIsUnique(ctx context.Context, email, exceptTenant string) (bool, error) {
	email = NormalizeEmail(email)
	if email == "" {
		return true, nil
	}
	query := s.client.User.Query().Where(user.EmailEqualFold(email))
	if exceptTenant = TenantKey(exceptTenant); exceptTenant != "" {
		query = query.Where(user.TenantNEQ(exceptTenant))
	}
	exists, err := query.Exist(ctx)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

func (s *Store) linkOIDCByVerifiedEmail(ctx context.Context, profile OIDCProfile) (*ent.User, error) {
	if !profile.EmailVerified || profile.Email == "" {
		return nil, nil
	}
	rows, err := s.client.User.Query().Where(user.EmailEqualFold(profile.Email)).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		return nil, nil
	}
	candidate := rows[0]
	if strings.TrimSpace(candidate.OidcIssuer) != "" || strings.TrimSpace(candidate.OidcSubject) != "" {
		return nil, nil
	}
	update := candidate.Update().
		SetOidcIssuer(profile.Issuer).
		SetOidcSubject(profile.Subject).
		SetOidcEmail(profile.Email).
		SetEmail(profile.Email)
	if profile.DisplayName != "" && strings.TrimSpace(candidate.DisplayName) == "" {
		update.SetDisplayName(profile.DisplayName)
	}
	if profile.AvatarURL != "" && strings.TrimSpace(candidate.AvatarURL) == "" {
		update.SetAvatarURL(profile.AvatarURL)
	}
	if err := update.Exec(ctx); err != nil {
		return nil, err
	}
	return s.client.User.Query().Where(user.Tenant(candidate.Tenant)).Only(ctx)
}

func renameTenantConfig(ctx context.Context, tx *ent.Tx, oldTenant, newTenant string) error {
	row, err := tx.TenantConfig.Query().Where(tenantconfig.Tenant(oldTenant)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := row.Update().SetTenant(newTenant).Exec(ctx); err != nil {
		return err
	}
	_, err = tx.RunRecord.Update().Where(runrecord.Tenant(oldTenant)).SetTenant(newTenant).Save(ctx)
	return err
}

func (s *Store) uniqueOIDCUserIdentity(ctx context.Context, profile OIDCProfile) (tenant string, username string, err error) {
	baseTenant := TenantKey(profile.Subject)
	if baseTenant == "" {
		baseTenant = TenantKey(profile.Email)
	}
	if baseTenant == "" {
		baseTenant = "oidc-user"
	}
	baseUsername := strings.TrimSpace(profile.Username)
	if baseUsername == "" {
		baseUsername = baseTenant
	}
	for i := 0; i < 100; i++ {
		candidateTenant := baseTenant
		candidateUsername := baseUsername
		if i > 0 {
			candidateTenant = fmt.Sprintf("%s-%d", trimTenantForSuffix(baseTenant, i), i)
			candidateUsername = fmt.Sprintf("%s-%d", strings.TrimSpace(baseUsername), i)
		}
		tenantExists, err := s.client.User.Query().Where(user.Tenant(candidateTenant)).Exist(ctx)
		if err != nil {
			return "", "", err
		}
		if tenantExists {
			continue
		}
		usernameExists, err := s.client.User.Query().Where(user.Username(candidateUsername)).Exist(ctx)
		if err != nil {
			return "", "", err
		}
		if usernameExists {
			continue
		}
		return candidateTenant, candidateUsername, nil
	}
	return "", "", fmt.Errorf("failed to allocate oidc user identity")
}

func trimTenantForSuffix(tenant string, suffix int) string {
	s := strconv.Itoa(suffix)
	limit := 64 - len(s) - 1
	if limit < 1 {
		return tenant
	}
	if len(tenant) > limit {
		return strings.Trim(tenant[:limit], "-_")
	}
	return tenant
}

func ValidateAvatarURL(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if len(value) > 512*1024 {
		return fmt.Errorf("avatar image is too large; keep it under 512 KiB after base64 encoding")
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "data:image/png;base64,") ||
		strings.HasPrefix(lower, "data:image/jpeg;base64,") ||
		strings.HasPrefix(lower, "data:image/webp;base64,") ||
		strings.HasPrefix(lower, "data:image/gif;base64,") {
		return nil
	}
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(value, "/") {
		return nil
	}
	return fmt.Errorf("avatar must be an http(s) URL, a relative path, or a base64 image data URL")
}

func NormalizeEmail(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	addr, err := mail.ParseAddress(value)
	if err != nil || addr.Address != value {
		return ""
	}
	return addr.Address
}

func NormalizeOptionalEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	normalized := NormalizeEmail(value)
	if normalized == "" {
		return "", fmt.Errorf("email is invalid")
	}
	return normalized, nil
}

func profileFromUser(row *ent.User) Profile {
	displayName := row.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName = row.Username
	}
	return Profile{
		Tenant:      row.Tenant,
		Username:    row.Username,
		Email:       row.Email,
		DisplayName: displayName,
		AvatarURL:   row.AvatarURL,
		IsAdmin:     row.IsAdmin,
		OIDC: OIDCIdentity{
			Linked: strings.TrimSpace(row.OidcIssuer) != "" && strings.TrimSpace(row.OidcSubject) != "",
			Issuer: row.OidcIssuer,
			Email:  row.OidcEmail,
		},
	}
}

func userRecordFromEnt(row *ent.User) UserRecord {
	return UserRecord{
		Tenant:       row.Tenant,
		Username:     row.Username,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		DisplayName:  row.DisplayName,
		AvatarURL:    row.AvatarURL,
		OIDCIssuer:   row.OidcIssuer,
		OIDCSubject:  row.OidcSubject,
		OIDCEmail:    row.OidcEmail,
		IsAdmin:      row.IsAdmin,
		CreatedAt:    row.CreatedAt,
	}
}

func adminUserFromEnt(row *ent.User) AdminUser {
	profile := profileFromUser(row)
	return AdminUser{
		Tenant:      profile.Tenant,
		Username:    profile.Username,
		Email:       profile.Email,
		DisplayName: profile.DisplayName,
		AvatarURL:   profile.AvatarURL,
		IsAdmin:     profile.IsAdmin,
		OIDC:        profile.OIDC,
		CreatedAt:   row.CreatedAt,
	}
}

func normalizeOIDCProfile(profile OIDCProfile) OIDCProfile {
	profile.Issuer = strings.TrimSpace(profile.Issuer)
	profile.Subject = strings.TrimSpace(profile.Subject)
	profile.Email = NormalizeEmail(profile.Email)
	profile.Username = strings.TrimSpace(firstNonEmpty(profile.Username, profile.PreferredUsername, profile.Email, profile.Subject))
	profile.DisplayName = strings.TrimSpace(firstNonEmpty(profile.DisplayName, profile.Nickname, profile.PreferredUsername, profile.Username, profile.Email))
	profile.AvatarURL = strings.TrimSpace(profile.AvatarURL)
	if err := ValidateAvatarURL(profile.AvatarURL); err != nil {
		profile.AvatarURL = ""
	}
	return profile
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (s *Store) Load() (Config, error) {
	if s.err != nil {
		return Config{}, s.err
	}
	row, err := s.client.SystemConfig.Query().Where(systemconfig.Key("default")).Only(context.Background())
	if ent.IsNotFound(err) {
		cfg := Default()
		if err := s.Save(cfg); err != nil {
			return Config{}, err
		}
		cfg = ApplyEnv(cfg)
		return cfg, Validate(cfg)
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal([]byte(row.Payload), &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	cfg = Normalize(cfg)
	cfg = ApplyEnv(cfg)
	return cfg, Validate(cfg)
}

func (s *Store) LoadUser(tenant string) (Config, error) {
	system, err := s.Load()
	if err != nil {
		return Config{}, err
	}
	tenant = TenantKey(tenant)
	if tenant == "" {
		return Config{}, fmt.Errorf("tenant is required")
	}
	row, err := s.client.TenantConfig.Query().Where(tenantconfig.Tenant(tenant)).Only(context.Background())
	if ent.IsNotFound(err) {
		user := Default().UserConfig()
		if err := s.SaveUser(tenant, WithUserConfig(system, user)); err != nil {
			return Config{}, err
		}
		cfg := ApplyEnv(WithUserConfig(system, user))
		return cfg, Validate(cfg)
	}
	if err != nil {
		return Config{}, err
	}
	var user UserConfig
	if err := json.Unmarshal([]byte(row.Payload), &user); err != nil {
		return Config{}, fmt.Errorf("decode user config: %w", err)
	}
	cfg := Normalize(WithUserConfig(system, user))
	cfg = ApplyEnv(cfg)
	return cfg, Validate(cfg)
}

func (s *Store) SaveUser(tenant string, cfg Config) error {
	system, err := s.Load()
	if err != nil {
		return err
	}
	tenant = TenantKey(tenant)
	if tenant == "" {
		return fmt.Errorf("tenant is required")
	}
	existing, hasExisting, err := s.loadStoredUserConfig(tenant)
	if err != nil {
		return err
	}
	user := Normalize(WithUserConfig(system, cfg.UserConfig())).UserConfig()
	if hasExisting {
		user = MergeUserSecrets(user, existing)
	}
	merged := WithUserConfig(system, user)
	if err := Validate(merged); err != nil {
		return err
	}
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return err
	}
	ctx := context.Background()
	row, err := s.client.TenantConfig.Query().Where(tenantconfig.Tenant(tenant)).Only(ctx)
	if ent.IsNotFound(err) {
		return s.client.TenantConfig.Create().
			SetTenant(tenant).
			SetPayload(string(data)).
			SetUpdatedAt(time.Now().UTC()).
			Exec(ctx)
	}
	if err != nil {
		return err
	}
	return row.Update().
		SetPayload(string(data)).
		SetUpdatedAt(time.Now().UTC()).
		Exec(ctx)
}

func (s *Store) loadStoredUserConfig(tenant string) (UserConfig, bool, error) {
	row, err := s.client.TenantConfig.Query().Where(tenantconfig.Tenant(TenantKey(tenant))).Only(context.Background())
	if ent.IsNotFound(err) {
		return UserConfig{}, false, nil
	}
	if err != nil {
		return UserConfig{}, false, err
	}
	var user UserConfig
	if err := json.Unmarshal([]byte(row.Payload), &user); err != nil {
		return UserConfig{}, false, fmt.Errorf("decode existing user config: %w", err)
	}
	return user, true, nil
}

func (s *Store) ListUsers() ([]string, error) {
	rows, err := s.client.TenantConfig.Query().Order(ent.Asc(tenantconfig.FieldTenant)).All(context.Background())
	if err != nil {
		return nil, err
	}
	users := make([]string, 0, len(rows))
	for _, row := range rows {
		users = append(users, row.Tenant)
	}
	return users, nil
}

func (s *Store) Save(cfg Config) error {
	cfg = Normalize(cfg)
	if err := Validate(cfg); err != nil {
		return err
	}
	system := Redacted(cfg)
	system.Runtime = RuntimeConfig{}
	system.TaskRuntime = nil
	system.Sources = nil
	system.Schedules = nil
	system.Targets = nil
	system.ExportOptions = nil
	system.BackupStrategies = nil
	system.Tasks = nil
	system.HistoryLimit = 0
	system.HistoryRetentionDays = 0
	data, err := json.MarshalIndent(system, "", "  ")
	if err != nil {
		return err
	}
	ctx := context.Background()
	row, err := s.client.SystemConfig.Query().Where(systemconfig.Key("default")).Only(ctx)
	if ent.IsNotFound(err) {
		return s.client.SystemConfig.Create().
			SetKey("default").
			SetPayload(string(data)).
			SetUpdatedAt(time.Now().UTC()).
			Exec(ctx)
	}
	if err != nil {
		return err
	}
	return row.Update().
		SetPayload(string(data)).
		SetUpdatedAt(time.Now().UTC()).
		Exec(ctx)
}

func Default() Config {
	return Config{
		Listen: ListenConfig{
			Host: "0.0.0.0",
			Port: DefaultPort,
		},
		Auth: AuthConfig{
			Username:         "admin",
			PasswordEnv:      "NMEM_SNAP_PASSWORD",
			SessionSecretEnv: "NMEM_SNAP_SESSION_SECRET",
			OIDC: OIDCConfig{
				Enabled:         false,
				ClientSecretEnv: "NMEM_SNAP_OIDC_CLIENT_SECRET",
				Scopes:          []string{"openid", "profile", "email"},
			},
		},
		Sources: []SourceConfig{
			{
				Key:     DefaultSourceKey,
				Name:    "Local Nowledge Mem",
				Enabled: true,
				Type:    "nowledgemem_api",
				NowledgeMem: NowledgeConfig{
					APIURL:    "http://127.0.0.1:14242",
					APIKeyEnv: "NMEM_API_KEY",
				},
			},
		},
		Schedules: []ScheduleConfig{
			{Key: DefaultDailyScheduleKey, Name: "Daily", Enabled: true, Type: "daily", Time: "03:00"},
			{Key: DefaultWeeklyScheduleKey, Name: "Weekly", Enabled: false, Type: "weekly", Time: "03:00", Weekday: "sunday"},
		},
		Targets: []TargetConfig{},
		ExportOptions: []ExportOptionConfig{
			{
				Key:    DefaultExportOptionKey,
				Name:   "Portable export",
				Export: defaultExportConfig(),
			},
		},
		BackupStrategies: []BackupStrategyConfig{
			{
				Key:  DefaultBackupStrategyKey,
				Name: "No remote cleanup",
				Retention: RetentionConfig{
					Mode: "none",
				},
			},
		},
		Tasks: []TaskConfig{
			{
				Key:               DefaultTaskKey,
				Name:              "Default backup",
				Enabled:           false,
				SourceKey:         DefaultSourceKey,
				ScheduleKey:       DefaultDailyScheduleKey,
				TargetKeys:        []string{},
				ExportOptionKey:   DefaultExportOptionKey,
				BackupStrategyKey: DefaultBackupStrategyKey,
				ObjectPrefix:      "nowledge-mem/{task}/{timestamp}",
				Encryption: EncryptionConfig{
					Enabled:     false,
					PasswordEnv: "NMEM_SNAP_ENCRYPTION_PASSWORD",
				},
			},
		},
		HistoryLimit:         100,
		HistoryRetentionDays: 180,
	}
}

func Normalize(cfg Config) Config {
	if strings.TrimSpace(cfg.Listen.Host) == "" {
		cfg.Listen.Host = "0.0.0.0"
	}
	if cfg.Listen.Port == 0 {
		cfg.Listen.Port = DefaultPort
	}
	cfg.Listen.BasePath = NormalizeBasePath(cfg.Listen.BasePath)
	if strings.TrimSpace(cfg.Auth.Username) == "" {
		cfg.Auth.Username = "admin"
	}
	if strings.TrimSpace(cfg.Auth.PasswordEnv) == "" {
		cfg.Auth.PasswordEnv = "NMEM_SNAP_PASSWORD"
	}
	if strings.TrimSpace(cfg.Auth.SessionSecretEnv) == "" {
		cfg.Auth.SessionSecretEnv = "NMEM_SNAP_SESSION_SECRET"
	}
	cfg.Auth.OIDC.IssuerURL = strings.TrimSpace(cfg.Auth.OIDC.IssuerURL)
	cfg.Auth.OIDC.ClientID = strings.TrimSpace(cfg.Auth.OIDC.ClientID)
	cfg.Auth.OIDC.ClientSecretEnv = defaultString(strings.TrimSpace(cfg.Auth.OIDC.ClientSecretEnv), "NMEM_SNAP_OIDC_CLIENT_SECRET")
	cfg.Auth.OIDC.RedirectURL = strings.TrimSpace(cfg.Auth.OIDC.RedirectURL)
	if len(cfg.Auth.OIDC.Scopes) == 0 {
		cfg.Auth.OIDC.Scopes = []string{"openid", "profile", "email"}
	}
	if cfg.HistoryLimit <= 0 {
		cfg.HistoryLimit = 100
	}
	if cfg.HistoryRetentionDays <= 0 {
		cfg.HistoryRetentionDays = 180
	}
	for i := range cfg.Sources {
		cfg.Sources[i].Key = strings.TrimSpace(cfg.Sources[i].Key)
		cfg.Sources[i].Name = defaultString(strings.TrimSpace(cfg.Sources[i].Name), cfg.Sources[i].Key)
		cfg.Sources[i].Type = strings.ToLower(strings.TrimSpace(cfg.Sources[i].Type))
		if cfg.Sources[i].Type == "" {
			cfg.Sources[i].Type = "nowledgemem_api"
		}
		cfg.Sources[i].NowledgeMem.APIURL = defaultString(strings.TrimSpace(cfg.Sources[i].NowledgeMem.APIURL), "http://127.0.0.1:14242")
		cfg.Sources[i].NowledgeMem.APIKeyEnv = defaultString(strings.TrimSpace(cfg.Sources[i].NowledgeMem.APIKeyEnv), sourceEnv(cfg.Sources[i].Key, "API_KEY"))
		if cfg.Sources[i].NowledgeMem.Timeout == 0 {
			cfg.Sources[i].NowledgeMem.Timeout = 5 * time.Minute
		}
		cfg.Sources[i].Directory.Path = strings.TrimSpace(cfg.Sources[i].Directory.Path)
	}
	for i := range cfg.Schedules {
		cfg.Schedules[i].Key = strings.TrimSpace(cfg.Schedules[i].Key)
		cfg.Schedules[i].Name = defaultString(strings.TrimSpace(cfg.Schedules[i].Name), cfg.Schedules[i].Key)
		cfg.Schedules[i].Type = strings.ToLower(strings.TrimSpace(cfg.Schedules[i].Type))
		cfg.Schedules[i].Time = defaultString(strings.TrimSpace(cfg.Schedules[i].Time), "03:00")
		cfg.Schedules[i].Weekday = strings.ToLower(strings.TrimSpace(cfg.Schedules[i].Weekday))
		cfg.Schedules[i].RunAt = strings.TrimSpace(cfg.Schedules[i].RunAt)
	}
	for i := range cfg.Targets {
		cfg.Targets[i].Key = strings.TrimSpace(cfg.Targets[i].Key)
		cfg.Targets[i].Name = defaultString(strings.TrimSpace(cfg.Targets[i].Name), cfg.Targets[i].Key)
		cfg.Targets[i].Type = strings.ToLower(strings.TrimSpace(cfg.Targets[i].Type))
		cfg.Targets[i].S3.Region = defaultString(strings.TrimSpace(cfg.Targets[i].S3.Region), "auto")
		cfg.Targets[i].S3.SecretAccessKeyEnv = defaultString(strings.TrimSpace(cfg.Targets[i].S3.SecretAccessKeyEnv), targetEnv(cfg.Targets[i].Key, "S3_SECRET_ACCESS_KEY"))
		cfg.Targets[i].WebDAV.PasswordEnv = defaultString(strings.TrimSpace(cfg.Targets[i].WebDAV.PasswordEnv), targetEnv(cfg.Targets[i].Key, "WEBDAV_PASSWORD"))
		cfg.Targets[i].GCS.BucketName = strings.TrimSpace(cfg.Targets[i].GCS.BucketName)
		cfg.Targets[i].GCS.RootPrefix = strings.TrimSpace(cfg.Targets[i].GCS.RootPrefix)
		cfg.Targets[i].GCS.CredentialsJSONEnv = defaultString(strings.TrimSpace(cfg.Targets[i].GCS.CredentialsJSONEnv), targetEnv(cfg.Targets[i].Key, "GCS_CREDENTIALS_JSON"))
		cfg.Targets[i].SFTP.Host = strings.TrimSpace(cfg.Targets[i].SFTP.Host)
		if cfg.Targets[i].SFTP.Port == 0 {
			cfg.Targets[i].SFTP.Port = 22
		}
		cfg.Targets[i].SFTP.RootPrefix = strings.TrimSpace(cfg.Targets[i].SFTP.RootPrefix)
		cfg.Targets[i].SFTP.Username = strings.TrimSpace(cfg.Targets[i].SFTP.Username)
		cfg.Targets[i].SFTP.PasswordEnv = defaultString(strings.TrimSpace(cfg.Targets[i].SFTP.PasswordEnv), targetEnv(cfg.Targets[i].Key, "SFTP_PASSWORD"))
		cfg.Targets[i].SFTP.PrivateKeyEnv = defaultString(strings.TrimSpace(cfg.Targets[i].SFTP.PrivateKeyEnv), targetEnv(cfg.Targets[i].Key, "SFTP_PRIVATE_KEY"))
		cfg.Targets[i].SFTP.PrivateKeyPassphraseEnv = defaultString(strings.TrimSpace(cfg.Targets[i].SFTP.PrivateKeyPassphraseEnv), targetEnv(cfg.Targets[i].Key, "SFTP_PRIVATE_KEY_PASSPHRASE"))
		cfg.Targets[i].SFTP.HostKeySHA256 = strings.TrimSpace(cfg.Targets[i].SFTP.HostKeySHA256)
	}
	if len(cfg.ExportOptions) == 0 {
		cfg.ExportOptions = []ExportOptionConfig{
			{Key: DefaultExportOptionKey, Name: "Portable export", Export: defaultExportConfig()},
		}
	}
	for i := range cfg.ExportOptions {
		cfg.ExportOptions[i].Key = strings.TrimSpace(cfg.ExportOptions[i].Key)
		cfg.ExportOptions[i].Name = defaultString(strings.TrimSpace(cfg.ExportOptions[i].Name), cfg.ExportOptions[i].Key)
		cfg.ExportOptions[i].Export = mergeExportConfig(defaultExportConfig(), cfg.ExportOptions[i].Export)
	}
	if len(cfg.BackupStrategies) == 0 {
		cfg.BackupStrategies = []BackupStrategyConfig{
			{Key: DefaultBackupStrategyKey, Name: "No remote cleanup", Retention: RetentionConfig{Mode: "none"}},
		}
	}
	for i := range cfg.BackupStrategies {
		cfg.BackupStrategies[i].Key = strings.TrimSpace(cfg.BackupStrategies[i].Key)
		cfg.BackupStrategies[i].Name = defaultString(strings.TrimSpace(cfg.BackupStrategies[i].Name), cfg.BackupStrategies[i].Key)
		cfg.BackupStrategies[i].Retention.Mode = strings.ToLower(strings.TrimSpace(cfg.BackupStrategies[i].Retention.Mode))
		if cfg.BackupStrategies[i].Retention.Mode == "" {
			cfg.BackupStrategies[i].Retention.Mode = "none"
		}
	}
	defaultExportKey := ""
	if len(cfg.ExportOptions) > 0 {
		defaultExportKey = cfg.ExportOptions[0].Key
	}
	defaultStrategyKey := ""
	if len(cfg.BackupStrategies) > 0 {
		defaultStrategyKey = cfg.BackupStrategies[0].Key
	}
	for i := range cfg.Tasks {
		cfg.Tasks[i].Key = strings.TrimSpace(cfg.Tasks[i].Key)
		cfg.Tasks[i].Name = defaultString(strings.TrimSpace(cfg.Tasks[i].Name), cfg.Tasks[i].Key)
		cfg.Tasks[i].SourceKey = strings.TrimSpace(cfg.Tasks[i].SourceKey)
		cfg.Tasks[i].ScheduleKey = strings.TrimSpace(cfg.Tasks[i].ScheduleKey)
		cfg.Tasks[i].ExportOptionKey = defaultString(strings.TrimSpace(cfg.Tasks[i].ExportOptionKey), defaultExportKey)
		cfg.Tasks[i].BackupStrategyKey = defaultString(strings.TrimSpace(cfg.Tasks[i].BackupStrategyKey), defaultStrategyKey)
		cfg.Tasks[i].ObjectPrefix = defaultString(strings.TrimSpace(cfg.Tasks[i].ObjectPrefix), "nowledge-mem/{task}/{timestamp}")
		cfg.Tasks[i].Encryption.PasswordEnv = defaultString(strings.TrimSpace(cfg.Tasks[i].Encryption.PasswordEnv), "NMEM_SNAP_ENCRYPTION_PASSWORD")
	}
	return cfg
}

func (c Config) UserConfig() UserConfig {
	return UserConfig{
		Sources:              cloneSlice(c.Sources),
		Schedules:            cloneSlice(c.Schedules),
		Targets:              cloneSlice(c.Targets),
		ExportOptions:        cloneSlice(c.ExportOptions),
		BackupStrategies:     cloneSlice(c.BackupStrategies),
		Tasks:                cloneSlice(c.Tasks),
		HistoryLimit:         c.HistoryLimit,
		HistoryRetentionDays: c.HistoryRetentionDays,
	}
}

func WithUserConfig(system Config, user UserConfig) Config {
	system.Sources = cloneSlice(user.Sources)
	system.Schedules = cloneSlice(user.Schedules)
	system.Targets = cloneSlice(user.Targets)
	system.ExportOptions = cloneSlice(user.ExportOptions)
	system.BackupStrategies = cloneSlice(user.BackupStrategies)
	system.Tasks = cloneSlice(user.Tasks)
	system.HistoryLimit = user.HistoryLimit
	system.HistoryRetentionDays = user.HistoryRetentionDays
	return system
}

func Runtime() RuntimeConfig {
	now := time.Now().In(time.Local)
	zone, offset := now.Zone()
	timezone := time.Local.String()
	if strings.TrimSpace(timezone) == "" {
		timezone = zone
	}
	if timezone == "" {
		timezone = "Local"
	}
	return RuntimeConfig{
		Timezone:      timezone,
		TimezoneLabel: fmt.Sprintf("%s (%s)", timezone, utcOffset(offset)),
	}
}

func utcOffset(seconds int) string {
	sign := "+"
	if seconds < 0 {
		sign = "-"
		seconds = -seconds
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("UTC%s%02d:%02d", sign, hours, minutes)
}

func MergeUserSecrets(incoming UserConfig, existing UserConfig) UserConfig {
	sourceByKey := make(map[string]SourceConfig, len(existing.Sources))
	for _, source := range existing.Sources {
		sourceByKey[source.Key] = source
	}
	for i := range incoming.Sources {
		existingSource, ok := sourceByKey[incoming.Sources[i].Key]
		if !ok {
			continue
		}
		incoming.Sources[i] = MergeSourceSecrets(incoming.Sources[i], existingSource)
	}

	targetByKey := make(map[string]TargetConfig, len(existing.Targets))
	for _, target := range existing.Targets {
		targetByKey[target.Key] = target
	}
	for i := range incoming.Targets {
		existingTarget, ok := targetByKey[incoming.Targets[i].Key]
		if !ok {
			continue
		}
		incoming.Targets[i] = MergeTargetSecrets(incoming.Targets[i], existingTarget)
	}

	taskByKey := make(map[string]TaskConfig, len(existing.Tasks))
	for _, task := range existing.Tasks {
		taskByKey[task.Key] = task
	}
	for i := range incoming.Tasks {
		existingTask, ok := taskByKey[incoming.Tasks[i].Key]
		if ok && incoming.Tasks[i].Encryption.Password == "" {
			incoming.Tasks[i].Encryption.Password = existingTask.Encryption.Password
		}
	}
	return incoming
}

func MergeSourceSecrets(incoming SourceConfig, existing SourceConfig) SourceConfig {
	if incoming.Type == "nowledgemem_api" && incoming.NowledgeMem.APIKey == "" {
		incoming.NowledgeMem.APIKey = existing.NowledgeMem.APIKey
	}
	return incoming
}

func MergeTargetSecrets(incoming TargetConfig, existing TargetConfig) TargetConfig {
	if incoming.Type == "s3" && incoming.S3.SecretAccessKey == "" {
		incoming.S3.SecretAccessKey = existing.S3.SecretAccessKey
	}
	if incoming.Type == "webdav" && incoming.WebDAV.Password == "" {
		incoming.WebDAV.Password = existing.WebDAV.Password
	}
	if incoming.Type == "gcs" && incoming.GCS.CredentialsJSON == "" {
		incoming.GCS.CredentialsJSON = existing.GCS.CredentialsJSON
	}
	if incoming.Type == "sftp" {
		if incoming.SFTP.Password == "" {
			incoming.SFTP.Password = existing.SFTP.Password
		}
		if incoming.SFTP.PrivateKey == "" {
			incoming.SFTP.PrivateKey = existing.SFTP.PrivateKey
		}
		if incoming.SFTP.PrivateKeyPassphrase == "" {
			incoming.SFTP.PrivateKeyPassphrase = existing.SFTP.PrivateKeyPassphrase
		}
	}
	return incoming
}

func ApplyEnv(cfg Config) Config {
	cfg = Normalize(cfg)
	if v := strings.TrimSpace(os.Getenv("BIND_HOST")); v != "" {
		cfg.Listen.Host = v
	}
	if v := strings.TrimSpace(os.Getenv("PORT")); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Listen.Port = port
		}
	}
	if v := strings.TrimSpace(os.Getenv("NMEM_SNAP_BASE_PATH")); v != "" {
		cfg.Listen.BasePath = NormalizeBasePath(v)
	}
	if v := strings.TrimSpace(os.Getenv("NMEM_API_URL")); v != "" {
		for i := range cfg.Sources {
			if cfg.Sources[i].Type == "nowledgemem_api" && cfg.Sources[i].Key == DefaultSourceKey {
				cfg.Sources[i].NowledgeMem.APIURL = v
			}
		}
	}
	if v := strings.TrimSpace(os.Getenv("NMEM_API_KEY")); v != "" {
		for i := range cfg.Sources {
			if cfg.Sources[i].Type == "nowledgemem_api" && cfg.Sources[i].Key == DefaultSourceKey && cfg.Sources[i].NowledgeMem.APIKey == "" {
				cfg.Sources[i].NowledgeMem.APIKey = v
			}
		}
	}
	if v := strings.TrimSpace(os.Getenv("NMEM_SNAP_AUTH_USERNAME")); v != "" {
		cfg.Auth.Username = v
	}
	if v := strings.TrimSpace(os.Getenv("NMEM_SNAP_OIDC_ENABLED")); v != "" {
		cfg.Auth.OIDC.Enabled = parseBool(v)
	}
	if v := strings.TrimSpace(os.Getenv("NMEM_SNAP_OIDC_ISSUER_URL")); v != "" {
		cfg.Auth.OIDC.IssuerURL = v
	}
	if v := strings.TrimSpace(os.Getenv("NMEM_SNAP_OIDC_CLIENT_ID")); v != "" {
		cfg.Auth.OIDC.ClientID = v
	}
	if v := strings.TrimSpace(os.Getenv("NMEM_SNAP_OIDC_REDIRECT_URL")); v != "" {
		cfg.Auth.OIDC.RedirectURL = v
	}
	if v := strings.TrimSpace(os.Getenv("NMEM_SNAP_OIDC_SCOPES")); v != "" {
		cfg.Auth.OIDC.Scopes = splitCSV(v)
	}
	if v := strings.TrimSpace(os.Getenv("NMEM_SNAP_OIDC_ALLOWED_EMAILS")); v != "" {
		cfg.Auth.OIDC.AllowedEmails = splitCSV(v)
	}
	if v := strings.TrimSpace(os.Getenv("NMEM_SNAP_OIDC_ALLOWED_DOMAINS")); v != "" {
		cfg.Auth.OIDC.AllowedDomains = splitCSV(v)
	}
	if cfg.Auth.OIDC.ClientSecret == "" {
		cfg.Auth.OIDC.ClientSecret = os.Getenv(cfg.Auth.OIDC.ClientSecretEnv)
	}
	for i := range cfg.Sources {
		if cfg.Sources[i].Type == "nowledgemem_api" && cfg.Sources[i].NowledgeMem.APIKey == "" {
			cfg.Sources[i].NowledgeMem.APIKey = os.Getenv(cfg.Sources[i].NowledgeMem.APIKeyEnv)
		}
	}
	for i := range cfg.Targets {
		switch cfg.Targets[i].Type {
		case "s3":
			if cfg.Targets[i].S3.SecretAccessKey == "" {
				cfg.Targets[i].S3.SecretAccessKey = os.Getenv(cfg.Targets[i].S3.SecretAccessKeyEnv)
			}
		case "webdav":
			if cfg.Targets[i].WebDAV.Password == "" {
				cfg.Targets[i].WebDAV.Password = os.Getenv(cfg.Targets[i].WebDAV.PasswordEnv)
			}
		case "gcs":
			if cfg.Targets[i].GCS.CredentialsJSON == "" {
				cfg.Targets[i].GCS.CredentialsJSON = os.Getenv(cfg.Targets[i].GCS.CredentialsJSONEnv)
			}
		case "sftp":
			if cfg.Targets[i].SFTP.Password == "" {
				cfg.Targets[i].SFTP.Password = os.Getenv(cfg.Targets[i].SFTP.PasswordEnv)
			}
			if cfg.Targets[i].SFTP.PrivateKey == "" {
				cfg.Targets[i].SFTP.PrivateKey = os.Getenv(cfg.Targets[i].SFTP.PrivateKeyEnv)
			}
			if cfg.Targets[i].SFTP.PrivateKeyPassphrase == "" {
				cfg.Targets[i].SFTP.PrivateKeyPassphrase = os.Getenv(cfg.Targets[i].SFTP.PrivateKeyPassphraseEnv)
			}
		}
	}
	for i := range cfg.Tasks {
		if cfg.Tasks[i].Encryption.Enabled && cfg.Tasks[i].Encryption.Password == "" {
			cfg.Tasks[i].Encryption.Password = os.Getenv(cfg.Tasks[i].Encryption.PasswordEnv)
		}
	}
	cfg.Runtime = Runtime()
	return cfg
}

func Redacted(cfg Config) Config {
	for i := range cfg.Sources {
		cfg.Sources[i].NowledgeMem.APIKey = ""
	}
	cfg.Auth.OIDC.ClientSecret = ""
	for i := range cfg.Targets {
		cfg.Targets[i].S3.SecretAccessKey = ""
		cfg.Targets[i].WebDAV.Password = ""
		cfg.Targets[i].GCS.CredentialsJSON = ""
		cfg.Targets[i].SFTP.Password = ""
		cfg.Targets[i].SFTP.PrivateKey = ""
		cfg.Targets[i].SFTP.PrivateKeyPassphrase = ""
	}
	for i := range cfg.Tasks {
		cfg.Tasks[i].Encryption.Password = ""
	}
	return cfg
}

func Validate(cfg Config) error {
	if cfg.Listen.Port < 1 || cfg.Listen.Port > 65535 {
		return fmt.Errorf("listen port must be between 1 and 65535")
	}
	if err := ValidateBasePath(cfg.Listen.BasePath); err != nil {
		return err
	}
	if cfg.Auth.OIDC.Enabled {
		if strings.TrimSpace(cfg.Auth.OIDC.IssuerURL) == "" {
			return fmt.Errorf("auth oidc issuer_url is required when oidc is enabled")
		}
		if strings.TrimSpace(cfg.Auth.OIDC.ClientID) == "" {
			return fmt.Errorf("auth oidc client_id is required when oidc is enabled")
		}
	}
	sourceKeys := make(map[string]struct{})
	for _, source := range cfg.Sources {
		if err := validateKey("source", source.Key); err != nil {
			return err
		}
		if _, ok := sourceKeys[source.Key]; ok {
			return fmt.Errorf("duplicate source key %q", source.Key)
		}
		sourceKeys[source.Key] = struct{}{}
		switch source.Type {
		case "nowledgemem_api":
			if strings.TrimSpace(source.NowledgeMem.APIURL) == "" {
				return fmt.Errorf("source %q nowledge_mem api_url is required", source.Key)
			}
			if err := validateHTTPURL(source.NowledgeMem.APIURL); err != nil {
				return fmt.Errorf("source %q nowledge_mem api_url: %w", source.Key, err)
			}
		case "directory":
			if strings.TrimSpace(source.Directory.Path) == "" {
				return fmt.Errorf("source %q directory path is required", source.Key)
			}
			if err := ValidateDirectorySource(source.Directory); err != nil {
				return fmt.Errorf("source %q directory: %w", source.Key, err)
			}
		default:
			return fmt.Errorf("source %q type must be nowledgemem_api or directory", source.Key)
		}
	}
	scheduleKeys := make(map[string]struct{})
	for _, schedule := range cfg.Schedules {
		if err := validateKey("schedule", schedule.Key); err != nil {
			return err
		}
		if _, ok := scheduleKeys[schedule.Key]; ok {
			return fmt.Errorf("duplicate schedule key %q", schedule.Key)
		}
		scheduleKeys[schedule.Key] = struct{}{}
		if schedule.Type != "daily" && schedule.Type != "weekly" && schedule.Type != "once" {
			return fmt.Errorf("schedule %q type must be daily, weekly, or once", schedule.Key)
		}
		switch schedule.Type {
		case "once":
			if _, err := ParseScheduleRunAt(schedule.RunAt, time.Local); err != nil {
				return fmt.Errorf("schedule %q run_at: %w", schedule.Key, err)
			}
		default:
			if _, _, err := ParseClock(schedule.Time); err != nil {
				return fmt.Errorf("schedule %q time: %w", schedule.Key, err)
			}
		}
		if schedule.Type == "weekly" {
			if _, err := ParseWeekday(schedule.Weekday); err != nil {
				return fmt.Errorf("schedule %q weekday: %w", schedule.Key, err)
			}
		}
	}
	targetKeys := make(map[string]struct{})
	for _, target := range cfg.Targets {
		if err := ValidateTargetConfig(target); err != nil {
			return err
		}
		if _, ok := targetKeys[target.Key]; ok {
			return fmt.Errorf("duplicate target key %q", target.Key)
		}
		targetKeys[target.Key] = struct{}{}
	}
	exportOptionKeys := make(map[string]struct{})
	for _, option := range cfg.ExportOptions {
		if err := validateKey("export option", option.Key); err != nil {
			return err
		}
		if _, ok := exportOptionKeys[option.Key]; ok {
			return fmt.Errorf("duplicate export option key %q", option.Key)
		}
		exportOptionKeys[option.Key] = struct{}{}
	}
	backupStrategyKeys := make(map[string]struct{})
	for _, strategy := range cfg.BackupStrategies {
		if err := validateKey("backup strategy", strategy.Key); err != nil {
			return err
		}
		if _, ok := backupStrategyKeys[strategy.Key]; ok {
			return fmt.Errorf("duplicate backup strategy key %q", strategy.Key)
		}
		backupStrategyKeys[strategy.Key] = struct{}{}
		if err := validateRetentionConfig("backup strategy", strategy.Key, strategy.Retention); err != nil {
			return err
		}
	}
	taskKeys := make(map[string]struct{})
	for _, task := range cfg.Tasks {
		if err := validateKey("task", task.Key); err != nil {
			return err
		}
		if _, ok := taskKeys[task.Key]; ok {
			return fmt.Errorf("duplicate task key %q", task.Key)
		}
		taskKeys[task.Key] = struct{}{}
		if _, ok := scheduleKeys[task.ScheduleKey]; !ok {
			return fmt.Errorf("task %q references missing schedule %q", task.Key, task.ScheduleKey)
		}
		if _, ok := sourceKeys[task.SourceKey]; !ok {
			return fmt.Errorf("task %q references missing source %q", task.Key, task.SourceKey)
		}
		for _, targetKey := range task.TargetKeys {
			if _, ok := targetKeys[targetKey]; !ok {
				return fmt.Errorf("task %q references missing target %q", task.Key, targetKey)
			}
		}
		if _, ok := exportOptionKeys[task.ExportOptionKey]; !ok {
			return fmt.Errorf("task %q references missing export option %q", task.Key, task.ExportOptionKey)
		}
		if _, ok := backupStrategyKeys[task.BackupStrategyKey]; !ok {
			return fmt.Errorf("task %q references missing backup strategy %q", task.Key, task.BackupStrategyKey)
		}
	}
	return nil
}

func NormalizeBasePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "/" || value == "." || value == "./" {
		return ""
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = strings.TrimRight(value, "/")
	if value == "" || value == "/" {
		return ""
	}
	return value
}

func ValidateBasePath(basePath string) error {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" {
		return nil
	}
	if !strings.HasPrefix(basePath, "/") {
		return fmt.Errorf("listen base_path must start with /")
	}
	if strings.HasSuffix(basePath, "/") {
		return fmt.Errorf("listen base_path must not end with /")
	}
	if strings.ContainsAny(basePath, "?#") {
		return fmt.Errorf("listen base_path must be a path, not a URL with query or fragment")
	}
	for _, r := range basePath {
		if r <= 0x20 || r == 0x7f {
			return fmt.Errorf("listen base_path must not contain whitespace or control characters")
		}
	}
	for _, segment := range strings.Split(strings.Trim(basePath, "/"), "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("listen base_path contains an invalid path segment")
		}
	}
	return nil
}

func ValidateTargetConfig(target TargetConfig) error {
	if err := validateKey("target", target.Key); err != nil {
		return err
	}
	switch target.Type {
	case "s3":
		if strings.TrimSpace(target.S3.EndpointURL) == "" {
			return fmt.Errorf("target %q s3 endpoint_url is required", target.Key)
		}
		if err := validateHTTPURL(target.S3.EndpointURL); err != nil {
			return fmt.Errorf("target %q s3 endpoint_url: %w", target.Key, err)
		}
		if strings.TrimSpace(target.S3.BucketName) == "" {
			return fmt.Errorf("target %q s3 bucket_name is required", target.Key)
		}
		if strings.TrimSpace(target.S3.AccessKeyID) == "" {
			return fmt.Errorf("target %q s3 access_key_id is required", target.Key)
		}
	case "webdav":
		if strings.TrimSpace(target.WebDAV.URL) == "" {
			return fmt.Errorf("target %q webdav url is required", target.Key)
		}
		if err := validateHTTPURL(target.WebDAV.URL); err != nil {
			return fmt.Errorf("target %q webdav url: %w", target.Key, err)
		}
	case "gcs":
		if strings.TrimSpace(target.GCS.BucketName) == "" {
			return fmt.Errorf("target %q gcs bucket_name is required", target.Key)
		}
	case "sftp":
		if strings.TrimSpace(target.SFTP.Host) == "" {
			return fmt.Errorf("target %q sftp host is required", target.Key)
		}
		if strings.ContainsAny(target.SFTP.Host, " \t\r\n/") {
			return fmt.Errorf("target %q sftp host must be a hostname or IP address", target.Key)
		}
		if target.SFTP.Port < 1 || target.SFTP.Port > 65535 {
			return fmt.Errorf("target %q sftp port must be between 1 and 65535", target.Key)
		}
		if strings.TrimSpace(target.SFTP.Username) == "" {
			return fmt.Errorf("target %q sftp username is required", target.Key)
		}
		hasPassword := strings.TrimSpace(target.SFTP.Password) != "" || strings.TrimSpace(target.SFTP.PasswordEnv) != ""
		hasPrivateKey := strings.TrimSpace(target.SFTP.PrivateKey) != "" || strings.TrimSpace(target.SFTP.PrivateKeyEnv) != ""
		if !hasPassword && !hasPrivateKey {
			return fmt.Errorf("target %q sftp password or private_key is required", target.Key)
		}
		if !target.SFTP.InsecureIgnoreHostKey && strings.TrimSpace(target.SFTP.HostKeySHA256) == "" {
			return fmt.Errorf("target %q sftp host_key_sha256 is required unless insecure_ignore_host_key is enabled", target.Key)
		}
	default:
		return fmt.Errorf("target %q type must be s3, webdav, gcs, or sftp", target.Key)
	}
	return nil
}

func validateRetentionConfig(kind string, key string, retention RetentionConfig) error {
	switch retention.Mode {
	case "", "none":
	case "keep_last":
		if retention.KeepLast < 1 {
			return fmt.Errorf("%s %q retention keep_last must be at least 1", kind, key)
		}
	case "keep_days":
		if retention.KeepDays < 1 {
			return fmt.Errorf("%s %q retention keep_days must be at least 1", kind, key)
		}
	case "keep_after":
		if strings.TrimSpace(retention.KeepAfter) == "" {
			return fmt.Errorf("%s %q retention keep_after is required", kind, key)
		}
		if _, err := parseDate(retention.KeepAfter); err != nil {
			return fmt.Errorf("%s %q retention keep_after: %w", kind, key, err)
		}
	case "keep_before":
		if strings.TrimSpace(retention.KeepBefore) == "" {
			return fmt.Errorf("%s %q retention keep_before is required", kind, key)
		}
		if _, err := parseDate(retention.KeepBefore); err != nil {
			return fmt.Errorf("%s %q retention keep_before: %w", kind, key, err)
		}
	default:
		return fmt.Errorf("%s %q retention mode must be none, keep_last, keep_days, keep_after, or keep_before", kind, key)
	}
	return nil
}

func validateHTTPURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("must be an absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("must use http or https")
	}
	return nil
}

func parseDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if _, err := time.Parse(time.RFC3339, raw); err == nil {
		return time.Parse(time.RFC3339, raw)
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("must use YYYY-MM-DD or RFC3339")
}

func AllowedDirectoryRoots() []DirectoryRoot {
	raw := strings.TrimSpace(os.Getenv("NMEM_SNAP_ALLOWED_SOURCE_ROOTS"))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	roots := make([]DirectoryRoot, 0, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := fmt.Sprintf("root%d", i+1)
		name := part
		if strings.Contains(part, "=") {
			pieces := strings.SplitN(part, "=", 2)
			key = TenantKey(pieces[0])
			name = pieces[0]
			part = strings.TrimSpace(pieces[1])
		}
		roots = append(roots, DirectoryRoot{Key: key, Name: name, Path: part})
	}
	return roots
}

func ValidateDirectorySource(source DirectorySource) error {
	roots := AllowedDirectoryRoots()
	if len(roots) == 0 {
		return fmt.Errorf("directory sources require NMEM_SNAP_ALLOWED_SOURCE_ROOTS")
	}
	cleaned, err := filepath.Abs(source.Path)
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
		cleaned = resolved
	}
	for _, root := range roots {
		rootPath, err := filepath.Abs(root.Path)
		if err != nil {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(rootPath); err == nil {
			rootPath = resolved
		}
		rel, err := filepath.Rel(rootPath, cleaned)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			if source.RootKey == "" || source.RootKey == root.Key {
				return nil
			}
		}
	}
	return fmt.Errorf("path %q is outside allowed source roots", source.Path)
}

func (c Config) Source(key string) (SourceConfig, bool) {
	for _, source := range c.Sources {
		if source.Key == key {
			return source, true
		}
	}
	return SourceConfig{}, false
}

func (c Config) Task(key string) (TaskConfig, bool) {
	for _, task := range c.Tasks {
		if task.Key == key {
			return task, true
		}
	}
	return TaskConfig{}, false
}

func (c Config) Schedule(key string) (ScheduleConfig, bool) {
	for _, schedule := range c.Schedules {
		if schedule.Key == key {
			return schedule, true
		}
	}
	return ScheduleConfig{}, false
}

func (c Config) Target(key string) (TargetConfig, bool) {
	for _, target := range c.Targets {
		if target.Key == key {
			return target, true
		}
	}
	return TargetConfig{}, false
}

func (c Config) ExportOption(key string) (ExportOptionConfig, bool) {
	for _, option := range c.ExportOptions {
		if option.Key == key {
			return option, true
		}
	}
	return ExportOptionConfig{}, false
}

func (c Config) BackupStrategy(key string) (BackupStrategyConfig, bool) {
	for _, strategy := range c.BackupStrategies {
		if strategy.Key == key {
			return strategy, true
		}
	}
	return BackupStrategyConfig{}, false
}

func (c Config) ResolveTask(task TaskConfig) (TaskConfig, error) {
	strategy, ok := c.BackupStrategy(task.BackupStrategyKey)
	if !ok {
		return TaskConfig{}, fmt.Errorf("backup strategy %q was not found", task.BackupStrategyKey)
	}
	task.Retention = strategy.Retention
	return task, nil
}

func defaultExportConfig() ExportConfig {
	return ExportConfig{
		IncludeMemories:             boolPtr(true),
		IncludeThreads:              boolPtr(true),
		IncludeMessages:             boolPtr(true),
		IncludeEntities:             boolPtr(true),
		IncludeLabels:               boolPtr(true),
		IncludeSources:              boolPtr(true),
		IncludeCommunities:          boolPtr(true),
		IncludeSkills:               boolPtr(true),
		IncludeEdges:                boolPtr(true),
		IncludeWorkingMemory:        boolPtr(true),
		IncludeWorkingMemoryArchive: boolPtr(false),
		IncludeSourceFiles:          boolPtr(false),
	}
}

func DefaultExportConfig() ExportConfig {
	return defaultExportConfig()
}

func mergeExportConfig(base ExportConfig, override ExportConfig) ExportConfig {
	result := base
	if override.IncludeMemories != nil {
		result.IncludeMemories = override.IncludeMemories
	}
	if override.IncludeThreads != nil {
		result.IncludeThreads = override.IncludeThreads
	}
	if override.IncludeMessages != nil {
		result.IncludeMessages = override.IncludeMessages
	}
	if override.IncludeEntities != nil {
		result.IncludeEntities = override.IncludeEntities
	}
	if override.IncludeLabels != nil {
		result.IncludeLabels = override.IncludeLabels
	}
	if override.IncludeSources != nil {
		result.IncludeSources = override.IncludeSources
	}
	if override.IncludeCommunities != nil {
		result.IncludeCommunities = override.IncludeCommunities
	}
	if override.IncludeSkills != nil {
		result.IncludeSkills = override.IncludeSkills
	}
	if override.IncludeEdges != nil {
		result.IncludeEdges = override.IncludeEdges
	}
	if override.IncludeWorkingMemory != nil {
		result.IncludeWorkingMemory = override.IncludeWorkingMemory
	}
	if override.IncludeWorkingMemoryArchive != nil {
		result.IncludeWorkingMemoryArchive = override.IncludeWorkingMemoryArchive
	}
	if override.IncludeSourceFiles != nil {
		result.IncludeSourceFiles = override.IncludeSourceFiles
	}
	return result
}

func ParseClock(raw string) (hour int, minute int, err error) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("must use HH:MM")
	}
	hour, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hour")
	}
	minute, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minute")
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("must be within 00:00 and 23:59")
	}
	return hour, minute, nil
}

func ParseScheduleRunAt(raw string, loc *time.Location) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("must be set for once schedules")
	}
	if loc == nil {
		loc = time.Local
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	for _, layout := range []string{
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("must use YYYY-MM-DDTHH:MM in the configured timezone or RFC3339")
}

func ParseWeekday(raw string) (time.Weekday, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "sunday", "sun", "0":
		return time.Sunday, nil
	case "monday", "mon", "1":
		return time.Monday, nil
	case "tuesday", "tue", "2":
		return time.Tuesday, nil
	case "wednesday", "wed", "3":
		return time.Wednesday, nil
	case "thursday", "thu", "4":
		return time.Thursday, nil
	case "friday", "fri", "5":
		return time.Friday, nil
	case "saturday", "sat", "6":
		return time.Saturday, nil
	default:
		return time.Sunday, fmt.Errorf("must be sunday..saturday")
	}
}

func EnabledTargetKeys(cfg Config) []string {
	keys := make([]string, 0, len(cfg.Targets))
	for _, target := range cfg.Targets {
		if target.Enabled {
			keys = append(keys, target.Key)
		}
	}
	slices.Sort(keys)
	return keys
}

func TenantKey(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_':
			b.WriteRune(r)
		case r == '.', r == '@':
			b.WriteByte('-')
		default:
			b.WriteByte('-')
		}
	}
	key := strings.Trim(b.String(), "-_")
	if key == "" {
		return ""
	}
	if len(key) > 64 {
		return key[:64]
	}
	return key
}

func validateKey(kind, key string) error {
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("%s key %q must match %s", kind, key, keyPattern.String())
	}
	return nil
}

func targetEnv(key, suffix string) string {
	key = strings.ToUpper(strings.NewReplacer("-", "_").Replace(key))
	return "NMEM_SNAP_TARGET_" + key + "_" + suffix
}

func sourceEnv(key, suffix string) string {
	key = strings.ToUpper(strings.NewReplacer("-", "_").Replace(key))
	return "NMEM_SNAP_SOURCE_" + key + "_" + suffix
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func boolPtr(v bool) *bool {
	return &v
}

func parseBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func cloneSlice[T any](in []T) []T {
	if in == nil {
		return nil
	}
	out := make([]T, len(in))
	copy(out, in)
	return out
}
