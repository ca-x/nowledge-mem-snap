package config

import "testing"

func TestValidateTargetConfigSupportsGCS(t *testing.T) {
	target := TargetConfig{
		Key:  "gcs-backup",
		Type: "gcs",
		GCS: GCSConfig{
			BucketName: "snap-backups",
		},
	}

	if err := ValidateTargetConfig(target); err != nil {
		t.Fatalf("ValidateTargetConfig: %v", err)
	}
}

func TestValidateTargetConfigRequiresSFTPHostKeyByDefault(t *testing.T) {
	target := TargetConfig{
		Key:  "sftp-backup",
		Type: "sftp",
		SFTP: SFTPConfig{
			Host:        "sftp.example.com",
			Port:        22,
			Username:    "backup",
			PasswordEnv: "SFTP_PASSWORD",
		},
	}

	if err := ValidateTargetConfig(target); err == nil {
		t.Fatal("ValidateTargetConfig accepted SFTP without host key verification")
	}

	target.SFTP.HostKeySHA256 = "abc123"
	if err := ValidateTargetConfig(target); err != nil {
		t.Fatalf("ValidateTargetConfig with host key: %v", err)
	}

	target.SFTP.HostKeySHA256 = ""
	target.SFTP.InsecureIgnoreHostKey = true
	if err := ValidateTargetConfig(target); err != nil {
		t.Fatalf("ValidateTargetConfig with insecure host key bypass: %v", err)
	}
}

func TestApplyEnvLoadsGCSAndSFTPSecrets(t *testing.T) {
	t.Setenv("TEST_GCS_JSON", `{"type":"service_account"}`)
	t.Setenv("TEST_SFTP_PASSWORD", "password")
	t.Setenv("TEST_SFTP_PRIVATE_KEY", "private-key")
	t.Setenv("TEST_SFTP_PRIVATE_KEY_PASSPHRASE", "passphrase")

	cfg := ApplyEnv(Config{
		Targets: []TargetConfig{
			{
				Key:  "gcs-backup",
				Type: "gcs",
				GCS: GCSConfig{
					BucketName:         "snap-backups",
					CredentialsJSONEnv: "TEST_GCS_JSON",
				},
			},
			{
				Key:  "sftp-backup",
				Type: "sftp",
				SFTP: SFTPConfig{
					Host:                    "sftp.example.com",
					Username:                "backup",
					PasswordEnv:             "TEST_SFTP_PASSWORD",
					PrivateKeyEnv:           "TEST_SFTP_PRIVATE_KEY",
					PrivateKeyPassphraseEnv: "TEST_SFTP_PRIVATE_KEY_PASSPHRASE",
					InsecureIgnoreHostKey:   true,
				},
			},
		},
	})

	if got := cfg.Targets[0].GCS.CredentialsJSON; got != `{"type":"service_account"}` {
		t.Fatalf("GCS credentials_json = %q", got)
	}
	sftp := cfg.Targets[1].SFTP
	if sftp.Port != 22 {
		t.Fatalf("SFTP port = %d, want 22", sftp.Port)
	}
	if sftp.Password != "password" || sftp.PrivateKey != "private-key" || sftp.PrivateKeyPassphrase != "passphrase" {
		t.Fatalf("SFTP secrets were not loaded from env: %#v", sftp)
	}
}

func TestRedactedStripsGCSAndSFTPSecrets(t *testing.T) {
	cfg := Redacted(Config{
		Targets: []TargetConfig{
			{
				Key:  "gcs-backup",
				Type: "gcs",
				GCS:  GCSConfig{CredentialsJSON: "secret-json"},
			},
			{
				Key:  "sftp-backup",
				Type: "sftp",
				SFTP: SFTPConfig{
					Password:             "secret-password",
					PrivateKey:           "secret-key",
					PrivateKeyPassphrase: "secret-passphrase",
				},
			},
		},
	})

	if cfg.Targets[0].GCS.CredentialsJSON != "" {
		t.Fatal("GCS credentials_json was not redacted")
	}
	sftp := cfg.Targets[1].SFTP
	if sftp.Password != "" || sftp.PrivateKey != "" || sftp.PrivateKeyPassphrase != "" {
		t.Fatalf("SFTP secrets were not redacted: %#v", sftp)
	}
}

func TestMergeTargetSecretsKeepsGCSAndSFTPSecrets(t *testing.T) {
	gcs := MergeTargetSecrets(
		TargetConfig{Key: "gcs-backup", Type: "gcs"},
		TargetConfig{Key: "gcs-backup", Type: "gcs", GCS: GCSConfig{CredentialsJSON: "secret-json"}},
	)
	if gcs.GCS.CredentialsJSON != "secret-json" {
		t.Fatalf("GCS credentials_json = %q", gcs.GCS.CredentialsJSON)
	}

	sftp := MergeTargetSecrets(
		TargetConfig{Key: "sftp-backup", Type: "sftp"},
		TargetConfig{
			Key:  "sftp-backup",
			Type: "sftp",
			SFTP: SFTPConfig{
				Password:             "secret-password",
				PrivateKey:           "secret-key",
				PrivateKeyPassphrase: "secret-passphrase",
			},
		},
	)
	if sftp.SFTP.Password != "secret-password" || sftp.SFTP.PrivateKey != "secret-key" || sftp.SFTP.PrivateKeyPassphrase != "secret-passphrase" {
		t.Fatalf("SFTP secrets were not merged: %#v", sftp.SFTP)
	}
}
