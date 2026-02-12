# 云监控 Web 服务（Kindle 显示）

`cloud-service/monitor` 是一个独立模块（不放在 `ecs` 内），用于拉取阿里云云监控（CMS）告警，并以 Kindle 友好的紧凑页面展示。

## 模块能力

- 后端：调用阿里云 CMS `DescribeAlertingMetricRuleResources`
- 多账号：支持配置多个阿里云账号并发拉取、统一聚合输出
- 前端：原生 `HTML + CSS + JS`，无分页、紧凑模式
- 时间：后端统一输出 UTC，前端按设备时区自动显示
- 刷新：支持手动刷新 + 自动轮询刷新
- 观测：结构化 JSON 日志，带 `trace_id`、`duration_ms` 等关键字段

## 快速启动

```bash
go -C cloud-service/monitor run .
```

首次运行会自动生成 `cloud-service/monitor/config.yaml` 模板并退出；补齐配置后再次启动。

## 配置说明

`config.yaml` 支持同名环境变量覆盖：

- `WEB_SERVER_ADDR`：监听地址，默认 `:8080`
- `WEB_REFRESH_SECONDS`：前端自动刷新秒数，默认 `60`
- `WEB_LOG_LEVEL`：日志等级，支持 `debug/info/warning/error`，默认 `warning`

### 单账号模式（兼容旧配置）

- `ALIBABA_CLOUD_ACCESS_KEY_ID`：阿里云 AK（必填）
- `ALIBABA_CLOUD_ACCESS_KEY_SECRET`：阿里云 SK（必填）
- `ALIBABA_CLOUD_CMS_REGION`：地域，默认 `cn-chengdu`
- `ALIBABA_CLOUD_CMS_RULE_ID`：告警规则 ID（可选）
- `ALIBABA_CLOUD_CMS_NAMESPACE`：命名空间（可选）
- `ALIBABA_CLOUD_CMS_GROUP_ID`：应用分组 ID（可选）
- `ALIBABA_CLOUD_CMS_DIMENSIONS`：资源维度 JSON（可选）
- `ALIBABA_CLOUD_CMS_ALERT_BEFORE_UTC`：UTC 时间过滤（RFC3339），例如 `2026-02-10T08:30:00Z`
- `ALIBABA_CLOUD_CMS_PAGE`：页码，默认 `1`
- `ALIBABA_CLOUD_CMS_PAGE_SIZE`：分页大小，默认 `20`
- `ALIBABA_CLOUD_CMS_TIMEOUT_SECONDS`：CMS 请求超时秒数，默认 `10`

### 多账号模式（推荐）

使用 `ALIBABA_CLOUD_CMS_ACCOUNTS` 数组配置多个账号。每个账号建议通过 `*_env` 指向环境变量注入密钥：

```yaml
ALIBABA_CLOUD_CMS_ACCOUNTS:
  - key: "prod-a"
    name: "生产A"
    region: "cn-chengdu"
    access_key_id_env: "MONITOR_ACCOUNT_A_AK"
    access_key_secret_env: "MONITOR_ACCOUNT_A_SK"
    rule_id: ""
    namespace: ""
    group_id: ""
    dimensions: ""
    alert_before_utc: ""
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
```

字段说明：

- `key`：账号唯一标识（用于前端标签与失败摘要）
- `name`：账号展示名（可选，默认等于 `key`）
- `region`：账号地域（可选，默认沿用全局默认）
- `access_key_id_env` / `access_key_secret_env`：环境变量名（推荐）
- `access_key_id` / `access_key_secret`：直接写入凭据（不推荐，仅测试）
- 其余筛选参数与单账号含义一致

## 认证方式（AK/SK）

后端按你要求使用 AK/SK 初始化客户端：

```go
config := &openapi.Config{
    AccessKeyId:     tea.String("...")
    AccessKeySecret: tea.String("...")
}
config.Endpoint = tea.String("metrics.cn-chengdu.aliyuncs.com")
```

实际代码中 AK/SK 来自配置文件或环境变量，不写死在业务代码中。

## HTTP 接口

- 页面：`GET /`
- 告警 API：`GET /api/alerts`
- 健康检查：`GET /healthz`

## 字段对照（CMS -> 本服务 API）

本服务重点抽取你关心的 4 个字段，并补充账号与等级标签：

- 报警资源（实例）：`Resource.instanceId`，若缺失则回退 `Dimensions.instanceId` -> `instance_id`
- 报警规则名称：`RuleName` -> `rule_name`
- 报警值：`MetricValues.Maximum` -> `alert_value_maximum`
- 报警时间：`LastAlertTime`（毫秒时间戳）-> UTC RFC3339Nano -> `last_alert_time_utc`
- 告警标签：按 `Escalation.Resource[].Level` 匹配当前 `Level` 的 `Tag` -> `alert_tag`
- 账号标识：配置中的 `key/name` -> `account_key/account_name`

`/api/alerts` 返回示例（节选）：

```json
{
  "ts_utc": "2026-02-12T09:10:11.123456Z",
  "refresh_seconds": 60,
  "total": 6,
  "accounts_total": 2,
  "accounts_success": 1,
  "accounts_failed": 1,
  "account_failures": [
    {
      "account_key": "prod-b",
      "account_name": "生产B",
      "error": "fetch failed"
    }
  ],
  "alerts": [
    {
      "account_key": "prod-a",
      "account_name": "生产A",
      "instance_id": "i-2ze35o869v7ap9g0b16u",
      "rule_name": "内存使用率",
      "rule_id": "SystemDefault_acs_ecs_dashboard_vm.MemoryUtilization",
      "alert_value_maximum": "90.18",
      "last_alert_time_utc": "2026-12-12T03:25:59Z",
      "alert_tag": "CRITICAL",
      "level": 2
    }
  ]
}
```

## Kindle 页面展示规则

- 使用紧凑模式，不启用分页
- 列表每项仅展示：账号标签、规则名、实例 ID、报警值（Maximum）、报警时间（本地）
- 页面顶部右侧显示：`总数`、`账号成功/总数`、`更新` 和 `刷新`按钮
- 页面不展示“地域”和“时区”文本，避免占用可视区域

## 稳定性与可观测性

- 外部调用显式超时控制（`timeout_seconds` / `ALIBABA_CLOUD_CMS_TIMEOUT_SECONDS`）
- 仅对 `429/5xx` 做 3 次指数退避重试
- 多账号查询支持“部分失败继续”，接口返回失败账号摘要
- 每次请求记录结构化日志：`ts_utc`、`module`、`action`、`provider`、`region`、`trace_id`、`duration_ms`、`err`
- 业务时间链路统一 UTC；前端仅做展示层时区转换

## 开发与验证

```bash
go -C cloud-service/monitor test ./...
```

提交前建议：

```bash
gofmt -w cloud-service/monitor/main.go cloud-service/monitor/cmd/config/config.go cloud-service/monitor/cmd/aliyunDescribeAlertingMetricRuleResources/aliyunDescribeAlertingMetricRuleResources.go cloud-service/monitor/cmd/config/config_test.go cloud-service/monitor/cmd/aliyunDescribeAlertingMetricRuleResources/aliyunDescribeAlertingMetricRuleResources_test.go
go -C cloud-service/monitor test ./...
```

## 安全提示

- `config.yaml` 建议仅保留模板和非敏感默认值
- 生产环境优先使用环境变量或密钥管理系统注入 AK/SK
- 日志与示例请保持脱敏
