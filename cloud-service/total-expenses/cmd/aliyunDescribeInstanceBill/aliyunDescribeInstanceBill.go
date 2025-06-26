package aliyunDescribeInstanceBill

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	bssopenapi20171214 "github.com/alibabacloud-go/bssopenapi-20171214/v6/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

var (
	resStr     *string
	sqlConnStr string
)

func ToSql() {
	log.Println("开始获取阿里云实例账单数据...")
	dataStr := Get()
	data := []byte(dataStr)

	if len(data) == 0 {
		log.Println("未获取到数据")
		return
	}

	checkSqlTable()
	writeSql(data)
}

// Description:
//
// 使用凭据初始化账号Client
//
// @return Client
//
// @throws Exception
func CreateClient() (_result *bssopenapi20171214.Client, _err error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(viper.GetString("ALIBABA_CLOUD_ACCESS_KEY_ID")),
		AccessKeySecret: tea.String(viper.GetString("ALIBABA_CLOUD_ACCESS_KEY_SECRET")),
	}
	// Endpoint 请参考 https://api.aliyun.com/product/BssOpenApi
	config.Endpoint = tea.String("business.aliyuncs.com")
	_result = &bssopenapi20171214.Client{}
	_result, _err = bssopenapi20171214.NewClient(config)
	return _result, _err
}

func _main(args []*string) (_err error) {
	client, _err := CreateClient()
	if _err != nil {
		return _err
	}

	customDay := viper.GetString("aliyunDescribeInstanceBillCustomDay")
	var currentMonth, billingDate string

	if customDay == "" {
		cstSh, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			log.Printf("加载时区失败: %v", err)
			return err
		}
		currentMonth = time.Now().In(cstSh).Format("2006-01")
		billingDate = time.Now().AddDate(0, 0, -1).In(cstSh).Format("2006-01-02") // 查询昨天的账单
	} else {
		if len(customDay) < 7 {
			log.Printf("自定义日期格式错误，需要至少7位字符: %s", customDay)
			return fmt.Errorf("自定义日期格式错误")
		}
		currentMonth = customDay[:7]
		billingDate = customDay
	}

	describeInstanceBillRequest := &bssopenapi20171214.DescribeInstanceBillRequest{
		BillingCycle: tea.String(currentMonth),
		BillingDate:  tea.String(billingDate),
		Granularity:  tea.String("DAILY"),
		MaxResults:   tea.Int32(300),
	}
	runtime := &util.RuntimeOptions{}
	tryErr := func() (_e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				_e = r
			}
		}()
		// 复制代码运行请自行打印 API 的返回值
		res, _err := client.DescribeInstanceBillWithOptions(describeInstanceBillRequest, runtime)
		if _err != nil {
			return _err
		}

		if *res.Body.Data.TotalCount > 300 {
			return fmt.Errorf("获取账单失败: 单次请求账单数量超过300")
		}

		// 将响应数据序列化为JSON字符串
		jsonData, err := json.Marshal(res.Body.Data)
		if err != nil {
			return err
		}
		jsonStr := string(jsonData)
		resStr = &jsonStr
		return nil
	}()

	if tryErr != nil {
		var error = &tea.SDKError{}
		if _t, ok := tryErr.(*tea.SDKError); ok {
			error = _t
		} else {
			error.Message = tea.String(tryErr.Error())
		}
		// 此处仅做打印展示，请谨慎对待异常处理，在工程项目中切勿直接忽略异常。
		// 错误 message
		fmt.Println(tea.StringValue(error.Message))
		// 诊断地址
		var data interface{}
		d := json.NewDecoder(strings.NewReader(tea.StringValue(error.Data)))
		d.Decode(&data)
		if m, ok := data.(map[string]interface{}); ok {
			recommend, _ := m["Recommend"]
			fmt.Println(recommend)
		}
		_, _err = util.AssertAsString(error.Message)
		if _err != nil {
			return _err
		}
	}
	return _err
}

