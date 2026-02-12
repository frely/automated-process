# automated-process

用于存放工作中使用的自动化服务与业务模块。

## 项目模块说明

当前项目包含以下云服务相关模块：

- `cloud-service/cloud-dns`：云 DNS 相关能力模块，用于域名解析记录的查询与管理。
- `cloud-service/ecs`：云服务器（ECS）相关能力模块，用于实例信息采集与资源视图整合。
- `cloud-service/total-expenses`：云资源费用汇总模块，用于成本数据归集与统计展示。
- `cloud-service/monitor`：云监控告警模块，用于多账号告警信息聚合与终端展示。

## 说明

- 各模块均为独立业务单元，可按实际场景单独部署与运行。
- 模块之间保持职责边界清晰，便于后续扩展与维护。
