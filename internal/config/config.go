package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/lib-x/nowledge-mem-snap/internal/persist"
	"github.com/lib-x/nowledge-mem-snap/internal/persist/ent"
	"github.com/lib-x/nowledge-mem-snap/internal/persist/ent/systemconfig"
	"github.com/lib-x/nowledge-mem-snap/internal/persist/ent/tenantconfig"
	"github.com/lib-x/nowledge-mem-snap/internal/persist/ent/user"
)

const (
	DefaultPort = 14335
)

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

type Config struct {
	Listen               ListenConfig     `json:"listen"`
	Auth                 AuthConfig       `json:"auth"`
	Export               ExportConfig     `json:"export"`
	Sources              []SourceConfig   `json:"sources"`
	Schedules            []ScheduleConfig `json:"schedules"`
	Targets              []TargetConfig   `json:"targets"`
	Tasks                []TaskConfig     `json:"tasks"`
	HistoryLimit         int              `json:"history_limit"`
	HistoryRetentionDays int              `json:"history_retention_days"`
}

type UserConfig struct {
	Export               ExportConfig     `json:"export"`
	Sources              []SourceConfig   `json:"sources"`
	Schedules            []ScheduleConfig `json:"schedules"`
	Targets              []TargetConfig   `json:"targets"`
	Tasks                []TaskConfig     `json:"tasks"`
	HistoryLimit         int              `json:"history_limit"`
	HistoryRetentionDays int              `json:"history_retention_days"`
}

type ListenConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type AuthConfig struct {
	Username         string     `json:"username"`
	PasswordEnv      string     `json:"password_env"`
	SessionSecretEnv string     `json:"session_secret_env"`
	OIDC             OIDCConfig `json:"oidc"`
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
	APIKeyEnv string        `json:"api_key_env"`
	APIKey    string        `json:"-"`
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
}

type S3Config struct {
	EndpointURL        string `json:"endpoint_url"`
	Region             string `json:"region"`
	PathStyle          bool   `json:"path_style"`
	BucketName         string `json:"bucket_name"`
	RootPrefix         string `json:"root_prefix"`
	AccessKeyID        string `json:"access_key_id"`
	SecretAccessKey    string `json:"-"`
	SecretAccessKeyEnv string `json:"secret_access_key_env"`
}

type WebDAVConfig struct {
	URL         string `json:"url"`
	RootPrefix  string `json:"root_prefix"`
	Username    string `json:"username"`
	Password    string `json:"-"`
	PasswordEnv string `json:"password_env"`
}

type TaskConfig struct {
	Key          string           `json:"key"`
	Name         string           `json:"name"`
	Enabled      bool             `json:"enabled"`
	SourceKey    string           `json:"source_key"`
	ScheduleKey  string           `json:"schedule_key"`
	TargetKeys   []string         `json:"target_keys"`
	ObjectPrefix string           `json:"object_prefix"`
	Encryption   EncryptionConfig `json:"encryption"`
	Retention    RetentionConfig  `json:"retention"`
	Export       ExportConfig     `json:"export"`
}

type EncryptionConfig struct {
	Enabled     bool   `json:"enabled"`
	PasswordEnv string `json:"password_env"`
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
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	AvatarURL    string    `json:"avatar_url"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
}

type Profile struct {
	Tenant      string `json:"tenant"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	IsAdmin     bool   `json:"is_admin"`
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
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	ctx := context.Background()
	existing, err := s.client.User.Query().Where(user.Username(username)).Only(ctx)
	if ent.IsNotFound(err) {
		err = s.client.User.Create().
			SetTenant(tenant).
			SetUsername(username).
			SetPasswordHash(hash).
			SetDisplayName(username).
			SetIsAdmin(admin).
			SetCreatedAt(time.Now().UTC()).
			Exec(ctx)
	} else if err == nil {
		err = existing.Update().
			SetPasswordHash(hash).
			SetIsAdmin(admin).
			Exec(ctx)
	}
	if err != nil {
		return err
	}
	_, _ = s.LoadUser(tenant)
	return nil
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
	rec := UserRecord{
		Tenant:       row.Tenant,
		Username:     row.Username,
		PasswordHash: row.PasswordHash,
		DisplayName:  row.DisplayName,
		AvatarURL:    row.AvatarURL,
		IsAdmin:      row.IsAdmin,
		CreatedAt:    row.CreatedAt,
	}
	return rec, ok, nil
}

