# 云服务商费用统计
支持查询阿里云、腾讯云账单分类统计。

## 准备
- pgsql
- Go
- 云服务商 ACCESS_KEY_ID、ACCESS_KEY_SECRET

## Linux 下使用
1、配置环境变量，例如：
```shell
export POSTGRES_HOST="10.43.237.229"
export POSTGRES_PORT="5432"
export POSTGRES_USER="postgres"
export POSTGRES_PASSWORD="postgres"
export POSTGRES_DB="aliyunBillId13600000000"

# 阿里云
export ALIBABA_CLOUD_ACCESS_KEY_ID="tFKFH4"
export ALIBABA_CLOUD_ACCESS_KEY_SECRET="tFKFH4zCGy7"

# 腾讯云
export SecretId="tFKFH4"
export SecretKey="tFKFH4zCGy7"
```
2、示例：
```go
package main

import (
	"github.com/frely/automated-process/cloud-service/total-expenses/cmd/aliyunQueryBillOverview"
	"github.com/frely/automated-process/cloud-service/total-expenses/cmd/checkEnv"
)

func main() {
	// 阿里云
	// 账号余额查询
	//fmt.Println(aliyunQueryAccountBalance.Get())

	// 账单总览查询
	checkEnv.CheckAliyun()
	aliyunQueryBillOverview.ToSql()

	// 腾讯云
	// 账号余额查询 取值CashAccountBalance
	//fmt.Println(tencentDescribeAccountBalance.Get())

	//账单总览查询
	//checkEnv.CheckTencent()
	//tencentDescribeBillSummaryByProduct.ToSql()
}
```