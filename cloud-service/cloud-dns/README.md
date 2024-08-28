# 云服务商云解析DNS统计
支持腾讯云云解析DNS统计，同时检查SSL证书到期时间。

## 准备
- pgsql
- Go
- 云服务商 ACCESS_KEY_ID、ACCESS_KEY_SECRET

## 使用

**注意:** 第一次运行程序会生成配置文件，请注意修改 `config.yaml`
```
示例：
```go
package main

import (
	"github.com/frely/automated-process/cloud-service/cloud-dns/cmd/config"
	"github.com/frely/automated-process/cloud-service/cloud-dns/cmd/tencentDescribeRecordList"
)

func main() {
	config.Init()

	// 腾讯云
	tencentDescribeRecordList.CheckSqlTable()
	tencentDescribeRecordList.Tosql()
}
```