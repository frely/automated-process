package aliyunDescribeAlertingMetricRuleResources

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	cms20190101 "github.com/alibabacloud-go/cms-20190101/v10/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

const (
	moduleName        = "monitor"
	actionName        = "aliyunDescribeAlertingMetricRuleResources"
	providerName      = "aliyun"
	maxRetryAttempts  = 3
	initialRetryDelay = 500 * time.Millisecond
)

type traceIDKey struct{}

// Config 是查询阿里云告警资源所需的业务参数。
type Config struct {
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

// AccountConfig 描述一个阿里云账号对应的查询配置。
type AccountConfig struct {
	Key  string
	Name string
	CMS  Config
}

// Alert 是前端展示的告警核心字段（实例、规则、报警值、报警时间）。
type Alert struct {
	AccountKey        string `json:"account_key"`
	AccountName       string `json:"account_name"`
	InstanceID        string `json:"instance_id"`
	RuleName          string `json:"rule_name"`
	RuleID            string `json:"rule_id"`
	AlertValueMaximum string `json:"alert_value_maximum"`
	LastAlertTimeUTC  string `json:"last_alert_time_utc"`
	AlertTag          string `json:"alert_tag"`
	Level             int32  `json:"level"`

	lastMS int64
}

// Result 是一次单账号 CMS 拉取结果。
type Result struct {
	Region       string  `json:"region"`
	RequestID    string  `json:"request_id"`
	Total        int32   `json:"total"`
	FetchedAtUTC string  `json:"fetched_at_utc"`
	Alerts       []Alert `json:"alerts"`
}

// AccountFailure 描述某个账号查询失败的摘要。
type AccountFailure struct {
	AccountKey  string `json:"account_key"`
	AccountName string `json:"account_name"`
	Error       string `json:"error"`
}

// MultiAccountResult 是多账号聚合后的结果。
type MultiAccountResult struct {
	Total           int32            `json:"total"`
	FetchedAtUTC    string           `json:"fetched_at_utc"`
	AccountsTotal   int              `json:"accounts_total"`
	AccountsSuccess int              `json:"accounts_success"`
	AccountsFailed  int              `json:"accounts_failed"`
	AccountFailures []AccountFailure `json:"account_failures"`
	Alerts          []Alert          `json:"alerts"`
}

// Service 封装 CMS 请求、重试和返回字段提取逻辑。
type Service struct {
	cfg    Config
	client cmsClient
	logger *slog.Logger
}

type alertLister interface {
	List(ctx context.Context) (*Result, error)
}

type multiAccountItem struct {
	key    string
	name   string
	region string
	lister alertLister
}

// MultiAccountService 负责并发拉取多个账号并聚合结果。
type MultiAccountService struct {
	accounts []multiAccountItem
	logger   *slog.Logger
}

type cmsClient interface {
	DescribeAlertingMetricRuleResourcesWithOptions(request *cms20190101.DescribeAlertingMetricRuleResourcesRequest, runtime *util.RuntimeOptions) (_result *cms20190101.DescribeAlertingMetricRuleResourcesResponse, _err error)
}

type cmsResponseError struct {
	Code      int
	Message   string
	RequestID string
}

func (e *cmsResponseError) Error() string {
	return fmt.Sprintf("aliyun cms non-success response: code=%d request_id=%s message=%s", e.Code, e.RequestID, e.Message)
}

var (
	sleepWithContext = func(ctx context.Context, d time.Duration) error {
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
	nowUTC = func() time.Time {
		return time.Now().UTC()
	}
)

// New 初始化查询服务并校验 AK/SK。
func New(cfg Config, logger *slog.Logger) (*Service, error) {
	cfg = normalizeConfig(cfg)
	if strings.TrimSpace(cfg.AccessKeyID) == "" {
		return nil, errors.New("missing ALIBABA_CLOUD_ACCESS_KEY_ID")
	}
	if strings.TrimSpace(cfg.AccessKeySecret) == "" {
		return nil, errors.New("missing ALIBABA_CLOUD_ACCESS_KEY_SECRET")
	}

	client, err := CreateClient(cfg.AccessKeyID, cfg.AccessKeySecret, cfg.RegionID)
	if err != nil {
		return nil, fmt.Errorf("create cms client: %w", err)
	}
	if logger == nil {
		logger = newJSONLogger(os.Stdout)
	}

	return &Service{cfg: cfg, client: client, logger: logger}, nil
}

// NewMultiAccount 初始化多账号聚合服务，每个账号使用独立 CMS Client。
func NewMultiAccount(accounts []AccountConfig, logger *slog.Logger) (*MultiAccountService, error) {
	if len(accounts) == 0 {
		return nil, errors.New("empty cms accounts")
	}
	if logger == nil {
		logger = newJSONLogger(os.Stdout)
	}

	items := make([]multiAccountItem, 0, len(accounts))
	seenKeys := make(map[string]struct{}, len(accounts))
	for idx, account := range accounts {
		key := strings.TrimSpace(account.Key)
		if key == "" {
			key = fmt.Sprintf("account-%d", idx+1)
		}
		if _, duplicated := seenKeys[key]; duplicated {
			return nil, fmt.Errorf("duplicate account key %q", key)
		}
		seenKeys[key] = struct{}{}

		name := strings.TrimSpace(account.Name)
		if name == "" {
			name = key
		}

		svc, err := New(account.CMS, logger)
		if err != nil {
			return nil, fmt.Errorf("init account %q: %w", key, err)
		}

		items = append(items, multiAccountItem{
			key:    key,
			name:   name,
			region: svc.cfg.RegionID,
			lister: svc,
		})
	}

	return &MultiAccountService{accounts: items, logger: logger}, nil
}

// CreateClient uses AK/SK mode to initialize the CMS client.
func CreateClient(accessKeyID, accessKeySecret, regionID string) (_result *cms20190101.Client, _err error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(strings.TrimSpace(accessKeyID)),
		AccessKeySecret: tea.String(strings.TrimSpace(accessKeySecret)),
	}
	config.Endpoint = tea.String("metrics." + strings.TrimSpace(regionID) + ".aliyuncs.com")
	_result, _err = cms20190101.NewClient(config)
	return _result, _err
}

// List 查询触发告警的资源列表，并做 429/5xx 重试。
func (s *Service) List(ctx context.Context) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	traceID := traceIDFromContext(ctx)
	if traceID == "" {
		traceID = generateTraceID()
		ctx = WithTraceID(ctx, traceID)
	}

	request, err := s.buildRequest()
	if err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()

	var lastErr error
	for attempt := 1; attempt <= maxRetryAttempts; attempt++ {
		if err := callCtx.Err(); err != nil {
			return nil, fmt.Errorf("context canceled before request: %w", err)
		}

		attemptStart := nowUTC()
		resp, err := s.client.DescribeAlertingMetricRuleResourcesWithOptions(request, buildRuntimeOptions(s.cfg.Timeout))
		if err == nil {
			if responseErr := validateResponse(resp); responseErr == nil {
				result := toResult(s.cfg.RegionID, resp)
				s.logEvent(callCtx, slog.LevelInfo, traceID, attemptStart, nil, "aliyun cms request succeeded",
					"attempt", attempt,
					"request_id", result.RequestID,
					"total", result.Total,
				)
				return result, nil
			} else {
				err = responseErr
			}
		}

		retriable, statusCode, errorCode := shouldRetry(err)
		requestID := ""
		if resp != nil && resp.Body != nil {
			requestID = tea.StringValue(resp.Body.RequestId)
		}

		level := slog.LevelError
		if retriable && attempt < maxRetryAttempts {
			level = slog.LevelWarn
		}
		s.logEvent(callCtx, level, traceID, attemptStart, err, "aliyun cms request failed",
			"attempt", attempt,
			"request_id", requestID,
			"status_code", statusCode,
			"error_code", errorCode,
		)

		lastErr = err
		if !retriable || attempt >= maxRetryAttempts {
			break
		}

		retryDelay := initialRetryDelay * time.Duration(1<<(attempt-1))
		if sleepErr := sleepWithContext(callCtx, retryDelay); sleepErr != nil {
			return nil, fmt.Errorf("wait before retry: %w", sleepErr)
		}
	}

	if lastErr == nil {
		lastErr = errors.New("empty response from aliyun cms")
	}
	return nil, fmt.Errorf("describe alerting metric rule resources failed: %w", lastErr)
}