func (s *Store) UpsertExternalUser(tenant, username, displayName, avatarURL string) (UserRecord, error) {
	tenant = TenantKey(tenant)
	username = strings.TrimSpace(username)
	if username == "" {
		username = tenant
	}
	if tenant == "" {
		return UserRecord{}, fmt.Errorf("tenant is required")
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = username
	}
	if err := ValidateAvatarURL(avatarURL); err != nil {
		avatarURL = ""
	}
	ctx := context.Background()
	row, err := s.client.User.Query().Where(user.Tenant(tenant)).Only(ctx)
	if ent.IsNotFound(err) {
		err = s.client.User.Create().
			SetTenant(tenant).
			SetUsername(username).
			SetPasswordHash("external:oidc").
			SetDisplayName(displayName).
			SetAvatarURL(strings.TrimSpace(avatarURL)).
			SetIsAdmin(false).
			SetCreatedAt(time.Now().UTC()).
			Exec(ctx)
		if err != nil {
			return UserRecord{}, err
		}
		row, err = s.client.User.Query().Where(user.Tenant(tenant)).Only(ctx)
	} else if err == nil {
		err = row.Update().
			SetDisplayName(displayName).
			SetAvatarURL(strings.TrimSpace(avatarURL)).
			Exec(ctx)
		if err == nil {
			row, err = s.client.User.Query().Where(user.Tenant(tenant)).Only(ctx)
		}
	}
	if err != nil {
		return UserRecord{}, err
	}
	return UserRecord{
		Tenant:       row.Tenant,
		Username:     row.Username,
		PasswordHash: row.PasswordHash,
		DisplayName:  row.DisplayName,
		AvatarURL:    row.AvatarURL,
		IsAdmin:      row.IsAdmin,
		CreatedAt:    row.CreatedAt,
	}, nil
}

func (s *Store) Profile(tenant string) (Profile, error) {
	row, err := s.client.User.Query().Where(user.Tenant(TenantKey(tenant))).Only(context.Background())
	if err != nil {
		return Profile{}, err
	}
	return profileFromUser(row), nil
}

