package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	aliyunDescribeAlertingMetricRuleResources "github.com/frely/automated-process/cloud-service/monitor/cmd/aliyunDescribeAlertingMetricRuleResources"
	"github.com/frely/automated-process/cloud-service/monitor/cmd/config"
)

// webAssets 打包 Kindle 页面静态资源，部署时无需额外挂载目录。
//
//go:embed web/static/*
var webAssets embed.FS

// webServer 负责路由、HTTP 处理和调用多账号云监控查询服务。
type webServer struct {
	cfg    config.AppConfig
	cmsSvc *aliyunDescribeAlertingMetricRuleResources.MultiAccountService
	logger *slog.Logger
}

// alertsAPIResponse 是前端展示所需的精简响应结构。
type alertsAPIResponse struct {
	TSUTC           string                                                     `json:"ts_utc"`
	RefreshSeconds  int                                                        `json:"refresh_seconds"`
	Total           int32                                                      `json:"total"`
	FetchedAtUTC    string                                                     `json:"fetched_at_utc"`
	AccountsTotal   int                                                        `json:"accounts_total"`
	AccountsSuccess int                                                        `json:"accounts_success"`
	AccountsFailed  int                                                        `json:"accounts_failed"`
	AccountFailures []aliyunDescribeAlertingMetricRuleResources.AccountFailure `json:"account_failures"`
	Alerts          []aliyunDescribeAlertingMetricRuleResources.Alert          `json:"alerts"`
}

type errorResponse struct {
	TSUTC string `json:"ts_utc"`
	Error string `json:"error"`
}

// main 启动监控 Web 服务：读取配置、初始化 CMS 客户端并暴露查询接口。
func main() {
	bootstrapLogger := newJSONLogger(os.Stdout, slog.LevelWarn)

	cfg, err := config.Init()
	if errors.Is(err, config.ErrConfigTemplateCreated) {
		bootstrapLogger.Warn("config template generated",
			"ts_utc", time.Now().UTC().Format(time.RFC3339Nano),
			"module", "monitor",
			"action", "initConfig",
			"provider", "aliyun",
			"region", "",
			"trace_id", "",
			"duration_ms", 0,
			"err", "",
		)
		return
	}
	if err != nil {
		bootstrapLogger.Error("failed to load config",
			"ts_utc", time.Now().UTC().Format(time.RFC3339Nano),
			"module", "monitor",
			"action", "initConfig",
			"provider", "aliyun",
			"region", "",
			"trace_id", "",
			"duration_ms", 0,
			"err", err.Error(),
		)
		os.Exit(1)
	}

	configuredLogLevel := cfg.LogLevel
	logLevel, normalizedLogLevel, validLogLevel := parseLogLevel(configuredLogLevel)
	cfg.LogLevel = normalizedLogLevel
	logger := newJSONLogger(os.Stdout, logLevel)
	if !validLogLevel {
		logger.Warn("invalid log level, fallback to warning",
			"ts_utc", time.Now().UTC().Format(time.RFC3339Nano),
			"module", "monitor",
			"action", "initLogger",
			"provider", "aliyun",
			"region", logRegionForConfig(cfg),
			"trace_id", "",
			"duration_ms", 0,
			"err", "",
			"configured_log_level", configuredLogLevel,
			"effective_log_level", normalizedLogLevel,
		)
	}

	accountConfigs := make([]aliyunDescribeAlertingMetricRuleResources.AccountConfig, 0, len(cfg.CMSAccounts))
	for _, account := range cfg.CMSAccounts {
		accountConfigs = append(accountConfigs, aliyunDescribeAlertingMetricRuleResources.AccountConfig{
			Key:  account.Key,
			Name: account.Name,
			CMS: aliyunDescribeAlertingMetricRuleResources.Config{
				AccessKeyID:     account.AccessKeyID,
				AccessKeySecret: account.AccessKeySecret,
				RegionID:        account.RegionID,
				RuleID:          account.RuleID,
				Namespace:       account.Namespace,
				GroupID:         account.GroupID,
				Dimensions:      account.Dimensions,
				AlertBeforeUTC:  account.AlertBeforeUTC,
				Page:            account.Page,
				PageSize:        account.PageSize,
				Timeout:         account.Timeout,
			},
		})
	}

	cmsSvc, err := aliyunDescribeAlertingMetricRuleResources.NewMultiAccount(accountConfigs, logger)
	if err != nil {
		logger.Error("failed to initialize aliyun cms service",
			"ts_utc", time.Now().UTC().Format(time.RFC3339Nano),
			"module", "monitor",
			"action", "initCMSService",
			"provider", "aliyun",
			"region", logRegionForConfig(cfg),
			"trace_id", "",
			"duration_ms", 0,
			"err", err.Error(),
		)
		os.Exit(1)
	}

	web := &webServer{
		cfg:    cfg,
		cmsSvc: cmsSvc,
		logger: logger,
	}

	handler, err := web.routes()
	if err != nil {
		logger.Error("failed to build routes",
			"ts_utc", time.Now().UTC().Format(time.RFC3339Nano),
			"module", "monitor",
			"action", "buildRoutes",
			"provider", "aliyun",
			"region", logRegionForConfig(cfg),
			"trace_id", "",
			"duration_ms", 0,
			"err", err.Error(),
		)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	logger.Info("monitor web server started",
		"ts_utc", time.Now().UTC().Format(time.RFC3339Nano),
		"module", "monitor",
		"action", "startHTTPServer",
		"provider", "aliyun",
		"region", logRegionForConfig(cfg),
		"trace_id", "",
		"duration_ms", 0,
		"err", "",
		"addr", cfg.ServerAddr,
		"accounts_total", len(accountConfigs),
	)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("monitor web server stopped unexpectedly",
			"ts_utc", time.Now().UTC().Format(time.RFC3339Nano),
			"module", "monitor",
			"action", "startHTTPServer",
			"provider", "aliyun",
			"region", logRegionForConfig(cfg),
			"trace_id", "",
			"duration_ms", 0,
			"err", err.Error(),
		)
		os.Exit(1)
	}
}

