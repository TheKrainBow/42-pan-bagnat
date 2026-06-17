package core

import (
	"backend/database"
	"os"
	"sync"
	"testing"
)

func resetOIDCTestConfig() {
	oidcConfigOnce = sync.Once{}
	oidcConfig = OIDCConfig{}
	oidcConfigErr = nil
}

func TestNormalizeClientID(t *testing.T) {
	if got := normalizeClientID("piscine-monitoring"); got != "panbagnat_piscine_monitoring" {
		t.Fatalf("unexpected client id: %s", got)
	}
}

func TestOIDCDiscoveryAndJWKS(t *testing.T) {
	t.Setenv("OIDC_ISSUER", "https://example.com")
	t.Setenv("OIDC_JWT_KEY_ID", "test-kid")
	resetOIDCTestConfig()

	doc, err := BuildOIDCDiscoveryDocument()
	if err != nil {
		t.Fatalf("BuildOIDCDiscoveryDocument: %v", err)
	}
	if got := doc["issuer"]; got != "https://example.com" {
		t.Fatalf("unexpected issuer: %v", got)
	}
	if got := doc["authorization_endpoint"]; got != "https://example.com/oauth/authorize" {
		t.Fatalf("unexpected authorization endpoint: %v", got)
	}

	jwks, err := BuildOIDCJWKS()
	if err != nil {
		t.Fatalf("BuildOIDCJWKS: %v", err)
	}
	keys, ok := jwks["keys"].([]map[string]any)
	if !ok || len(keys) != 1 {
		t.Fatalf("unexpected JWKS shape: %#v", jwks)
	}
	if keys[0]["kid"] != "test-kid" {
		t.Fatalf("unexpected kid: %v", keys[0]["kid"])
	}
	if keys[0]["kty"] != "RSA" {
		t.Fatalf("unexpected key type: %v", keys[0]["kty"])
	}
}

func TestBuildOIDCUserClaimsEmailFromLogin(t *testing.T) {
	module := Module{ID: "module_01", Slug: "wiki-test", Name: "Wiki test"}
	client := database.OIDCClient{}

	t.Run("student", func(t *testing.T) {
		user := User{ID: "user_01", FtLogin: "heinz", FtIsStaff: false}
		claims := BuildOIDCUserClaims(user, module, client, []string{"openid", "email"}, "")
		if got := claims.Email; got != "heinz@student.42nice.fr" {
			t.Fatalf("unexpected email: %s", got)
		}
		if !claims.EmailVerified {
			t.Fatalf("expected email_verified to be true")
		}
	})

	t.Run("staff", func(t *testing.T) {
		user := User{ID: "user_02", FtLogin: "heinz", FtIsStaff: true}
		claims := BuildOIDCUserClaims(user, module, client, []string{"openid", "email"}, "")
		if got := claims.Email; got != "heinz@42nice.fr" {
			t.Fatalf("unexpected email: %s", got)
		}
		if !claims.EmailVerified {
			t.Fatalf("expected email_verified to be true")
		}
	})
}

func TestMain(m *testing.M) {
	code := m.Run()
	resetOIDCTestConfig()
	os.Exit(code)
}
