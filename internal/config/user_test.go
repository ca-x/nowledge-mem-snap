package config

import "testing"

func TestUpsertOIDCUserDoesNotMergeLocalUserByUsername(t *testing.T) {
	store := NewStore(t.TempDir())
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	if err := store.CreateLocalUser("czyt", "password", true); err != nil {
		t.Fatalf("CreateLocalUser: %v", err)
	}

	oidcUser, err := store.UpsertOIDCUser(OIDCProfile{
		Issuer:            "https://issuer.example",
		Subject:           "subject-123",
		Email:             "czyt@example.com",
		PreferredUsername: "czyt",
		DisplayName:       "OIDC Czyt",
	})
	if err != nil {
		t.Fatalf("UpsertOIDCUser: %v", err)
	}
	if oidcUser.Tenant == "czyt" {
		t.Fatalf("OIDC user tenant = %q, want a separate tenant", oidcUser.Tenant)
	}
	if oidcUser.Username == "czyt" {
		t.Fatalf("OIDC username = %q, want a unique username", oidcUser.Username)
	}

	local, err := store.Profile("czyt")
	if err != nil {
		t.Fatalf("Profile: %v", err)
	}
	if local.OIDC.Linked {
		t.Fatal("local user was silently linked to OIDC")
	}

	linked, ok, err := store.OIDCUser("https://issuer.example", "subject-123")
	if err != nil {
		t.Fatalf("OIDCUser: %v", err)
	}
	if !ok {
		t.Fatal("OIDC user was not found by issuer/subject")
	}
	if linked.Tenant != oidcUser.Tenant {
		t.Fatalf("linked tenant = %q, want %q", linked.Tenant, oidcUser.Tenant)
	}
}

func TestUpsertOIDCUserLinksLocalUserByVerifiedEmail(t *testing.T) {
	store := NewStore(t.TempDir())
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	if err := store.CreateLocalUser("local", "password", true); err != nil {
		t.Fatalf("CreateLocalUser: %v", err)
	}
	if _, err := store.UpdateUser("local", UserUpdate{
		Username:    "local",
		Email:       "czyt@example.com",
		DisplayName: "Local User",
		IsAdmin:     true,
	}); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	oidcUser, err := store.UpsertOIDCUser(OIDCProfile{
		Issuer:            "https://issuer.example",
		Subject:           "subject-123",
		Email:             "czyt@example.com",
		EmailVerified:     true,
		PreferredUsername: "different",
		DisplayName:       "OIDC Czyt",
	})
	if err != nil {
		t.Fatalf("UpsertOIDCUser: %v", err)
	}
	if oidcUser.Tenant != "local" {
		t.Fatalf("OIDC user tenant = %q, want local", oidcUser.Tenant)
	}

	profile, err := store.Profile("local")
	if err != nil {
		t.Fatalf("Profile: %v", err)
	}
	if !profile.OIDC.Linked {
		t.Fatal("local user was not linked to OIDC")
	}
	if profile.OIDC.Email != "czyt@example.com" {
		t.Fatalf("OIDC email = %q, want czyt@example.com", profile.OIDC.Email)
	}
}

func TestUpsertOIDCUserDoesNotLinkLocalUserByUnverifiedEmail(t *testing.T) {
	store := NewStore(t.TempDir())
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	if err := store.CreateLocalUser("local", "password", true); err != nil {
		t.Fatalf("CreateLocalUser: %v", err)
	}
	if _, err := store.UpdateUser("local", UserUpdate{
		Username:    "local",
		Email:       "czyt@example.com",
		DisplayName: "Local User",
		IsAdmin:     true,
	}); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	oidcUser, err := store.UpsertOIDCUser(OIDCProfile{
		Issuer:            "https://issuer.example",
		Subject:           "subject-123",
		Email:             "czyt@example.com",
		EmailVerified:     false,
		PreferredUsername: "different",
		DisplayName:       "OIDC Czyt",
	})
	if err != nil {
		t.Fatalf("UpsertOIDCUser: %v", err)
	}
	if oidcUser.Tenant == "local" {
		t.Fatalf("OIDC user tenant = local, want separate user")
	}

	profile, err := store.Profile("local")
	if err != nil {
		t.Fatalf("Profile: %v", err)
	}
	if profile.OIDC.Linked {
		t.Fatal("local user was linked despite unverified OIDC email")
	}
}

func TestDeleteUserRejectsLastAdministrator(t *testing.T) {
	store := NewStore(t.TempDir())
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	if err := store.CreateLocalUser("admin", "password", true); err != nil {
		t.Fatalf("CreateLocalUser: %v", err)
	}
	if err := store.DeleteUser("admin"); err == nil {
		t.Fatal("DeleteUser deleted the last administrator")
	}
}
