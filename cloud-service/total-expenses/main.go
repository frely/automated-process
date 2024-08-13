package main

import (
	"github.com/automated-process/total-expenses/cmd/aliyunQueryBillOverview"
	"github.com/automated-process/total-expenses/cmd/checkEnv"
)

func main() {
	// 阿里云
	// 账号余额查询
	//fmt.Println(aliyunQueryAccountBalance.Get())

	// 账单总览查询
	//fmt.Println(aliyunQueryBillOverview.Get())
	checkEnv.CheckAliyun()
	aliyunQueryBillOverview.ToSql()

	// 腾讯云
	// 需要设置环境变量：SecretId、SecretKey

	// 账号余额查询 取值CashAccountBalance
	//fmt.Println(tencentDescribeAccountBalance.Get())

	//账单总览查询
	//fmt.Println(tencentDescribeBillSummaryByProduct.Get())

}