// List 并发拉取多个账号的告警，支持部分失败继续返回。
func (m *MultiAccountService) List(ctx context.Context) (*MultiAccountResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(m.accounts) == 0 {
		return nil, errors.New("no account configured")
	}

	traceID := traceIDFromContext(ctx)
	if traceID == "" {
		traceID = generateTraceID()
		ctx = WithTraceID(ctx, traceID)
	}

	listStartedAt := nowUTC()
	result := &MultiAccountResult{
		FetchedAtUTC:    listStartedAt.Format(time.RFC3339Nano),
		AccountsTotal:   len(m.accounts),
		AccountFailures: make([]AccountFailure, 0),
		Alerts:          make([]Alert, 0),
	}

	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		firstErr error
	)

	for _, account := range m.accounts {
		account := account
		wg.Add(1)
		go func() {
			defer wg.Done()
			accountStartedAt := nowUTC()

			accountResult, err := account.lister.List(ctx)
			if err != nil {
				mu.Lock()
				failure := AccountFailure{
					AccountKey:  account.key,
					AccountName: account.name,
					Error:       "fetch failed",
				}
				result.AccountFailures = append(result.AccountFailures, failure)
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()

				m.logAggregate(ctx, slog.LevelWarn, traceID, accountStartedAt, err, "multi account query failed",
					"account_key", account.key,
					"account_name", account.name,
					"region", account.region,
				)
				return
			}

			alerts := make([]Alert, 0, len(accountResult.Alerts))
			for _, alert := range accountResult.Alerts {
				alert.AccountKey = account.key
				alert.AccountName = account.name
				alerts = append(alerts, alert)
			}

			mu.Lock()
			result.Alerts = append(result.Alerts, alerts...)
			result.AccountsSuccess++
			mu.Unlock()

			m.logAggregate(ctx, slog.LevelInfo, traceID, accountStartedAt, nil, "multi account query succeeded",
				"account_key", account.key,
				"account_name", account.name,
				"region", account.region,
				"request_id", accountResult.RequestID,
				"total", accountResult.Total,
			)
		}()
	}

	wg.Wait()
	result.AccountsFailed = len(result.AccountFailures)
	result.Total = int32(len(result.Alerts))
	sortAlertsByLastAlertTime(result.Alerts)

	if result.AccountsSuccess == 0 {
		if firstErr == nil {
			firstErr = errors.New("all account query failed")
		}
		m.logAggregate(ctx, slog.LevelError, traceID, listStartedAt, firstErr, "all accounts failed",
			"accounts_total", result.AccountsTotal,
			"accounts_failed", result.AccountsFailed,
		)
		return nil, fmt.Errorf("all account queries failed: %w", firstErr)
	}

	m.logAggregate(ctx, slog.LevelInfo, traceID, listStartedAt, nil, "multi account query finished",
		"accounts_total", result.AccountsTotal,
		"accounts_success", result.AccountsSuccess,
		"accounts_failed", result.AccountsFailed,
		"total", result.Total,
	)

	return result, nil
}

