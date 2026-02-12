package aliyunDescribeAlertingMetricRuleResources

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	cms20190101 "github.com/alibabacloud-go/cms-20190101/v10/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

type mockCMSClient struct {
	responses []*cms20190101.DescribeAlertingMetricRuleResourcesResponse
	errors    []error
	calls     int
}

func (m *mockCMSClient) DescribeAlertingMetricRuleResourcesWithOptions(_ *cms20190101.DescribeAlertingMetricRuleResourcesRequest, _ *util.RuntimeOptions) (*cms20190101.DescribeAlertingMetricRuleResourcesResponse, error) {
	idx := m.calls
	m.calls++

	var resp *cms20190101.DescribeAlertingMetricRuleResourcesResponse
	if idx < len(m.responses) {
		resp = m.responses[idx]
	}

	var err error
	if idx < len(m.errors) {
		err = m.errors[idx]
	}

	return resp, err
}

func TestListRetriesOn429ThenSuccess(t *testing.T) {
	mockClient := &mockCMSClient{
		responses: []*cms20190101.DescribeAlertingMetricRuleResourcesResponse{
			nil,
			{
				Body: &cms20190101.DescribeAlertingMetricRuleResourcesResponseBody{
					Success:   tea.Bool(true),
					RequestId: tea.String("req-200"),
					Total:     tea.Int32(1),
					Resources: &cms20190101.DescribeAlertingMetricRuleResourcesResponseBodyResources{
						Resource: []*cms20190101.DescribeAlertingMetricRuleResourcesResponseBodyResourcesResource{
							{
								RuleName:      tea.String("内存使用率"),
								RuleId:        tea.String("rule-memory"),
								Resource:      tea.String("userId=1361709,instanceId=i-2ze35o869v7ap9g0b16u"),
								MetricValues:  tea.String("{\"Maximum\":90.18}"),
								LastAlertTime: tea.String("1765539559000"),
								Level:         tea.Int32(2),
								Escalation: &cms20190101.DescribeAlertingMetricRuleResourcesResponseBodyResourcesResourceEscalation{
									Resource: []*cms20190101.DescribeAlertingMetricRuleResourcesResponseBodyResourcesResourceEscalationResource{
										{Level: tea.Int32(4), Tag: tea.String("INFO")},
										{Level: tea.Int32(2), Tag: tea.String("CRITICAL")},
									},
								},
							},
						},
					},
				},
			},
		},
		errors: []error{
			&tea.SDKError{StatusCode: tea.Int(429), Code: tea.String("TooManyRequests"), Message: tea.String("rate limit")},
			nil,
		},
	}

	restore := overrideTestHooks()
	defer restore()

	sleepCalls := 0
	sleepWithContext = func(_ context.Context, _ time.Duration) error {
		sleepCalls++
		return nil
	}

	svc := &Service{
		cfg:    Config{RegionID: "cn-chengdu", Timeout: 2 * time.Second, Page: 1, PageSize: 20},
		client: mockClient,
		logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	result, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if mockClient.calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", mockClient.calls)
	}
	if sleepCalls != 1 {
		t.Fatalf("expected 1 retry sleep, got %d", sleepCalls)
	}
	if result.Total != 1 {
		t.Fatalf("expected total=1, got %d", result.Total)
	}
	if len(result.Alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(result.Alerts))
	}
	alert := result.Alerts[0]
	if alert.InstanceID != "i-2ze35o869v7ap9g0b16u" {
		t.Fatalf("unexpected instance id: %s", alert.InstanceID)
	}
	if alert.RuleName != "内存使用率" {
		t.Fatalf("unexpected rule name: %s", alert.RuleName)
	}
	if alert.AlertValueMaximum != "90.18" {
		t.Fatalf("unexpected maximum value: %s", alert.AlertValueMaximum)
	}
	if alert.LastAlertTimeUTC == "" {
		t.Fatal("expected last alert time in utc")
	}
	if alert.AlertTag != "CRITICAL" {
		t.Fatalf("unexpected alert tag: %s", alert.AlertTag)
	}
}

