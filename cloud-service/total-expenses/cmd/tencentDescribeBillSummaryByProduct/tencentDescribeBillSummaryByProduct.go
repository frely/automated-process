package tencentDescribeBillSummaryByProduct

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/frely/automated-process/cloud-service/total-expenses/cmd/config"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	billing "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/billing/v20180709"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

var (
	billMonth  string
	sqlConnStr string
)

type AutoGenerated struct {
	Response struct {
		RequestID          string `json:"RequestId"`
		ResourceSummarySet []struct {
			ActionTypeName     string `json:"ActionTypeName"`
			BillMonth          string `json:"BillMonth"`
			BusinessCode       string `json:"BusinessCode"`
			BusinessCodeName   string `json:"BusinessCodeName"`
			CashPayAmount      string `json:"CashPayAmount"`
			ConfigDesc         string `json:"ConfigDesc"`
			Discount           string `json:"Discount"`
			ExtendField1       string `json:"ExtendField1"`
			ExtendField2       string `json:"ExtendField2"`
			ExtendField3       string `json:"ExtendField3"`
			ExtendField4       string `json:"ExtendField4"`
			ExtendField5       string `json:"ExtendField5"`
			FeeBeginTime       string `json:"FeeBeginTime"`
			FeeEndTime         string `json:"FeeEndTime"`
			IncentivePayAmount string `json:"IncentivePayAmount"`
			InstanceType       string `json:"InstanceType"`
			OperateUin         string `json:"OperateUin"`
			OrderId            string `json:"OrderId"`
			OriginalCostWithRI string `json:"OriginalCostWithRI"`
			OriginalCostWithSP string `json:"OriginalCostWithSP"`
			OwnerUin           string `json:"OwnerUin"`
			PayModeName        string `json:"PayModeName"`
			PayTime            string `json:"PayTime"`
			PayerUin           string `json:"PayerUin"`
			ProductCode        string `json:"ProductCode"`
			ProductCodeName    string `json:"ProductCodeName"`
			ProjectName        string `json:"ProjectName"`
			RealTotalCost      string `json:"RealTotalCost"`
			ReduceType         string `json:"ReduceType"`
			RegionId           int    `json:"RegionId"`
			RegionName         string `json:"RegionName"`
			ResourceId         string `json:"ResourceId"`
			ResourceName       string `json:"ResourceName"`
			Tags               []any  `json:"Tags"`
			TotalCost          string `json:"TotalCost"`
			TransferPayAmount  string `json:"TransferPayAmount"`
			VoucherPayAmount   string `json:"VoucherPayAmount"`
			ZoneName           string `json:"ZoneName"`
		} `json:"ResourceSummarySet"`
		Total any `json:"Total"`
	} `json:"Response"`
}

func ToSql() {
	config.Init()
	checkSqlTable()
	data := Get()
	if data != "" {
		writeSql([]byte(data))
	}
}

func Get() string {
	// 实例化一个认证对象，入参需要传入腾讯云账户 SecretId 和 SecretKey，此处还需注意密钥对的保密
	// 代码泄露可能会导致 SecretId 和 SecretKey 泄露，并威胁账号下所有资源的安全性。以下代码示例仅供参考，建议采用更安全的方式来使用密钥，请参见：https://cloud.tencent.com/document/product/1278/85305
	// 密钥可前往官网控制台 https://console.cloud.tencent.com/cam/capi 进行获取
	credential := common.NewCredential(
		viper.GetString("SecretId"),
		viper.GetString("SecretKey"),
	)
	// 实例化一个client选项，可选的，没有特殊需求可以跳过
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "billing.tencentcloudapi.com"
	// 实例化要请求产品的client对象,clientProfile是可选的
	client, _ := billing.NewClient(credential, "", cpf)

	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := billing.NewDescribeBillResourceSummaryRequest()

	// 判断是否出账
	cstSh, _ := time.LoadLocation("Asia/Shanghai")
	nowTime, _ := strconv.Atoi(time.Now().In(cstSh).Format("20060102"))
	billMonthTime, _ := strconv.Atoi(time.Now().In(cstSh).Format("200601") + "03")
	if nowTime <= billMonthTime {
		log.Println("账单未出，请明日再试")
		os.Exit(0)
	}

	customMonth := viper.GetString("customMonth")
	if customMonth != "" {
		// 查询指定账单
		billMonth = customMonth
	} else {
		//查询上个月账单
		billMonth = time.Now().AddDate(0, -1, 0).In(cstSh).Format("2006-01")
	}
	log.Println("查询账期：", billMonth)

	request.Offset = common.Uint64Ptr(0)
	request.Limit = common.Uint64Ptr(1000)
	request.Month = common.StringPtr(billMonth)

	// 返回的resp是一个DescribeBillResourceSummaryResponse的实例，与请求对象对应
	response, err := client.DescribeBillResourceSummary(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		log.Println("An API error has returned: %s", err)
		return ""
	}
	if err != nil {
		panic(err)
	}
	// 输出json格式的字符串回包
	return response.ToJsonString()
}