func Get() string {
	err := _main(tea.StringSlice(os.Args[1:]))
	if err != nil {
		log.Printf("获取数据时发生错误: %v", err)
		return ""
	}

	if resStr == nil {
		log.Println("未获取到数据")
		return ""
	}

	return *resStr
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

	rows, err := db.Query(`select count(*) from pg_class where relname = 'aliyunDescribeInstanceBill';`)
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
			log.Println("创建表：aliyunDescribeInstanceBill")
			sqlData := `
				CREATE TABLE "public"."aliyunDescribeInstanceBill" (
				"AfterDiscountAmount" NUMERIC NOT NULL,
				"InstanceSpec" VARCHAR(200) NOT NULL,
				"ProductName" VARCHAR(200) NOT NULL,
				"InstanceID" VARCHAR(200) NOT NULL,
				"BillAccountID" VARCHAR(200) NOT NULL,
				"BillingDate" VARCHAR(200) NOT NULL,
				"ListPriceUnit" VARCHAR(200) NOT NULL,
				"ListPrice" VARCHAR(200) NOT NULL,
				"InvoiceDiscount" NUMERIC NOT NULL,
				"Item" VARCHAR(200) NOT NULL,
				"SubscriptionType" VARCHAR(200) NOT NULL,
				"PretaxGrossAmount" NUMERIC NOT NULL,
				"InstanceConfig" VARCHAR(1000) NOT NULL,
				"Currency" VARCHAR(200) NOT NULL,
				"CommodityCode" VARCHAR(200) NOT NULL,
				"ItemName" VARCHAR(200) NOT NULL,
				"CostUnit" VARCHAR(200) NOT NULL,
				"ResourceGroup" VARCHAR(200) NOT NULL,
				"BillingType" VARCHAR(200) NOT NULL,
				"DeductedByCoupons" NUMERIC NOT NULL,
				"Usage" VARCHAR(200) NOT NULL,
				"ProductDetail" VARCHAR(200) NOT NULL,
				"ProductCode" VARCHAR(200) NOT NULL,
				"Zone" VARCHAR(200) NOT NULL,
				"ProductType" VARCHAR(200) NOT NULL,
				"BizType" VARCHAR(200) NOT NULL,
				"BillingItem" VARCHAR(200) NOT NULL,
				"NickName" VARCHAR(200) NOT NULL,
				"PipCode" VARCHAR(200) NOT NULL,
				"IntranetIP" VARCHAR(200) NOT NULL,
				"ServicePeriodUnit" VARCHAR(200) NOT NULL,
				"ServicePeriod" VARCHAR(200) NOT NULL,
				"DeductedByResourcePackage" VARCHAR(200) NOT NULL,
				"UsageUnit" VARCHAR(200) NOT NULL,
				"InternetIP" VARCHAR(200) NOT NULL,
				"PretaxAmount" NUMERIC NOT NULL,
				"OwnerID" VARCHAR(200) NOT NULL,
				"BillAccountName" VARCHAR(200) NOT NULL,
				"Region" VARCHAR(200) NOT NULL,
				"Tag" VARCHAR(1000) NOT NULL
				);`
			rows2, err := db.Query(sqlData)
			if err != nil {
				log.Fatalln("创建表失败：", err)
			}
			rows2.Close()

			commitList := [][]string{
				{"AfterDiscountAmount", "优惠后金额(包含券抵扣的应付金额数据。计算规则为:优惠后金额=官网目录价-优惠金额)"},
				{"InstanceSpec", "实例规格"},
				{"ProductName", "产品名称"},
				{"InstanceID", "实例ID"},
				{"BillAccountID", "账单所属账号ID"},
				{"BillingDate", "账单日期"},
				{"ListPriceUnit", "单价单位"},
				{"ListPrice", "单价"},
				{"InvoiceDiscount", "优惠金额"},
				{"Item", "账单类型"},
				{"SubscriptionType", "订阅类型"},
				{"PretaxGrossAmount", "原始金额"},
				{"InstanceConfig", "实例详细配置"},
				{"Currency", "币种"},
				{"CommodityCode", "商品Code，与费用中心产品明细Code一致"},
				{"ItemName", "项目名称"},
				{"CostUnit", "财务单元"},
				{"ResourceGroup", "资源组"},
				{"BillingType", "计费方式"},
				{"DeductedByCoupons", "优惠券优惠金额"},
				{"Usage", "用量"},
				{"ProductDetail", "产品明细"},
				{"ProductCode", "产品代码"},
				{"Zone", "可用区"},
				{"ProductType", "产品类型"},
				{"BizType", "业务类型"},
				{"BillingItem", "计费项"},
				{"NickName", "实例昵称"},
				{"PipCode", "产品code，与费用中心账单产品code一致"},
				{"IntranetIP", "内网IP"},
				{"ServicePeriodUnit", "服务时长单位"},
				{"ServicePeriod", "服务时长"},
				{"DeductedByResourcePackage", "资源包抵扣"},
				{"UsageUnit", "用量单位"},
				{"InternetIP", "公网IP"},
				{"PretaxAmount", "应付金额"},
				{"OwnerID", "资源owner账号AccountID(多账号代付场景)"},
				{"BillAccountName", "账单所属账号名称"},
				{"Region", "地域"},
				{"Tag", "资源标签"},
			}
			for _, v := range commitList {
				sqlData := fmt.Sprintf(`COMMENT ON COLUMN "aliyunDescribeInstanceBill"."%s" IS '%s'`, v[0], v[1])
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
	var p struct {
		Items []struct {
			AfterDiscountAmount       float64 `json:"AfterDiscountAmount"`
			InstanceSpec              string  `json:"InstanceSpec"`
			ProductName               string  `json:"ProductName"`
			InstanceID                string  `json:"InstanceID"`
			BillAccountID             string  `json:"BillAccountID"`
			BillingDate               string  `json:"BillingDate"`
			ListPriceUnit             string  `json:"ListPriceUnit"`
			ListPrice                 string  `json:"ListPrice"`
			InvoiceDiscount           float64 `json:"InvoiceDiscount"`
			Item                      string  `json:"Item"`
			SubscriptionType          string  `json:"SubscriptionType"`
			PretaxGrossAmount         float64 `json:"PretaxGrossAmount"`
			InstanceConfig            string  `json:"InstanceConfig"`
			Currency                  string  `json:"Currency"`
			CommodityCode             string  `json:"CommodityCode"`
			ItemName                  string  `json:"ItemName"`
			CostUnit                  string  `json:"CostUnit"`
			ResourceGroup             string  `json:"ResourceGroup"`
			BillingType               string  `json:"BillingType"`
			DeductedByCoupons         float64 `json:"DeductedByCoupons"`
			Usage                     string  `json:"Usage"`
			ProductDetail             string  `json:"ProductDetail"`
			ProductCode               string  `json:"ProductCode"`
			Zone                      string  `json:"Zone"`
			ProductType               string  `json:"ProductType"`
			BizType                   string  `json:"BizType"`
			BillingItem               string  `json:"BillingItem"`
			NickName                  string  `json:"NickName"`
			PipCode                   string  `json:"PipCode"`
			IntranetIP                string  `json:"IntranetIP"`
			ServicePeriodUnit         string  `json:"ServicePeriodUnit"`
			ServicePeriod             string  `json:"ServicePeriod"`
			DeductedByResourcePackage string  `json:"DeductedByResourcePackage"`
			UsageUnit                 string  `json:"UsageUnit"`
			InternetIP                string  `json:"InternetIP"`
			PretaxAmount              float64 `json:"PretaxAmount"`
			OwnerID                   string  `json:"OwnerID"`
			BillAccountName           string  `json:"BillAccountName"`
			Region                    string  `json:"Region"`
			Tag                       string  `json:"Tag"`
		} `json:"Items"`
	}

	err := json.Unmarshal(data, &p)
	if err != nil {
		log.Printf("解析返回值失败: %v", err)
		return
	}

	if len(p.Items) == 0 {
		log.Println("没有数据需要写入")
		return
	}

	db, err := sql.Open("postgres", sqlConnStr)
	if err != nil {
		log.Fatalln("连接数据库失败", err)
	}
	defer db.Close()

	// 使用参数化查询防止SQL注入
	sqlQuery := `INSERT INTO "aliyunDescribeInstanceBill"(
		"AfterDiscountAmount", "InstanceSpec", "ProductName", "InstanceID", "BillAccountID", 
		"BillingDate", "ListPriceUnit", "ListPrice", "InvoiceDiscount", "Item", 
		"SubscriptionType", "PretaxGrossAmount", "InstanceConfig", "Currency", "CommodityCode", 
		"ItemName", "CostUnit", "ResourceGroup", "BillingType", "DeductedByCoupons", 
		"Usage", "ProductDetail", "ProductCode", "Zone", "ProductType", 
		"BizType", "BillingItem", "NickName", "PipCode", "IntranetIP", 
		"ServicePeriodUnit", "ServicePeriod", "DeductedByResourcePackage", "UsageUnit", "InternetIP", 
		"PretaxAmount", "OwnerID", "BillAccountName", "Region", "Tag"
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
		$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
		$31, $32, $33, $34, $35, $36, $37, $38, $39, $40
	)`

	successCount := 0
	for _, i := range p.Items {
		_, err := db.Exec(sqlQuery,
			i.AfterDiscountAmount,
			i.InstanceSpec,
			i.ProductName,
			i.InstanceID,
			i.BillAccountID,
			i.BillingDate,
			i.ListPriceUnit,
			i.ListPrice,
			i.InvoiceDiscount,
			i.Item,
			i.SubscriptionType,
			i.PretaxGrossAmount,
			i.InstanceConfig,
			i.Currency,
			i.CommodityCode,
			i.ItemName,
			i.CostUnit,
			i.ResourceGroup,
			i.BillingType,
			i.DeductedByCoupons,
			i.Usage,
			i.ProductDetail,
			i.ProductCode,
			i.Zone,
			i.ProductType,
			i.BizType,
			i.BillingItem,
			i.NickName,
			i.PipCode,
			i.IntranetIP,
			i.ServicePeriodUnit,
			i.ServicePeriod,
			i.DeductedByResourcePackage,
			i.UsageUnit,
			i.InternetIP,
			i.PretaxAmount,
			i.OwnerID,
			i.BillAccountName,
			i.Region,
			i.Tag)

		if err != nil {
			log.Printf("插入数据失败: %v, 实例ID: %s", err, i.InstanceID)
		} else {
			successCount++
		}
	}
	log.Printf("写入数据完成，成功插入 %d/%d 条记录", successCount, len(p.Items))
}
