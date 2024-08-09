package main

import (
	"fmt"
	"os"
)

func main() {
	if os.Getenv("ACCESS_KEY_ID") == "" {
		fmt.Println("ACCESS_KEY_ID Not Find")
		os.Exit(0)
	}
	if os.Getenv("ACCESS_KEY_SECRET") == "" {
		fmt.Println("ACCESS_KEY_SECRET Not Find")
		os.Exit(0)
	}
	// 阿里云
	// 账号余额查询
	//fmt.Println(aliyunQueryAccountBalance.Get())

	// 账单总览查询
	//fmt.Println(aliyunQueryBillOverview.Get())

	// 腾讯云
	// 账号余额查询 取值CashAccountBalance
	//fmt.Println(tencentDescribeAccountBalance.Get())

	//账单总览查询
	//fmt.Println(tencentDescribeBillSummaryByProduct.Get())

}
