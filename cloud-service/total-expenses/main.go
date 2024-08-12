package main

import (
	"github.com/automated-process/total-expenses/cmd/aliyunQueryBillOverview"
	"github.com/automated-process/total-expenses/cmd/checkEnv"
)

func main() {
	// export env='["主账号ID"， "ACCESS_KEY_ID", "ACCESS_KEY_SECRET"]'

	// 阿里云
	// 需要设置环境变量：ALIBABA_CLOUD_ACCESS_KEY_ID、ALIBABA_CLOUD_ACCESS_KEY_SECRET

	// 账号余额查询
	//fmt.Println(aliyunQueryAccountBalance.Get())

	// 账单总览查询
	//fmt.Println(aliyunQueryBillOverview.Get())
	checkEnv.Check()
	aliyunQueryBillOverview.ToSql()

	// 腾讯云
	// 需要设置环境变量：SecretId、SecretKey

	// 账号余额查询 取值CashAccountBalance
	//fmt.Println(tencentDescribeAccountBalance.Get())

	//账单总览查询
	//fmt.Println(tencentDescribeBillSummaryByProduct.Get())
}
