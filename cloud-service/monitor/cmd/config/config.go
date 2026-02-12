package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ErrConfigTemplateCreated 表示首次运行已生成配置模板并安全退出。
var ErrConfigTemplateCreated = errors.New("config template created")

// AppConfig 是监控 Web 服务的运行配置。
type AppConfig struct {
	ServerAddr     string
	RefreshSeconds int
	LogLevel       string
	CMS            CMSConfig
	CMSAccounts    []CMSAccountConfig
}

// CMSConfig 描述阿里云云监控查询参数和鉴权配置（兼容旧版单账号）。
type CMSConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	RegionID        string
	RuleID          string
	Namespace       string
	GroupID         string
	Dimensions      string
	AlertBeforeUTC  string
	Page            int32
	PageSize        int32
	Timeout         time.Duration
}

// CMSAccountConfig 描述一个账号的查询参数和鉴权信息。
type CMSAccountConfig struct {
	Key             string
	Name            string
	AccessKeyID     string
	AccessKeySecret string
	RegionID        string
	RuleID          string
	Namespace       string
	GroupID         string
	Dimensions      string
	AlertBeforeUTC  string
	Page            int32
	PageSize        int32
	Timeout         time.Duration
}

type cmsAccountTemplate struct {
	Key                string `mapstructure:"key"`
	Name               string `mapstructure:"name"`
	RegionID           string `mapstructure:"region"`
	AccessKeyID        string `mapstructure:"access_key_id"`
	AccessKeySecret    string `mapstructure:"access_key_secret"`
	AccessKeyIDEnv     string `mapstructure:"access_key_id_env"`
	AccessKeySecretEnv string `mapstructure:"access_key_secret_env"`
	RuleID             string `mapstructure:"rule_id"`
	Namespace          string `mapstructure:"namespace"`
	GroupID            string `mapstructure:"group_id"`
	Dimensions         string `mapstructure:"dimensions"`
	AlertBeforeUTC     string `mapstructure:"alert_before_utc"`
	Page               int    `mapstructure:"page"`
	PageSize           int    `mapstructure:"page_size"`
	TimeoutSeconds     int    `mapstructure:"timeout_seconds"`
}

// Init 负责加载配置、填充默认值并校验关键项。
func Init() (AppConfig, error) {
	setDefaults()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			if writeErr := viper.SafeWriteConfigAs("config.yaml"); writeErr != nil {
				return AppConfig{}, fmt.Errorf("write config.yaml: %w", writeErr)
			}
			return AppConfig{}, ErrConfigTemplateCreated
		}
		return AppConfig{}, fmt.Errorf("read config.yaml: %w", err)
	}

	cfg := AppConfig{
		ServerAddr:     strings.TrimSpace(viper.GetString("WEB_SERVER_ADDR")),
		RefreshSeconds: viper.GetInt("WEB_REFRESH_SECONDS"),
		LogLevel:       strings.ToLower(strings.TrimSpace(viper.GetString("WEB_LOG_LEVEL"))),
		CMS: CMSConfig{
			AccessKeyID:     strings.TrimSpace(viper.GetString("ALIBABA_CLOUD_ACCESS_KEY_ID")),
			AccessKeySecret: strings.TrimSpace(viper.GetString("ALIBABA_CLOUD_ACCESS_KEY_SECRET")),
			RegionID:        strings.TrimSpace(viper.GetString("ALIBABA_CLOUD_CMS_REGION")),
			RuleID:          strings.TrimSpace(viper.GetString("ALIBABA_CLOUD_CMS_RULE_ID")),
			Namespace:       strings.TrimSpace(viper.GetString("ALIBABA_CLOUD_CMS_NAMESPACE")),
			GroupID:         strings.TrimSpace(viper.GetString("ALIBABA_CLOUD_CMS_GROUP_ID")),
			Dimensions:      strings.TrimSpace(viper.GetString("ALIBABA_CLOUD_CMS_DIMENSIONS")),
			AlertBeforeUTC:  strings.TrimSpace(viper.GetString("ALIBABA_CLOUD_CMS_ALERT_BEFORE_UTC")),
			Page:            int32(viper.GetInt("ALIBABA_CLOUD_CMS_PAGE")),
			PageSize:        int32(viper.GetInt("ALIBABA_CLOUD_CMS_PAGE_SIZE")),
			Timeout:         time.Duration(viper.GetInt("ALIBABA_CLOUD_CMS_TIMEOUT_SECONDS")) * time.Second,
		},
	}

	if cfg.ServerAddr == "" {
		cfg.ServerAddr = ":8080"
	}
	if cfg.RefreshSeconds <= 0 {
		cfg.RefreshSeconds = 60
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "warning"
	}
	if cfg.CMS.Page <= 0 {
		cfg.CMS.Page = 1
	}
	if cfg.CMS.PageSize <= 0 {
		cfg.CMS.PageSize = 20
	}
	if cfg.CMS.Timeout < time.Second {
		cfg.CMS.Timeout = 10 * time.Second
	}
	if cfg.CMS.RegionID == "" {
		cfg.CMS.RegionID = "cn-chengdu"
	}

	accounts, err := loadCMSAccounts(cfg.CMS)
	if err != nil {
		return AppConfig{}, err
	}
	cfg.CMSAccounts = accounts

	return cfg, nil
}

