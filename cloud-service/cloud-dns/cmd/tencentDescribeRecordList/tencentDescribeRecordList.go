package tencentDescribeRecordList

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/frely/automated-process/cloud-service/cloud-dns/cmd/tencentDescribeDomainList"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

var (
	sqlConnStr string
)

type AutoGenerated struct {
	Response struct {
		RecordCountInfo struct {
			SubdomainCount int `json:"SubdomainCount"`
			ListCount      int `json:"ListCount"`
			TotalCount     int `json:"TotalCount"`
		} `json:"RecordCountInfo"`
		RecordList []struct {
			RecordId      int    `json:"RecordId"`
			Value         string `json:"Value"`
			Status        string `json:"Status"`
			UpdatedOn     string `json:"UpdatedOn"`
			Name          string `json:"Name"`
			Line          string `json:"Line"`
			LineId        string `json:"LineId"`
			Type          string `json:"Type"`
			MonitorStatus string `json:"MonitorStatus"`
			Remark        string `json:"Remark"`
			TTL           int    `json:"TTL"`
			MX            int    `json:"MX"`
		} `json:"RecordList"`
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

// 写入数据库
func Tosql() {
	for _, v := range tencentDescribeDomainList.Get() {
		if v[1] == "ENABLE" {
			data := []byte(getRecordList(v[0], "A"))
			writeSql(data, v[0])
			data = []byte(getRecordList(v[0], "CNAME"))
			writeSql(data, v[0])
		}
		// 限制速率，避免请求失败
		time.Sleep(3 * time.Second)
	}
}

func getRecordList(domain, recordType string) string {
	// 实例化一个认证对象，入参需要传入腾讯云账户 SecretId 和 SecretKey，此处还需注意密钥对的保密
	// 代码泄露可能会导致 SecretId 和 SecretKey 泄露，并威胁账号下所有资源的安全性。以下代码示例仅供参考，建议采用更安全的方式来使用密钥，请参见：https://cloud.tencent.com/document/product/1278/85305
	// 密钥可前往官网控制台 https://console.cloud.tencent.com/cam/capi 进行获取
	credential := common.NewCredential(
		viper.GetString("SecretId"),
		viper.GetString("SecretKey"),
	)
	// 实例化一个client选项，可选的，没有特殊需求可以跳过
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "dnspod.tencentcloudapi.com"
	// 实例化要请求产品的client对象,clientProfile是可选的
	client, _ := dnspod.NewClient(credential, "", cpf)

	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := dnspod.NewDescribeRecordListRequest()

	request.Domain = common.StringPtr(domain)
	request.RecordType = common.StringPtr(recordType)

	// 返回的resp是一个DescribeRecordListResponse的实例，与请求对象对应
	response, err := client.DescribeRecordList(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		log.Printf("An API error has returned: %s", err)
	}
	if err != nil {
		log.Println(err)
	}
	// 输出json格式的字符串回包
	//fmt.Printf("%s", response.ToJsonString())
	return response.ToJsonString()
}

// 表不存在则新建，表存在则清空表
func CheckSqlTable() {
	sqlConnStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		viper.GetString("POSTGRES_USER"),
		viper.GetString("POSTGRES_PASSWORD"),
		viper.GetString("POSTGRES_HOST"),
		viper.GetString("POSTGRES_PORT"),
		viper.GetString("POSTGRES_DB"))
	db, err := sql.Open("postgres", sqlConnStr)
	if err != nil {
		log.Fatalln("连接数据库失败", err)
	}
	defer db.Close()

	rows, err := db.Query(`select count(*) from pg_class where relname = 'tencentDomainList';`)
	defer rows.Close()
	if err != nil {
		log.Fatalln("查询表失败：", err)
	}
	for rows.Next() {
		var count string
		err := rows.Scan(&count)
		if err != nil {
			log.Fatalln("获取表失败", err)
		}
		if count == "1" {
			log.Println("清空表")
			sqlData := `TRUNCATE TABLE "tencentDomainList"`
			rows2, err := db.Query(sqlData)
			if err != nil {
				log.Fatalln("清空表失败：", err)
			}
			rows2.Close()
		} else {
			log.Println("创建表：tencentDomainList")
			sqlData := `
				CREATE TABLE "public"."tencentDomainList" (
				"Line" VARCHAR(200) NOT NULL,
				"LineId" VARCHAR(200) NOT NULL,
				"MX" INTEGER NOT NULL,
				"MonitorStatus" VARCHAR(200) NOT NULL,
				"Name" VARCHAR(200) NOT NULL,
				"Record" VARCHAR(200) NOT NULL,
				"Domain" VARCHAR(200) NOT NULL,
				"RecordId" INTEGER NOT NULL,
				"Remark" VARCHAR(200) NOT NULL,
				"Status" VARCHAR(200) NOT NULL,
				"TTL" INTEGER NOT NULL,
				"Type" VARCHAR(200) NOT NULL,
				"UpdatedOn" VARCHAR(200) NOT NULL,
				"Value" VARCHAR(200) NOT NULL,
				"NotBefore" VARCHAR(200),
				"NotAfter" VARCHAR(200),
				"Subject" VARCHAR(200)
				);`
			rows2, err := db.Query(sqlData)
			if err != nil {
				log.Fatalln("创建表失败：", err)
			}
			rows2.Close()
			commitList := [][]string{
				{"NotBefore", "颁发日期"},
				{"NotAfter", "到期日期"},
				{"Subject", "DNS名称"},
			}
			for _, v := range commitList {
				sqlData := fmt.Sprintf(`COMMENT ON COLUMN "tencentDomainList"."%s" IS '%s'`, v[0], v[1])
				rows3, err := db.Query(sqlData)
				if err != nil {
					log.Fatalln("添加注释失败: ", sqlData, err)
				}
				rows3.Close()
			}
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatalln("查询sql失败", err)
	}
}

func writeSql(data []byte, domain string) {
	var p AutoGenerated
	err := json.Unmarshal(data, &p)
	if err != nil {
		log.Fatalln("解析返回值失败", err)
	}

	sqlConnStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		viper.GetString("POSTGRES_USER"),
		viper.GetString("POSTGRES_PASSWORD"),
		viper.GetString("POSTGRES_HOST"),
		viper.GetString("POSTGRES_PORT"),
		viper.GetString("POSTGRES_DB"))
	db, err := sql.Open("postgres", sqlConnStr)
	if err != nil {
		log.Fatalln("连接数据库失败 func writeSql()", err)
	}
	defer db.Close()
	for _, i := range p.Response.RecordList {
		if i.Name == "@" {
			sqlData := fmt.Sprintf(`INSERT INTO "tencentDomainList"("Line","LineId","MX","MonitorStatus","Name","Record","Domain","RecordId","Remark","Status","TTL","Type","UpdatedOn","Value") VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s')`,
				i.Line,
				i.LineId,
				strconv.Itoa(i.MX),
				i.MonitorStatus,
				i.Name,
				domain,
				domain,
				strconv.Itoa(i.RecordId),
				i.Remark,
				i.Status,
				strconv.Itoa(i.TTL),
				i.Type,
				i.UpdatedOn,
				i.Value)
			rows, err := db.Query(sqlData)
			if err != nil {
				log.Fatalln("sql执行失败 func writeSql(): ", err, sqlData)
			}
			rows.Close()
		} else {
			sqlData := fmt.Sprintf(`INSERT INTO "tencentDomainList"("Line","LineId","MX","MonitorStatus","Name","Record","Domain","RecordId","Remark","Status","TTL","Type","UpdatedOn","Value") VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s')`,
				i.Line,
				i.LineId,
				strconv.Itoa(i.MX),
				i.MonitorStatus,
				i.Name,
				i.Name+"."+domain,
				domain,
				strconv.Itoa(i.RecordId),
				i.Remark,
				i.Status,
				strconv.Itoa(i.TTL),
				i.Type,
				i.UpdatedOn,
				i.Value)
			rows, err := db.Query(sqlData)
			if err != nil {
				log.Fatalln("sql执行失败 func writeSql(): ", err, sqlData)
			}
			rows.Close()
		}
	}
	log.Println("写入数据完成")
}