func sortAlertsByLastAlertTime(alerts []Alert) {
	sort.SliceStable(alerts, func(i, j int) bool {
		left := parseRFC3339NanoToUnixMilli(alerts[i].LastAlertTimeUTC)
		right := parseRFC3339NanoToUnixMilli(alerts[j].LastAlertTimeUTC)
		if left == right {
			return alerts[i].RuleName < alerts[j].RuleName
		}
		return left > right
	})
}

func parseRFC3339NanoToUnixMilli(raw string) int64 {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return 0
	}
	return parsed.UTC().UnixMilli()
}

func (m *MultiAccountService) logAggregate(ctx context.Context, level slog.Level, traceID string, startedAt time.Time, err error, message string, attrs ...any) {
	if m.logger == nil {
		return
	}
	errText := ""
	if err != nil {
		errText = err.Error()
	}
	logAttrs := []any{
		"ts_utc", nowUTC().Format(time.RFC3339Nano),
		"module", moduleName,
		"action", "aliyunDescribeAlertingMetricRuleResourcesMultiAccount",
		"provider", providerName,
		"region", "multi",
		"trace_id", traceID,
		"duration_ms", nowUTC().Sub(startedAt).Milliseconds(),
		"err", errText,
	}
	logAttrs = append(logAttrs, attrs...)

	switch {
	case level >= slog.LevelError:
		m.logger.ErrorContext(ctx, message, logAttrs...)
	case level >= slog.LevelWarn:
		m.logger.WarnContext(ctx, message, logAttrs...)
	default:
		m.logger.InfoContext(ctx, message, logAttrs...)
	}
}

// buildRequest 组装 CMS 查询请求参数。
func (s *Service) buildRequest() (*cms20190101.DescribeAlertingMetricRuleResourcesRequest, error) {
	request := &cms20190101.DescribeAlertingMetricRuleResourcesRequest{
		RegionId: tea.String(s.cfg.RegionID),
		Page:     tea.Int32(s.cfg.Page),
		PageSize: tea.Int32(s.cfg.PageSize),
	}
	if s.cfg.RuleID != "" {
		request.RuleId = tea.String(s.cfg.RuleID)
	}
	if s.cfg.Namespace != "" {
		request.Namespace = tea.String(s.cfg.Namespace)
	}
	if s.cfg.GroupID != "" {
		request.GroupId = tea.String(s.cfg.GroupID)
	}
	if s.cfg.Dimensions != "" {
		request.Dimensions = tea.String(s.cfg.Dimensions)
	}
	if s.cfg.AlertBeforeUTC != "" {
		parsed, err := parseAlertBeforeUTC(s.cfg.AlertBeforeUTC)
		if err != nil {
			return nil, err
		}
		request.AlertBeforeTime = tea.String(strconv.FormatInt(parsed.UnixMilli(), 10))
	}
	return request, nil
}

