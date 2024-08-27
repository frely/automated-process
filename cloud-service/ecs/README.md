# 云服务商ECS信息统计
支持查询阿里云ECS实例的详细信息。

## 准备
- pgsql
- Go
- 云服务商 ACCESS_KEY_ID、ACCESS_KEY_SECRET

## 使用

**注意:** 第一次运行程序会生成配置文件，请注意修改 `config.yaml`

示例：
```go
package main

import (
	"github.com/frely/automated-process/cloud-service/ecs/cmd/aliyunDescribeInstances"
	"github.com/frely/automated-process/cloud-service/ecs/cmd/config"
)

func main() {
	config.Init()

	// 阿里云
	aliyunDescribeInstances.CheckSqlTable()
	aliyunDescribeInstances.ToSql()
}
```