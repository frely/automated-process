# 云服务商费用统计
支持查询阿里云、腾讯云账单分类统计。

## 准备
- pgsql
- Go
- 云服务商 ACCESS_KEY_ID、ACCESS_KEY_SECRET

## 使用

**注意:** 第一次运行程序会生成配置文件，请注意修改 `config.yaml`

示例：
```go
package main

import "github.com/frely/automated-process/cloud-service/total-expenses/cmd/tencentDescribeBillSummaryByProduct"

func main() {
	// 阿里云
	// 账号余额查询
	//fmt.Println(aliyunQueryAccountBalance.Get())

	// 账单总览查询
	//aliyunQueryBillOverview.ToSql()

	// 腾讯云
	// 账号余额查询 取值CashAccountBalance
	//fmt.Println(tencentDescribeAccountBalance.Get())

	//账单总览查询
	tencentDescribeBillSummaryByProduct.ToSql()
}
```