func normalizeConfig(cfg Config) Config {
	cfg.RegionID = strings.TrimSpace(cfg.RegionID)
	if cfg.RegionID == "" {
		cfg.RegionID = "cn-chengdu"
	}
	cfg.RuleID = strings.TrimSpace(cfg.RuleID)
	cfg.Namespace = strings.TrimSpace(cfg.Namespace)
	cfg.GroupID = strings.TrimSpace(cfg.GroupID)
	cfg.Dimensions = strings.TrimSpace(cfg.Dimensions)
	cfg.AlertBeforeUTC = strings.TrimSpace(cfg.AlertBeforeUTC)
	if cfg.Page <= 0 {
		cfg.Page = 1
	}
	if cfg.PageSize <= 0 {
		cfg.PageSize = 20
	}
	if cfg.Timeout < time.Second {
		cfg.Timeout = 10 * time.Second
	}
	return cfg
}

func parseAlertBeforeUTC(raw string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid ALIBABA_CLOUD_CMS_ALERT_BEFORE_UTC=%q, expected RFC3339 UTC (for example 2026-02-10T08:30:00Z)", raw)
}

func buildRuntimeOptions(timeout time.Duration) *util.RuntimeOptions {
	timeoutMillis := int(timeout / time.Millisecond)
	if timeoutMillis <= 0 {
		timeoutMillis = 1000
	}
	connectMillis := timeoutMillis / 2
	if connectMillis < 1000 {
		connectMillis = 1000
	}
	return (&util.RuntimeOptions{}).
		SetConnectTimeout(connectMillis).
		SetReadTimeout(timeoutMillis)
}

func validateResponse(resp *cms20190101.DescribeAlertingMetricRuleResourcesResponse) error {
	if resp == nil {
		return errors.New("nil response")
	}
	if resp.Body == nil {
		return errors.New("nil response body")
	}
	if tea.BoolValue(resp.Body.Success) {
		return nil
	}
	return &cmsResponseError{
		Code:      int(tea.Int32Value(resp.Body.Code)),
		Message:   tea.StringValue(resp.Body.Message),
		RequestID: tea.StringValue(resp.Body.RequestId),
	}
}

func shouldRetry(err error) (bool, int, string) {
	if err == nil {
		return false, 0, ""
	}

	var sdkErr *tea.SDKError
	if errors.As(err, &sdkErr) {
		statusCode := 0
		if sdkErr.StatusCode != nil {
			statusCode = *sdkErr.StatusCode
		}
		errorCode := tea.StringValue(sdkErr.Code)
		if statusCode == 0 && errorCode != "" {
			if parsedCode, parseErr := strconv.Atoi(errorCode); parseErr == nil {
				statusCode = parsedCode
			}
		}
		return statusCode == 429 || statusCode >= 500, statusCode, errorCode
	}

	var responseErr *cmsResponseError
	if errors.As(err, &responseErr) {
		return responseErr.Code == 429 || responseErr.Code >= 500, responseErr.Code, ""
	}

	return false, 0, ""
}

// toResult 将 SDK 返回转换为前端使用的精简字段。
func toResult(regionID string, resp *cms20190101.DescribeAlertingMetricRuleResourcesResponse) *Result {
	result := &Result{
		Region:       regionID,
		RequestID:    tea.StringValue(resp.Body.RequestId),
		Total:        tea.Int32Value(resp.Body.Total),
		FetchedAtUTC: nowUTC().Format(time.RFC3339Nano),
		Alerts:       []Alert{},
	}
	if resp.Body.Resources == nil || len(resp.Body.Resources.Resource) == 0 {
		return result
	}

	alerts := make([]Alert, 0, len(resp.Body.Resources.Resource))
	for _, resource := range resp.Body.Resources.Resource {
		if resource == nil {
			continue
		}

		resourceText := tea.StringValue(resource.Resource)
		dimensionsText := tea.StringValue(resource.Dimensions)
		metricValuesText := tea.StringValue(resource.MetricValues)
		lastAlertUTC, lastAlertMS := unixMilliStringToUTC(tea.StringValue(resource.LastAlertTime))

		alerts = append(alerts, Alert{
			InstanceID:        extractInstanceID(resourceText, dimensionsText),
			RuleName:          tea.StringValue(resource.RuleName),
			RuleID:            tea.StringValue(resource.RuleId),
			AlertValueMaximum: extractMaximumValue(metricValuesText),
			LastAlertTimeUTC:  lastAlertUTC,
			AlertTag:          extractAlertTag(resource),
			Level:             tea.Int32Value(resource.Level),
			lastMS:            lastAlertMS,
		})
	}

	sort.SliceStable(alerts, func(i, j int) bool {
		return alerts[i].lastMS > alerts[j].lastMS
	})
	for i := range alerts {
		alerts[i].lastMS = 0
	}

	result.Alerts = alerts
	return result
}

