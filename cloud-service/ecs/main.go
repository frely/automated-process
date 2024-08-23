package main

import (
	"github.com/automated-process/ecs/cmd/aliyunDescribeInstances"
	"github.com/automated-process/ecs/cmd/checkEnv"
)

func main() {
	// 阿里云
	checkEnv.CheckAliyun()
	aliyunDescribeInstances.CheckSqlTable()
	aliyunDescribeInstances.Get()
}
