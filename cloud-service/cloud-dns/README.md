# 云服务商云解析DNS统计
支持腾讯云云解析DNS统计。

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
export POSTGRES_DB="DomainList"

# 腾讯云
export SecretId="tFKFH4"
export SecretKey="tFKFH4zCGy7"
```
2、执行程序
```shell
git clone https://github.com/frely/automated-process.git

cd automated-process/cloud-service/cloud-dns/

go mod tidy

取消注释 go run main.go
```