func logRegionForConfig(cfg config.AppConfig) string {
	if len(cfg.CMSAccounts) == 1 {
		return cfg.CMSAccounts[0].RegionID
	}
	return "multi"
}

// routes 注册静态页面、业务 API 和健康检查端点。
func (w *webServer) routes() (http.Handler, error) {
	staticFS, err := fs.Sub(webAssets, "web/static")
	if err != nil {
		return nil, fmt.Errorf("load static assets: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/", w.handleIndex)
	mux.HandleFunc("/api/alerts", w.handleAlerts)
	mux.HandleFunc("/healthz", w.handleHealthz)
	return mux, nil
}

// handleIndex 返回 Kindle 告警列表页面。
func (w *webServer) handleIndex(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeJSON(resp, http.StatusMethodNotAllowed, errorResponse{TSUTC: time.Now().UTC().Format(time.RFC3339Nano), Error: "method not allowed"})
		return
	}
	if req.URL.Path != "/" {
		http.NotFound(resp, req)
		return
	}

	body, err := fs.ReadFile(webAssets, "web/static/index.html")
	if err != nil {
		writeJSON(resp, http.StatusInternalServerError, errorResponse{TSUTC: time.Now().UTC().Format(time.RFC3339Nano), Error: "failed to load page"})
		return
	}

	resp.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = resp.Write(body)
}