// loadCMSAccounts 优先读取多账号配置；若未配置则回退到旧版单账号配置。
func loadCMSAccounts(defaultCMS CMSConfig) ([]CMSAccountConfig, error) {
	var templates []cmsAccountTemplate
	if err := viper.UnmarshalKey("ALIBABA_CLOUD_CMS_ACCOUNTS", &templates); err != nil {
		return nil, fmt.Errorf("decode ALIBABA_CLOUD_CMS_ACCOUNTS: %w", err)
	}

	if len(templates) == 0 {
		if defaultCMS.AccessKeyID == "" {
			return nil, errors.New("missing ALIBABA_CLOUD_ACCESS_KEY_ID")
		}
		if defaultCMS.AccessKeySecret == "" {
			return nil, errors.New("missing ALIBABA_CLOUD_ACCESS_KEY_SECRET")
		}
		return []CMSAccountConfig{buildLegacySingleAccount(defaultCMS)}, nil
	}

	accounts := make([]CMSAccountConfig, 0, len(templates))
	seenKeys := make(map[string]struct{}, len(templates))
	for idx, template := range templates {
		account, err := normalizeAccountTemplate(template, defaultCMS, idx)
		if err != nil {
			return nil, err
		}
		if _, existed := seenKeys[account.Key]; existed {
			return nil, fmt.Errorf("duplicate account key %q in ALIBABA_CLOUD_CMS_ACCOUNTS", account.Key)
		}
		seenKeys[account.Key] = struct{}{}
		accounts = append(accounts, account)
	}

	return accounts, nil
}

func buildLegacySingleAccount(defaultCMS CMSConfig) CMSAccountConfig {
	return CMSAccountConfig{
		Key:             "default",
		Name:            "default",
		AccessKeyID:     strings.TrimSpace(defaultCMS.AccessKeyID),
		AccessKeySecret: strings.TrimSpace(defaultCMS.AccessKeySecret),
		RegionID:        strings.TrimSpace(defaultCMS.RegionID),
		RuleID:          strings.TrimSpace(defaultCMS.RuleID),
		Namespace:       strings.TrimSpace(defaultCMS.Namespace),
		GroupID:         strings.TrimSpace(defaultCMS.GroupID),
		Dimensions:      strings.TrimSpace(defaultCMS.Dimensions),
		AlertBeforeUTC:  strings.TrimSpace(defaultCMS.AlertBeforeUTC),
		Page:            defaultCMS.Page,
		PageSize:        defaultCMS.PageSize,
		Timeout:         defaultCMS.Timeout,
	}
}

