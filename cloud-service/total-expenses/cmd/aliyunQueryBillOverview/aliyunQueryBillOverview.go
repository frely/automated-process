package aliyunQueryBillOverview

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	bssopenapi20171214 "github.com/alibabacloud-go/bssopenapi-20171214/v5/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

var (
	billingcycle string
	resStr       string
	sqlConnStr   string
)

type AutoGenerated struct {
	Item []struct {
		AdjustAmount          float64 `json:"AdjustAmount"`
		BillAccountID         string  `json:"BillAccountID"`
		BillAccountName       string  `json:"BillAccountName"`
		BizType               string  `json:"BizType"`
		CashAmount            float64 `json:"CashAmount"`
		CommodityCode         string  `json:"CommodityCode"`
		Currency              string  `json:"Currency"`
		DeductedByCashCoupons float64 `json:"DeductedByCashCoupons"`
		DeductedByCoupons     float64 `json:"DeductedByCoupons"`
		DeductedByPrepaidCard float64 `json:"DeductedByPrepaidCard"`
		InvoiceDiscount       float64 `json:"InvoiceDiscount"`
		Item                  string  `json:"Item"`
		OutstandingAmount     float64 `json:"OutstandingAmount"`
		OwnerID               string  `json:"OwnerID"`
		PaymentAmount         float64 `json:"PaymentAmount"`
		PipCode               string  `json:"PipCode"`
		PretaxAmount          float64 `json:"PretaxAmount"`
		PretaxGrossAmount     float64 `json:"PretaxGrossAmount"`
		ProductCode           string  `json:"ProductCode"`
		ProductDetail         string  `json:"ProductDetail"`
		ProductName           string  `json:"ProductName"`
		ProductType           string  `json:"ProductType"`
		RoundDownDiscount     string  `json:"RoundDownDiscount"`
		SubscriptionType      string  `json:"SubscriptionType"`
	} `json:"Item"`
}

func ToSql() {
	data := []byte(Get())
	checkSqlTable()
	writeSql(data)
}

// Description:
//
// 使用AK&SK初始化账号Client
//
// @return Client
//
// @throws Exception
func CreateClient() (_result *bssopenapi20171214.Client, _err error) {
	// 工程代码泄露可能会导致 AccessKey 泄露，并威胁账号下所有资源的安全性。以下代码示例仅供参考。
	// 建议使用更安全的 STS 方式，更多鉴权访问方式请参见：https://help.aliyun.com/document_detail/378661.html。
	config := &openapi.Config{
		// 必填，请确保代码运行环境设置了环境变量 ALIBABA_CLOUD_ACCESS_KEY_ID。
		AccessKeyId: tea.String(viper.GetString("ALIBABA_CLOUD_ACCESS_KEY_ID")),
		// 必填，请确保代码运行环境设置了环境变量 ALIBABA_CLOUD_ACCESS_KEY_SECRET。
		AccessKeySecret: tea.String(viper.GetString("ALIBABA_CLOUD_ACCESS_KEY_SECRET")),
	}
	// Endpoint 请参考 https://api.aliyun.com/product/BssOpenApi
	config.Endpoint = tea.String("business.aliyuncs.com")
	_result = &bssopenapi20171214.Client{}
	_result, _err = bssopenapi20171214.NewClient(config)
	return _result, _err
}