func (s *Store) UpdateProfile(tenant, displayName, avatarURL string) (Profile, error) {
	displayName = strings.TrimSpace(displayName)
	avatarURL = strings.TrimSpace(avatarURL)
	if displayName == "" {
		return Profile{}, fmt.Errorf("display name is required")
	}
	if err := ValidateAvatarURL(avatarURL); err != nil {
		return Profile{}, err
	}
	ctx := context.Background()
	row, err := s.client.User.Query().Where(user.Tenant(TenantKey(tenant))).Only(ctx)
	if err != nil {
		return Profile{}, err
	}
	if err := row.Update().
		SetDisplayName(displayName).
		SetAvatarURL(avatarURL).
		Exec(ctx); err != nil {
		return Profile{}, err
	}
	row, err = s.client.User.Query().Where(user.Tenant(TenantKey(tenant))).Only(ctx)
	if err != nil {
		return Profile{}, err
	}
	return profileFromUser(row), nil
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

func profileFromUser(row *ent.User) Profile {
	displayName := row.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName = row.Username
	}
	return Profile{
		Tenant:      row.Tenant,
		Username:    row.Username,
		DisplayName: displayName,
		AvatarURL:   row.AvatarURL,
		IsAdmin:     row.IsAdmin,
	}
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
	user := Normalize(WithUserConfig(system, cfg.UserConfig())).UserConfig()
	merged := WithUserConfig(system, user)
	if err := Validate(merged); err != nil {
		return err
	}
	data, err := json.MarshalIndent(Redacted(WithUserConfig(system, user)).UserConfig(), "", "  ")
	if err != nil {
		return err
	}
	ctx := context.Background()
	tenant = TenantKey(tenant)
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
	system.Export = Config{}.Export
	system.Sources = nil
	system.Schedules = nil
	system.Targets = nil
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
		Export: defaultExportConfig(),
		Sources: []SourceConfig{
			{
				Key:     "local-mem",
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
			{Key: "daily", Name: "Daily", Enabled: true, Type: "daily", Time: "03:00"},
			{Key: "weekly", Name: "Weekly", Enabled: false, Type: "weekly", Time: "03:00", Weekday: "sunday"},
		},
		Targets: []TargetConfig{},
		Tasks: []TaskConfig{
			{
				Key:          "default",
				Name:         "Default backup",
				Enabled:      false,
				SourceKey:    "local-mem",
				ScheduleKey:  "daily",
				TargetKeys:   []string{},
				ObjectPrefix: "nowledge-mem/{task}/{timestamp}",
				Encryption: EncryptionConfig{
					Enabled:     false,
					PasswordEnv: "NMEM_SNAP_ENCRYPTION_PASSWORD",
				},
				Retention: RetentionConfig{
					Mode: "none",
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
	cfg.Export = mergeExportConfig(defaultExportConfig(), cfg.Export)
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
	}
	for i := range cfg.Tasks {
		cfg.Tasks[i].Key = strings.TrimSpace(cfg.Tasks[i].Key)
		cfg.Tasks[i].Name = defaultString(strings.TrimSpace(cfg.Tasks[i].Name), cfg.Tasks[i].Key)
		cfg.Tasks[i].SourceKey = strings.TrimSpace(cfg.Tasks[i].SourceKey)
		cfg.Tasks[i].ScheduleKey = strings.TrimSpace(cfg.Tasks[i].ScheduleKey)
		cfg.Tasks[i].ObjectPrefix = defaultString(strings.TrimSpace(cfg.Tasks[i].ObjectPrefix), "nowledge-mem/{task}/{timestamp}")
		cfg.Tasks[i].Encryption.PasswordEnv = defaultString(strings.TrimSpace(cfg.Tasks[i].Encryption.PasswordEnv), "NMEM_SNAP_ENCRYPTION_PASSWORD")
		cfg.Tasks[i].Retention.Mode = strings.ToLower(strings.TrimSpace(cfg.Tasks[i].Retention.Mode))
		if cfg.Tasks[i].Retention.Mode == "" {
			cfg.Tasks[i].Retention.Mode = "none"
		}
	}
	return cfg
}

func (c Config) UserConfig() UserConfig {
	return UserConfig{
		Export:               c.Export,
		Sources:              cloneSlice(c.Sources),
		Schedules:            cloneSlice(c.Schedules),
		Targets:              cloneSlice(c.Targets),
		Tasks:                cloneSlice(c.Tasks),
		HistoryLimit:         c.HistoryLimit,
		HistoryRetentionDays: c.HistoryRetentionDays,
	}
}

func WithUserConfig(system Config, user UserConfig) Config {
	system.Export = user.Export
	system.Sources = cloneSlice(user.Sources)
	system.Schedules = cloneSlice(user.Schedules)
	system.Targets = cloneSlice(user.Targets)
	system.Tasks = cloneSlice(user.Tasks)
	system.HistoryLimit = user.HistoryLimit
	system.HistoryRetentionDays = user.HistoryRetentionDays
	return system
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
	if v := strings.TrimSpace(os.Getenv("NMEM_API_URL")); v != "" {
		for i := range cfg.Sources {
			if cfg.Sources[i].Type == "nowledgemem_api" && cfg.Sources[i].Key == "local-mem" {
				cfg.Sources[i].NowledgeMem.APIURL = v
			}
		}
	}
	if v := strings.TrimSpace(os.Getenv("NMEM_API_KEY")); v != "" {
		for i := range cfg.Sources {
			if cfg.Sources[i].Type == "nowledgemem_api" && cfg.Sources[i].Key == "local-mem" {
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
			cfg.Targets[i].S3.SecretAccessKey = os.Getenv(cfg.Targets[i].S3.SecretAccessKeyEnv)
		case "webdav":
			cfg.Targets[i].WebDAV.Password = os.Getenv(cfg.Targets[i].WebDAV.PasswordEnv)
		}
	}
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
	}
	return cfg
}

func Validate(cfg Config) error {
	if cfg.Listen.Port < 1 || cfg.Listen.Port > 65535 {
		return fmt.Errorf("listen port must be between 1 and 65535")
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
		if err := validateKey("target", target.Key); err != nil {
			return err
		}
		if _, ok := targetKeys[target.Key]; ok {
			return fmt.Errorf("duplicate target key %q", target.Key)
		}
		targetKeys[target.Key] = struct{}{}
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
		default:
			return fmt.Errorf("target %q type must be s3 or webdav", target.Key)
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
		switch task.Retention.Mode {
		case "", "none":
		case "keep_last":
			if task.Retention.KeepLast < 1 {
				return fmt.Errorf("task %q retention keep_last must be at least 1", task.Key)
			}
		case "keep_days":
			if task.Retention.KeepDays < 1 {
				return fmt.Errorf("task %q retention keep_days must be at least 1", task.Key)
			}
		case "keep_after":
			if strings.TrimSpace(task.Retention.KeepAfter) == "" {
				return fmt.Errorf("task %q retention keep_after is required", task.Key)
			}
			if _, err := parseDate(task.Retention.KeepAfter); err != nil {
				return fmt.Errorf("task %q retention keep_after: %w", task.Key, err)
			}
		case "keep_before":
			if strings.TrimSpace(task.Retention.KeepBefore) == "" {
				return fmt.Errorf("task %q retention keep_before is required", task.Key)
			}
			if _, err := parseDate(task.Retention.KeepBefore); err != nil {
				return fmt.Errorf("task %q retention keep_before: %w", task.Key, err)
			}
		default:
			return fmt.Errorf("task %q retention mode must be none, keep_last, keep_days, keep_after, or keep_before", task.Key)
		}
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

func (t TaskConfig) EffectiveExport(global ExportConfig) ExportConfig {
	return mergeExportConfig(global, t.Export)
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
