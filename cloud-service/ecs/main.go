package main

import (
	"github.com/frely/automated-process/cloud-service/ecs/cmd/config"
)

func main() {
	config.Init()

	// 阿里云
	//aliyunDescribeInstances.CheckSqlTable()
	//aliyunDescribeInstances.ToSql()

	// 腾讯云
	//tencentDescribeInstances.CheckSqlTable()
	//tencentDescribeInstances.ToSql()
}
