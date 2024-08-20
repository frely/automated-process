package tencentDescribeRecordList

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/automated-process/cloud-dns/cmd/tencentDescribeDomainList"
	_ "github.com/lib/pq"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

var (
	sqlConnStr string = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"))
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

func Tosql() {
	checkSqlTable()
	for _, v := range tencentDescribeDomainList.Get() {
		if v[1] == "ENABLE" {
			data := []byte(getRecordList(v[0]))
			writeSql(data, v[0])
		}
		// 限制速率，避免请求失败
		time.Sleep(3 * time.Second)
	}
}

func getRecordList(domain string) string {
	// 实例化一个认证对象，入参需要传入腾讯云账户 SecretId 和 SecretKey，此处还需注意密钥对的保密
	// 代码泄露可能会导致 SecretId 和 SecretKey 泄露，并威胁账号下所有资源的安全性。以下代码示例仅供参考，建议采用更安全的方式来使用密钥，请参见：https://cloud.tencent.com/document/product/1278/85305
	// 密钥可前往官网控制台 https://console.cloud.tencent.com/cam/capi 进行获取
	credential := common.NewCredential(
		os.Getenv("SecretId"),
		os.Getenv("SecretKey"),
	)
	// 实例化一个client选项，可选的，没有特殊需求可以跳过
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "dnspod.tencentcloudapi.com"
	// 实例化要请求产品的client对象,clientProfile是可选的
	client, _ := dnspod.NewClient(credential, "", cpf)

	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := dnspod.NewDescribeRecordListRequest()

	request.Domain = common.StringPtr(domain)
	request.RecordType = common.StringPtr("A")

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

func checkSqlTable() {
	// 如果不存在表，则新建。
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
			rows2.Close()
			if err != nil {
				log.Fatalln("清空表失败：", err)
			}
		} else {
			log.Println("创建表：tencentDomainList")
			sqlData := `
				CREATE TABLE "public"."tencentDomainList" (
				"Line" VARCHAR(200) NOT NULL,
				"LineId" VARCHAR(200) NOT NULL,
				"MX" INTEGER NOT NULL,
				"MonitorStatus" VARCHAR(200) NOT NULL,
				"Name" VARCHAR(200) NOT NULL,
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
			rows2.Close()
			if err != nil {
				log.Fatalln("创建表失败：", err)
			}
			commitList := [][]string{
				{"NotBefore", "颁发日期"},
				{"NotAfter", "到期日期"},
				{"Subject", "DNS名称"},
			}
			for _, v := range commitList {
				sqlData := fmt.Sprintf(`COMMENT ON COLUMN "tencentDomainList"."%s" IS '%s'`, v[0], v[1])
				rows3, err := db.Query(sqlData)
				rows3.Close()
				if err != nil {
					log.Fatalln("添加注释失败: ", sqlData, err)
				}
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

	db, err := sql.Open("postgres", sqlConnStr)
	if err != nil {
		log.Fatalln("连接数据库失败 func writeSql()", err)
	}
	defer db.Close()
	for _, i := range p.Response.RecordList {
		if i.Name == "@" {
			sqlData := fmt.Sprintf(`INSERT INTO "tencentDomainList"("Line","LineId","MX","MonitorStatus","Name","Domain","RecordId","Remark","Status","TTL","Type","UpdatedOn","Value") VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s')`,
				i.Line,
				i.LineId,
				strconv.Itoa(i.MX),
				i.MonitorStatus,
				i.Name,
				domain,
				strconv.Itoa(i.RecordId),
				i.Remark,
				i.Status,
				strconv.Itoa(i.TTL),
				i.Type,
				i.UpdatedOn,
				i.Value)
			rows, err := db.Query(sqlData)
			rows.Close()
			if err != nil {
				log.Fatalln("sql执行失败 func writeSql(): ", err, sqlData)
			}
		} else {
			sqlData := fmt.Sprintf(`INSERT INTO "tencentDomainList"("Line","LineId","MX","MonitorStatus","Name","Domain","RecordId","Remark","Status","TTL","Type","UpdatedOn","Value") VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s')`,
				i.Line,
				i.LineId,
				strconv.Itoa(i.MX),
				i.MonitorStatus,
				i.Name,
				i.Name+"."+domain,
				strconv.Itoa(i.RecordId),
				i.Remark,
				i.Status,
				strconv.Itoa(i.TTL),
				i.Type,
				i.UpdatedOn,
				i.Value)

			rows, err := db.Query(sqlData)
			rows.Close()
			if err != nil {
				log.Fatalln("sql执行失败 func writeSql(): ", err, sqlData)
			}
		}
	}
	log.Println("写入数据完成")
}

func GetCheckList() []string {
	db, err := sql.Open("postgres", sqlConnStr)
	if err != nil {
		log.Fatalln("连接数据库失败", err)
	}
	defer db.Close()

	rows, err := db.Query(`select "Domain" from public."tencentDomainList" where "Status" = 'ENABLE'`)
	defer rows.Close()
	if err != nil {
		log.Fatalln("查询表失败：", err)
	}
	domainList := []string{}
	for rows.Next() {
		var domain string
		err := rows.Scan(&domain)
		if err != nil {
			log.Fatalln("获取表失败", err)
		}
		domainList = append(domainList, domain)
	}
	if err := rows.Err(); err != nil {
		log.Fatalln("查询sql失败", err)
	}

	return domainList
}

func SslCheck(domainList []string) {
	var cstSh, _ = time.LoadLocation("Asia/Shanghai")
	nowTime, _ := time.Parse("2006-01-02 15:04:05", time.Now().In(cstSh).Format("2006-01-02 15:04:05"))

	db, err := sql.Open("postgres", sqlConnStr)
	if err != nil {
		log.Fatalln("连接数据库失败", err)
	}
	defer db.Close()

	dialer := net.Dialer{Timeout: time.Second * 3}

	for _, domain := range domainList {
		log.Println("======== 开始检查：", domain)
		conn, err := tls.DialWithDialer(&dialer, "tcp", domain+":443", nil)
		if err != nil {
			log.Println("连接错误或未配置证书：", domain)
		} else {
			cert := conn.ConnectionState().PeerCertificates[0]

			// 时间信息
			//fmt.Printf("NotBefore: %v\n", cert.NotBefore.In(cstSh))
			//fmt.Printf("NotAfter: %v\n", cert.NotAfter.In(cstSh))
			//fmt.Printf("Issuer: %v\n", cert.Issuer)
			//fmt.Printf("Subject: %v\n", cert.Subject)

			endTime, _ := time.Parse("2006-01-02 15:04:05", cert.NotAfter.In(cstSh).Format("2006-01-02 15:04:05"))
			d := endTime.Sub(nowTime).Hours() / 24
			dStr := strings.Split(strconv.FormatFloat(d, 'g', -1, 64), ".")
			dInt, _ := strconv.Atoi(dStr[0])

			if dInt < 0 {
				log.Printf("%s：证书已到期", domain)
			} else {
				log.Printf("%s：到期时间还有%d天", domain, dInt)
			}

			sqlData := fmt.Sprintf(`UPDATE public."tencentDomainList" SET "NotBefore" = '%s', "NotAfter" = '%s', "Subject" = '%s' WHERE "Domain"='%s'`, cert.NotBefore.In(cstSh), cert.NotAfter.In(cstSh), cert.Subject, domain)
			rows, err := db.Query(sqlData)
			defer rows.Close()
			if err != nil {
				log.Fatalln("写入表失败：", err, sqlData)
			}
		}
	}
}