func checkSqlTable() {
	// 如果不存在表，则新建。
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

	rows, err := db.Query(`select count(*) from pg_class where relname = 'tencentDescribeBillSummaryByProduct';`)
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
		if count == "0" {
			log.Println("创建表：tencentDescribeBillSummaryByProduct")
			sqlData := `
				CREATE TABLE "public"."tencentDescribeBillSummaryByProduct" (
				"Month" VARCHAR(7) NOT NULL,
				"ActionTypeName" VARCHAR(200) NOT NULL,
				"BillMonth" VARCHAR(200) NOT NULL,
				"BusinessCode" VARCHAR(200) NOT NULL,
				"BusinessCodeName" VARCHAR(200) NOT NULL,
				"CashPayAmount" NUMERIC NOT NULL,
				"ConfigDesc" VARCHAR(200),
				"Discount" NUMERIC NOT NULL,
				"ExtendField1" VARCHAR(200) NOT NULL,
				"ExtendField2" VARCHAR(200) NOT NULL,
				"ExtendField3" VARCHAR(200) NOT NULL,
				"ExtendField4" VARCHAR(200) NOT NULL,
				"ExtendField5" VARCHAR(200) NOT NULL,
				"FeeBeginTime" VARCHAR(200) NOT NULL,
				"FeeEndTime" VARCHAR(200) NOT NULL,
				"IncentivePayAmount" NUMERIC NOT NULL,
				"InstanceType" VARCHAR(200) NOT NULL,
				"OperateUin" VARCHAR(200) NOT NULL,
				"OrderId" VARCHAR(200) NOT NULL,
				"OriginalCostWithRI" NUMERIC NOT NULL,
				"OriginalCostWithSP" NUMERIC NOT NULL,
				"OwnerUin" VARCHAR(200) NOT NULL,
				"PayModeName" VARCHAR(200) NOT NULL,
				"PayTime" VARCHAR(200) NOT NULL,
				"PayerUin" VARCHAR(200) NOT NULL,
				"ProductCode" VARCHAR(200) NOT NULL,
				"ProductCodeName" VARCHAR(200) NOT NULL,
				"ProjectName" VARCHAR(200) NOT NULL,
				"RealTotalCost" NUMERIC NOT NULL,
				"ReduceType" VARCHAR(200) NOT NULL,
				"RegionId" INTEGER NOT NULL,
				"RegionName" VARCHAR(200) NOT NULL,
				"ResourceId" VARCHAR(200) NOT NULL,
				"ResourceName" VARCHAR(200) NOT NULL,
				"Tags" VARCHAR(200) NOT NULL,
				"TotalCost" NUMERIC NOT NULL,
				"TransferPayAmount" NUMERIC NOT NULL,
				"VoucherPayAmount" NUMERIC NOT NULL,
				"ZoneName" VARCHAR(200) NOT NULL);`
			rows2, err := db.Query(sqlData)
			if err != nil {
				log.Fatalln("创建表失败：", err)
			}
			rows2.Close()
			commitList := [][]string{
				{"Month", "账期"},
				{"BusinessCode", "产品名称代码"},
				{"BusinessCodeName", "产品名称"},
				{"CashPayAmount", "现金支付金额"},
				{"OwnerUin", "账单所属账号ID"},
				{"RealTotalCost", "实际总成本"},
				{"TotalCost", "总成本"},
				{"TransferPayAmount", "转账付款金额"},
				{"VoucherPayAmount", "凭证支付金额"},
			}
			for _, v := range commitList {
				sqlData := fmt.Sprintf(`COMMENT ON COLUMN "tencentDescribeBillSummaryByProduct"."%s" IS '%s'`, v[0], v[1])
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

func writeSql(data []byte) {
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

	for _, i := range p.Response.ResourceSummarySet {
		sqlData := fmt.Sprintf(`INSERT INTO "tencentDescribeBillSummaryByProduct"("Month", "ActionTypeName", "BillMonth", "BusinessCode", "BusinessCodeName", "CashPayAmount", "ConfigDesc", "Discount", "ExtendField1", "ExtendField2", "ExtendField3", "ExtendField4", "ExtendField5", "FeeBeginTime", "FeeEndTime", "IncentivePayAmount", "InstanceType", "OperateUin", "OrderId", "OriginalCostWithRI", "OriginalCostWithSP", "OwnerUin", "PayModeName", "PayTime", "PayerUin", "ProductCode", "ProductCodeName", "ProjectName", "RealTotalCost", "ReduceType", "RegionId", "RegionName", "ResourceId", "ResourceName", "Tags", "TotalCost", "TransferPayAmount", "VoucherPayAmount", "ZoneName") VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s')`,
			billMonth,
			i.ActionTypeName,
			i.BillMonth,
			i.BusinessCode,
			i.BusinessCodeName,
			i.CashPayAmount,
			i.ConfigDesc,
			i.Discount,
			i.ExtendField1,
			i.ExtendField2,
			i.ExtendField3,
			i.ExtendField4,
			i.ExtendField5,
			i.FeeBeginTime,
			i.FeeEndTime,
			i.IncentivePayAmount,
			i.InstanceType,
			i.OperateUin,
			i.OrderId,
			i.OriginalCostWithRI,
			i.OriginalCostWithSP,
			i.OwnerUin,
			i.PayModeName,
			i.PayTime,
			i.PayerUin,
			i.ProductCode,
			i.ProductCodeName,
			i.ProjectName,
			i.RealTotalCost,
			i.ReduceType,
			strconv.Itoa(i.RegionId),
			i.RegionName,
			i.ResourceId,
			i.ResourceName,
			i.Tags,
			i.TotalCost,
			i.TransferPayAmount,
			i.VoucherPayAmount,
			i.ZoneName)

		rows, err := db.Query(sqlData)
		if err != nil {
			log.Fatalln("sql执行失败 func writeSql(): ", err, sqlData)
		}
		rows.Close()
	}
	log.Println("写入数据完成")
}