// extractInstanceID 优先从 Resource 字段提取实例 ID，缺失时回退解析 Dimensions。
func extractInstanceID(resourceText, dimensionsText string) string {
	for _, part := range strings.Split(resourceText, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(kv[0]), "instanceId") {
			return strings.TrimSpace(kv[1])
		}
	}

	dimensions := decodeJSONObject(dimensionsText)
	for key, value := range dimensions {
		if strings.EqualFold(key, "instanceId") {
			return normalizeJSONScalar(value)
		}
	}

	return ""
}

// extractMaximumValue 从 MetricValues JSON 中提取 Maximum 作为报警值。
func extractMaximumValue(metricValuesText string) string {
	metricValues := decodeJSONObject(metricValuesText)
	for key, value := range metricValues {
		if strings.EqualFold(key, "Maximum") {
			return normalizeJSONScalar(value)
		}
	}
	return ""
}

// extractAlertTag 根据当前触发 Level 匹配 Escalation 中的业务标签（如 CRITICAL/INFO）。
func extractAlertTag(resource *cms20190101.DescribeAlertingMetricRuleResourcesResponseBodyResourcesResource) string {
	if resource == nil {
		return ""
	}

	level := tea.Int32Value(resource.Level)
	if resource.Escalation != nil && len(resource.Escalation.Resource) > 0 {
		fallbackTag := ""
		for _, escalator := range resource.Escalation.Resource {
			if escalator == nil {
				continue
			}
			tag := strings.ToUpper(strings.TrimSpace(tea.StringValue(escalator.Tag)))
			if tag == "" {
				continue
			}
			if fallbackTag == "" {
				fallbackTag = tag
			}
			if tea.Int32Value(escalator.Level) == level {
				return tag
			}
		}
		if fallbackTag != "" {
			return fallbackTag
		}
	}

	if level == 0 {
		return "OK"
	}
	return fmt.Sprintf("LEVEL_%d", level)
}

// decodeJSONObject 解析 SDK 返回中的 JSON 字符串字段。
func decodeJSONObject(raw string) map[string]any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return map[string]any{}
	}

	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	var data map[string]any
	if err := decoder.Decode(&data); err != nil {
		return map[string]any{}
	}
	return data
}

func normalizeJSONScalar(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", value)
	}
}

// unixMilliStringToUTC 将毫秒时间戳转换为 UTC RFC3339Nano。
func unixMilliStringToUTC(raw string) (string, int64) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", 0
	}
	ms, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return "", 0
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339Nano), ms
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(traceID) == "" {
		return ctx
	}
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

func traceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	traceID, _ := ctx.Value(traceIDKey{}).(string)
	return strings.TrimSpace(traceID)
}

func generateTraceID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return fmt.Sprintf("trace-%d", nowUTC().UnixNano())
}

// logEvent 输出云调用链路日志，满足生产审计与排障要求。
func (s *Service) logEvent(ctx context.Context, level slog.Level, traceID string, startedAt time.Time, err error, message string, attrs ...any) {
	errText := ""
	if err != nil {
		errText = err.Error()
	}
	logAttrs := []any{
		"ts_utc", nowUTC().Format(time.RFC3339Nano),
		"module", moduleName,
		"action", actionName,
		"provider", providerName,
		"region", s.cfg.RegionID,
		"trace_id", traceID,
		"duration_ms", nowUTC().Sub(startedAt).Milliseconds(),
		"err", errText,
	}
	logAttrs = append(logAttrs, attrs...)

	switch {
	case level >= slog.LevelError:
		s.logger.ErrorContext(ctx, message, logAttrs...)
	case level >= slog.LevelWarn:
		s.logger.WarnContext(ctx, message, logAttrs...)
	default:
		s.logger.InfoContext(ctx, message, logAttrs...)
	}
}

func newJSONLogger(writer io.Writer) *slog.Logger {
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slog.LevelWarn})
	return slog.New(handler)
}