func normalizeAccountTemplate(template cmsAccountTemplate, defaultCMS CMSConfig, index int) (CMSAccountConfig, error) {
	account := CMSAccountConfig{
		Key:            strings.TrimSpace(template.Key),
		Name:           strings.TrimSpace(template.Name),
		RegionID:       strings.TrimSpace(template.RegionID),
		RuleID:         strings.TrimSpace(template.RuleID),
		Namespace:      strings.TrimSpace(template.Namespace),
		GroupID:        strings.TrimSpace(template.GroupID),
		Dimensions:     strings.TrimSpace(template.Dimensions),
		AlertBeforeUTC: strings.TrimSpace(template.AlertBeforeUTC),
	}
	if account.Key == "" {
		account.Key = "account-" + strconv.Itoa(index+1)
	}
	if account.Name == "" {
		account.Name = account.Key
	}
	if account.RegionID == "" {
		account.RegionID = defaultCMS.RegionID
	}
	if account.RegionID == "" {
		account.RegionID = "cn-chengdu"
	}

	account.Page = int32(template.Page)
	if account.Page <= 0 {
		account.Page = defaultCMS.Page
	}
	if account.Page <= 0 {
		account.Page = 1
	}

	account.PageSize = int32(template.PageSize)
	if account.PageSize <= 0 {
		account.PageSize = defaultCMS.PageSize
	}
	if account.PageSize <= 0 {
		account.PageSize = 20
	}

	if template.TimeoutSeconds > 0 {
		account.Timeout = time.Duration(template.TimeoutSeconds) * time.Second
	} else {
		account.Timeout = defaultCMS.Timeout
	}
	if account.Timeout < time.Second {
		account.Timeout = 10 * time.Second
	}

	account.AccessKeyID = resolveCredential(template.AccessKeyID, template.AccessKeyIDEnv)
	account.AccessKeySecret = resolveCredential(template.AccessKeySecret, template.AccessKeySecretEnv)
	if account.AccessKeyID == "" {
		return CMSAccountConfig{}, fmt.Errorf("missing access key id for account %q: set access_key_id or access_key_id_env", account.Key)
	}
	if account.AccessKeySecret == "" {
		return CMSAccountConfig{}, fmt.Errorf("missing access key secret for account %q: set access_key_secret or access_key_secret_env", account.Key)
	}

	return account, nil
}

func resolveCredential(value string, envKey string) string {
	envKey = strings.TrimSpace(envKey)
	if envKey != "" {
		return strings.TrimSpace(os.Getenv(envKey))
	}
	return strings.TrimSpace(value)
}

// setDefaults 写入配置模板和环境变量默认值。
func setDefaults() {
	viper.SetDefault("WEB_SERVER_ADDR", ":8080")
	viper.SetDefault("WEB_REFRESH_SECONDS", 60)
	viper.SetDefault("WEB_LOG_LEVEL", "warning")

	viper.SetDefault("ALIBABA_CLOUD_ACCESS_KEY_ID", "")
	viper.SetDefault("ALIBABA_CLOUD_ACCESS_KEY_SECRET", "")
	viper.SetDefault("ALIBABA_CLOUD_CMS_REGION", "cn-chengdu")
	viper.SetDefault("ALIBABA_CLOUD_CMS_RULE_ID", "")
	viper.SetDefault("ALIBABA_CLOUD_CMS_NAMESPACE", "")
	viper.SetDefault("ALIBABA_CLOUD_CMS_GROUP_ID", "")
	viper.SetDefault("ALIBABA_CLOUD_CMS_DIMENSIONS", "")
	viper.SetDefault("ALIBABA_CLOUD_CMS_ALERT_BEFORE_UTC", "")
	viper.SetDefault("ALIBABA_CLOUD_CMS_PAGE", 1)
	viper.SetDefault("ALIBABA_CLOUD_CMS_PAGE_SIZE", 20)
	viper.SetDefault("ALIBABA_CLOUD_CMS_TIMEOUT_SECONDS", 10)
	viper.SetDefault("ALIBABA_CLOUD_CMS_ACCOUNTS", []any{})
}
