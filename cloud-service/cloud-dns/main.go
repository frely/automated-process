package main

import (
	"github.com/frely/automated-process/cloud-service/cloud-dns/cmd/config"
	"github.com/frely/automated-process/cloud-service/cloud-dns/cmd/sslCheck"
	"github.com/frely/automated-process/cloud-service/cloud-dns/cmd/tencentDescribeRecordList"
)

func main() {
	config.Init()

	// 腾讯云
	tencentDescribeRecordList.CheckSqlTable()
	tencentDescribeRecordList.Tosql()
	sslCheck.Check()
}
