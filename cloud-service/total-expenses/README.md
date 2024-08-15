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
2、执行程序
```shell
git clone https://github.com/frely/automated-process.git

cd automated-process/

go mod tidy

cd cloud-service/total-expenses/

取消注释 go run main.go
```