func TestListNoRetryOn400(t *testing.T) {
	mockClient := &mockCMSClient{
		errors: []error{
			&tea.SDKError{StatusCode: tea.Int(400), Code: tea.String("InvalidParameter"), Message: tea.String("bad request")},
		},
	}

	restore := overrideTestHooks()
	defer restore()

	sleepWithContext = func(_ context.Context, _ time.Duration) error {
		return errors.New("sleep should not be called")
	}

	svc := &Service{
		cfg:    Config{RegionID: "cn-chengdu", Timeout: 2 * time.Second, Page: 1, PageSize: 20},
		client: mockClient,
		logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mockClient.calls != 1 {
		t.Fatalf("expected 1 attempt, got %d", mockClient.calls)
	}
}

func TestParseAlertBeforeUTCConvertsUTC(t *testing.T) {
	parsed, err := parseAlertBeforeUTC("2026-02-11T10:30:00+08:00")
	if err != nil {
		t.Fatalf("parseAlertBeforeUTC returned error: %v", err)
	}
	if got := parsed.Format(time.RFC3339); got != "2026-02-11T02:30:00Z" {
		t.Fatalf("unexpected utc conversion: %s", got)
	}
}

func TestExtractInstanceIDFromDimensions(t *testing.T) {
	instanceID := extractInstanceID("", "{\"instanceId\":\"i-abc\"}")
	if instanceID != "i-abc" {
		t.Fatalf("unexpected instance id from dimensions: %s", instanceID)
	}
}

func overrideTestHooks() func() {
	oldSleep := sleepWithContext
	oldNow := nowUTC

	nowUTC = func() time.Time {
		return time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC)
	}

	return func() {
		sleepWithContext = oldSleep
		nowUTC = oldNow
	}
}

type mockAlertLister struct {
	result *Result
	err    error
}

func (m *mockAlertLister) List(_ context.Context) (*Result, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestMultiAccountListPartialFailure(t *testing.T) {
	svc := &MultiAccountService{
		accounts: []multiAccountItem{
			{
				key:    "prod-a",
				name:   "生产A",
				region: "cn-chengdu",
				lister: &mockAlertLister{result: &Result{
					RequestID: "req-1",
					Total:     1,
					Alerts: []Alert{{
						RuleName:         "CPU使用率",
						LastAlertTimeUTC: "2026-02-12T01:02:03Z",
					}},
				}},
			},
			{
				key:    "prod-b",
				name:   "生产B",
				region: "cn-hangzhou",
				lister: &mockAlertLister{err: errors.New("request failed")},
			},
		},
		logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	result, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if result.AccountsTotal != 2 {
		t.Fatalf("unexpected accounts total: %d", result.AccountsTotal)
	}
	if result.AccountsSuccess != 1 {
		t.Fatalf("unexpected accounts success: %d", result.AccountsSuccess)
	}
	if result.AccountsFailed != 1 {
		t.Fatalf("unexpected accounts failed: %d", result.AccountsFailed)
	}
	if len(result.AccountFailures) != 1 {
		t.Fatalf("expected 1 account failure, got %d", len(result.AccountFailures))
	}
	if result.AccountFailures[0].Error != "fetch failed" {
		t.Fatalf("unexpected failure message: %s", result.AccountFailures[0].Error)
	}
	if len(result.Alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(result.Alerts))
	}
	if result.Alerts[0].AccountKey != "prod-a" {
		t.Fatalf("unexpected account key: %s", result.Alerts[0].AccountKey)
	}
	if result.Alerts[0].AccountName != "生产A" {
		t.Fatalf("unexpected account name: %s", result.Alerts[0].AccountName)
	}
}

func TestMultiAccountListAllFailed(t *testing.T) {
	svc := &MultiAccountService{
		accounts: []multiAccountItem{
			{key: "prod-a", name: "生产A", region: "cn-chengdu", lister: &mockAlertLister{err: errors.New("a failed")}},
			{key: "prod-b", name: "生产B", region: "cn-shanghai", lister: &mockAlertLister{err: errors.New("b failed")}},
		},
		logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error when all accounts failed")
	}
}

func TestMultiAccountListSortsAcrossAccounts(t *testing.T) {
	svc := &MultiAccountService{
		accounts: []multiAccountItem{
			{
				key:    "prod-a",
				name:   "生产A",
				region: "cn-chengdu",
				lister: &mockAlertLister{result: &Result{
					Alerts: []Alert{{RuleName: "Rule-A", LastAlertTimeUTC: "2026-02-12T01:00:00Z"}},
				}},
			},
			{
				key:    "prod-b",
				name:   "生产B",
				region: "cn-shanghai",
				lister: &mockAlertLister{result: &Result{
					Alerts: []Alert{{RuleName: "Rule-B", LastAlertTimeUTC: "2026-02-12T02:00:00Z"}},
				}},
			},
		},
		logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	result, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(result.Alerts) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(result.Alerts))
	}
	if result.Alerts[0].RuleName != "Rule-B" {
		t.Fatalf("unexpected first rule order: %s", result.Alerts[0].RuleName)
	}
	if result.Alerts[1].RuleName != "Rule-A" {
		t.Fatalf("unexpected second rule order: %s", result.Alerts[1].RuleName)
	}
}