// handleAlerts 查询最新告警并输出前端需要的字段。
func (w *webServer) handleAlerts(resp http.ResponseWriter, req *http.Request) {
	startedAt := time.Now().UTC()
	if req.Method != http.MethodGet {
		writeJSON(resp, http.StatusMethodNotAllowed, errorResponse{TSUTC: time.Now().UTC().Format(time.RFC3339Nano), Error: "method not allowed"})
		return
	}

	traceID := strings.TrimSpace(req.Header.Get("X-Trace-Id"))
	if traceID == "" {
		traceID = newTraceID()
	}

	ctx := aliyunDescribeAlertingMetricRuleResources.WithTraceID(req.Context(), traceID)
	result, err := w.cmsSvc.List(ctx)
	if err != nil {
		w.logRequest(ctx, slog.LevelError, traceID, startedAt, err, "api alerts failed")
		writeJSON(resp, http.StatusBadGateway, errorResponse{TSUTC: time.Now().UTC().Format(time.RFC3339Nano), Error: "failed to fetch alerts"})
		return
	}

	level := slog.LevelInfo
	if result.AccountsFailed > 0 {
		level = slog.LevelWarn
	}
	w.logRequest(ctx, level, traceID, startedAt, nil, "api alerts succeeded",
		"total", result.Total,
		"accounts_total", result.AccountsTotal,
		"accounts_success", result.AccountsSuccess,
		"accounts_failed", result.AccountsFailed,
	)

	writeJSON(resp, http.StatusOK, alertsAPIResponse{
		TSUTC:           time.Now().UTC().Format(time.RFC3339Nano),
		RefreshSeconds:  w.cfg.RefreshSeconds,
		Total:           result.Total,
		FetchedAtUTC:    result.FetchedAtUTC,
		AccountsTotal:   result.AccountsTotal,
		AccountsSuccess: result.AccountsSuccess,
		AccountsFailed:  result.AccountsFailed,
		AccountFailures: result.AccountFailures,
		Alerts:          result.Alerts,
	})
}

// handleHealthz 用于进程存活探针。
func (w *webServer) handleHealthz(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeJSON(resp, http.StatusMethodNotAllowed, errorResponse{TSUTC: time.Now().UTC().Format(time.RFC3339Nano), Error: "method not allowed"})
		return
	}
	writeJSON(resp, http.StatusOK, map[string]string{
		"status": "ok",
		"ts_utc": time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// logRequest 输出 HTTP 层结构化日志，便于排障和审计。
func (w *webServer) logRequest(ctx context.Context, level slog.Level, traceID string, startedAt time.Time, err error, message string, attrs ...any) {
	errText := ""
	if err != nil {
		errText = err.Error()
	}

	logAttrs := []any{
		"ts_utc", time.Now().UTC().Format(time.RFC3339Nano),
		"module", "monitor",
		"action", "httpAlertsAPI",
		"provider", "aliyun",
		"region", logRegionForConfig(w.cfg),
		"trace_id", traceID,
		"duration_ms", time.Since(startedAt).Milliseconds(),
		"err", errText,
	}
	logAttrs = append(logAttrs, attrs...)

	switch {
	case level >= slog.LevelError:
		w.logger.ErrorContext(ctx, message, logAttrs...)
	case level >= slog.LevelWarn:
		w.logger.WarnContext(ctx, message, logAttrs...)
	default:
		w.logger.InfoContext(ctx, message, logAttrs...)
	}
}

// writeJSON 统一设置 JSON 响应头并返回数据。
func writeJSON(resp http.ResponseWriter, status int, payload any) {
	resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp.Header().Set("Cache-Control", "no-store")
	resp.WriteHeader(status)
	_ = json.NewEncoder(resp).Encode(payload)
}

// parseLogLevel 将配置文件中的日志等级转换为 slog 等级，未知值默认 warning。
func parseLogLevel(raw string) (slog.Level, string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "debug":
		return slog.LevelDebug, "debug", true
	case "info":
		return slog.LevelInfo, "info", true
	case "warn", "warning":
		return slog.LevelWarn, "warning", true
	case "error":
		return slog.LevelError, "error", true
	default:
		return slog.LevelWarn, "warning", false
	}
}

func newJSONLogger(writer *os.File, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}

// newTraceID 为一次请求生成链路追踪 ID。
func newTraceID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return fmt.Sprintf("trace-%d", time.Now().UTC().UnixNano())
}
