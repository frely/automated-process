package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestInitLegacySingleAccountCompatible(t *testing.T) {
	withTempWorkdir(t, func(dir string) {
		viper.Reset()
		defer viper.Reset()

		content := `
WEB_SERVER_ADDR: ":8080"
WEB_REFRESH_SECONDS: 60
WEB_LOG_LEVEL: "warning"
ALIBABA_CLOUD_ACCESS_KEY_ID: "ak-single"
ALIBABA_CLOUD_ACCESS_KEY_SECRET: "sk-single"
ALIBABA_CLOUD_CMS_REGION: "cn-chengdu"
ALIBABA_CLOUD_CMS_PAGE: 1
ALIBABA_CLOUD_CMS_PAGE_SIZE: 20
ALIBABA_CLOUD_CMS_TIMEOUT_SECONDS: 10
`
		mustWriteFile(t, filepath.Join(dir, "config.yaml"), content)

		cfg, err := Init()
		if err != nil {
			t.Fatalf("Init returned error: %v", err)
		}
		if len(cfg.CMSAccounts) != 1 {
			t.Fatalf("expected 1 legacy account, got %d", len(cfg.CMSAccounts))
		}
		if cfg.CMSAccounts[0].Key != "default" {
			t.Fatalf("unexpected account key: %s", cfg.CMSAccounts[0].Key)
		}
		if cfg.CMSAccounts[0].AccessKeyID != "ak-single" {
			t.Fatalf("unexpected access key id: %s", cfg.CMSAccounts[0].AccessKeyID)
		}
	})
}

func TestInitMultiAccountsFromEnv(t *testing.T) {
	withTempWorkdir(t, func(dir string) {
		viper.Reset()
		defer viper.Reset()

		t.Setenv("MONITOR_ACCOUNT_A_AK", "ak-a")
		t.Setenv("MONITOR_ACCOUNT_A_SK", "sk-a")
		t.Setenv("MONITOR_ACCOUNT_B_AK", "ak-b")
		t.Setenv("MONITOR_ACCOUNT_B_SK", "sk-b")

		content := `
WEB_SERVER_ADDR: ":8080"
WEB_REFRESH_SECONDS: 60
WEB_LOG_LEVEL: "warning"
ALIBABA_CLOUD_CMS_REGION: "cn-chengdu"
ALIBABA_CLOUD_CMS_ACCOUNTS:
  - key: "prod-a"
    name: "生产A"
    region: "cn-chengdu"
    access_key_id_env: "MONITOR_ACCOUNT_A_AK"
    access_key_secret_env: "MONITOR_ACCOUNT_A_SK"
    page: 1
    page_size: 20
    timeout_seconds: 10
  - key: "prod-b"
    name: "生产B"
    region: "cn-hangzhou"
    access_key_id_env: "MONITOR_ACCOUNT_B_AK"
    access_key_secret_env: "MONITOR_ACCOUNT_B_SK"
    page: 1
    page_size: 20
    timeout_seconds: 10
`
		mustWriteFile(t, filepath.Join(dir, "config.yaml"), content)

		cfg, err := Init()
		if err != nil {
			t.Fatalf("Init returned error: %v", err)
		}
		if len(cfg.CMSAccounts) != 2 {
			t.Fatalf("expected 2 accounts, got %d", len(cfg.CMSAccounts))
		}
		if cfg.CMSAccounts[0].AccessKeyID != "ak-a" || cfg.CMSAccounts[0].AccessKeySecret != "sk-a" {
			t.Fatal("unexpected account A credentials")
		}
		if cfg.CMSAccounts[1].AccessKeyID != "ak-b" || cfg.CMSAccounts[1].AccessKeySecret != "sk-b" {
			t.Fatal("unexpected account B credentials")
		}
	})
}

func TestInitMultiAccountsMissingCredential(t *testing.T) {
	withTempWorkdir(t, func(dir string) {
		viper.Reset()
		defer viper.Reset()

		content := `
WEB_SERVER_ADDR: ":8080"
WEB_REFRESH_SECONDS: 60
WEB_LOG_LEVEL: "warning"
ALIBABA_CLOUD_CMS_REGION: "cn-chengdu"
ALIBABA_CLOUD_CMS_ACCOUNTS:
  - key: "prod-a"
    name: "生产A"
    region: "cn-chengdu"
    access_key_id_env: "MISSING_ACCOUNT_A_AK"
    access_key_secret_env: "MISSING_ACCOUNT_A_SK"
`
		mustWriteFile(t, filepath.Join(dir, "config.yaml"), content)

		_, err := Init()
		if err == nil {
			t.Fatal("expected error when account credential is missing")
		}
		if !strings.Contains(err.Error(), "prod-a") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func withTempWorkdir(t *testing.T, fn func(dir string)) {
	t.Helper()
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()
	fn(dir)
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}
}