func _main(args []*string) (_err error) {
	// 判断是否出账
	cstSh, _ := time.LoadLocation("Asia/Shanghai")
	nowTime, _ := strconv.Atoi(time.Now().In(cstSh).Format("20060102"))

	billingcycleTime, _ := strconv.Atoi(time.Now().In(cstSh).Format("200601") + "02")
	if nowTime <= billingcycleTime {
		log.Println("账单未出，请明日再试")
		os.Exit(0)
	}

	customMonth := viper.GetString("customMonth")
	if customMonth != "" {
		// 查询指定账单
		billingcycle = customMonth
	} else {
		// 查询上个月账单
		billingcycle = time.Now().AddDate(0, -1, 0).In(cstSh).Format("2006-01")
	}
	log.Println("查询账期：", billingcycle)

	client, _err := CreateClient()
	if _err != nil {
		return _err
	}

	queryBillOverviewRequest := &bssopenapi20171214.QueryBillOverviewRequest{
		BillingCycle: tea.String(billingcycle),
	}
	runtime := &util.RuntimeOptions{}
	tryErr := func() (_e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				_e = r
			}
		}()
		// 复制代码运行请自行打印 API 的返回值
		res, _err := client.QueryBillOverviewWithOptions(queryBillOverviewRequest, runtime)
		if _err != nil {
			return _err
		}
		resStr = res.Body.Data.Items.String()

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
		log.Println("func Get()", err)
	}
	return resStr
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

	rows, err := db.Query(`select count(*) from pg_class where relname = 'aliyunQueryBillOverview';`)
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
			log.Println("创建表：aliyunQueryBillOverview")
			sqlData := `
				CREATE TABLE "public"."aliyunQueryBillOverview" (
				"BillingCycle" VARCHAR(7) NOT NULL,
				"DeductedByCoupons" NUMERIC NOT NULL,
				"RoundDownDiscount" NUMERIC NOT NULL,
				"ProductName" VARCHAR(200) NOT NULL,
				"ProductDetail" VARCHAR(200) NOT NULL,
				"ProductCode" VARCHAR(200) NOT NULL,
				"BillAccountID" VARCHAR(200) NOT NULL,
				"ProductType" VARCHAR(200),
				"DeductedByCashCoupons" NUMERIC NOT NULL,
				"OutstandingAmount" NUMERIC NOT NULL,
				"BizType" VARCHAR(200),
				"PaymentAmount" NUMERIC NOT NULL,
				"PipCode" VARCHAR(200),
				"DeductedByPrepaidCard" NUMERIC NOT NULL,
				"InvoiceDiscount" NUMERIC NOT NULL,
				"Item" VARCHAR(200),
				"SubscriptionType" VARCHAR(200),
				"PretaxGrossAmount" NUMERIC NOT NULL,
				"PretaxAmount" NUMERIC NOT NULL,
				"OwnerID" VARCHAR(200),
				"Currency" VARCHAR(200),
				"CommodityCode" VARCHAR(200),
				"BillAccountName" VARCHAR(200),
				"AdjustAmount" NUMERIC NOT NULL,
				"CashAmount" NUMERIC NOT NULL
				);`
			rows2, err := db.Query(sqlData)
			if err != nil {
				log.Fatalln("创建表失败：", err)
			}
			rows2.Close()

			commitList := [][]string{
				{"BillingCycle", "账期"},
				{"DeductedByCoupons", "优惠劵抵扣"},
				{"RoundDownDiscount", "抹零优惠"},
				{"ProductName", "产品名称"},
				{"ProductDetail", "产品明细"},
				{"ProductCode", "产品代码"},
				{"BillAccountID", "账单所属账号ID"},
				{"ProductType", "产品类型"},
				{"DeductedByCashCoupons", "代金券抵扣"},
				{"OutstandingAmount", "未结清金额或信用结算金额（普通用户的欠费，或者信用客户信用额度消耗）"},
				{"BizType", "业务类型"},
				{"PaymentAmount", "现金支付（含信用额度退款抵扣）"},
				{"PipCode", "产品Code，与费用中心账单产品Code一致"},
				{"DeductedByPrepaidCard", "储蓄卡抵扣"},
				{"InvoiceDiscount", "优惠金额"},
				{"PretaxGrossAmount", "原始金额"},
				{"PretaxAmount", "应付金额"},
				{"OwnerID", "账单OwnerID"},
				{"CommodityCode", "商品Code，与费用中心产品明细Code一致"},
				{"BillAccountName", "账单所属账号名称"},
				{"AdjustAmount", "信用额度退款抵扣"},
				{"CashAmount", "现金支付（不包含信用额度退款抵扣）"},
			}
			for _, v := range commitList {
				sqlData := fmt.Sprintf(`COMMENT ON COLUMN "aliyunQueryBillOverview"."%s" IS '%s'`, v[0], v[1])
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

	for _, i := range p.Item {
		sqlData := fmt.Sprintf(`INSERT INTO "aliyunQueryBillOverview"("BillingCycle", "DeductedByCoupons", "RoundDownDiscount", "ProductName", "ProductDetail", "ProductCode", "BillAccountID", "ProductType", "DeductedByCashCoupons", "OutstandingAmount", "BizType", "PaymentAmount", "PipCode", "DeductedByPrepaidCard", "InvoiceDiscount", "Item", "SubscriptionType", "PretaxGrossAmount", "PretaxAmount", "OwnerID", "Currency", "CommodityCode", "BillAccountName", "AdjustAmount", "CashAmount") VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s')`,
			billingcycle,
			strconv.FormatFloat(i.DeductedByCoupons, 'f', 2, 64),
			i.RoundDownDiscount,
			i.ProductName,
			i.ProductDetail,
			i.ProductCode,
			i.BillAccountID,
			i.ProductType,
			strconv.FormatFloat(i.DeductedByCashCoupons, 'f', 2, 64),
			strconv.FormatFloat(i.OutstandingAmount, 'f', 2, 64),
			i.BizType,
			strconv.FormatFloat(i.PaymentAmount, 'f', 2, 64),
			i.PipCode,
			strconv.FormatFloat(i.DeductedByPrepaidCard, 'f', 2, 64),
			strconv.FormatFloat(i.InvoiceDiscount, 'f', 2, 64),
			i.Item,
			i.SubscriptionType,
			strconv.FormatFloat(i.PretaxGrossAmount, 'f', 2, 64),
			strconv.FormatFloat(i.PretaxAmount, 'f', 2, 64),
			i.OwnerID,
			i.Currency,
			i.CommodityCode,
			i.BillAccountName,
			strconv.FormatFloat(i.AdjustAmount, 'f', 2, 64),
			strconv.FormatFloat(i.CashAmount, 'f', 2, 64))

		rows, err := db.Query(sqlData)
		if err != nil {
			log.Fatalln("sql执行失败 func writeSql(): ", err, sqlData)
		}
		rows.Close()
	}
	log.Println("写入数据完成")
}
