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
