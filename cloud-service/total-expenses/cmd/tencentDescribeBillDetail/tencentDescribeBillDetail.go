package tencentDescribeBillDetail

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	billing "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/billing/v20180709"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

var (
	resStr     *string
	sqlConnStr string
)

func ToSql() {
	checkSqlTable()
	data := Get()
	if data != "" {
		writeSql([]byte(data))
	}
}

func Get() string {
	// 实例化一个认证对象，入参需要传入腾讯云账户 SecretId 和 SecretKey，此处还需注意密钥对的保密
	// 代码泄露可能会导致 SecretId 和 SecretKey 泄露，并威胁账号下所有资源的安全性
	// 以下代码示例仅供参考，建议采用更安全的方式来使用密钥
	// 请参见：https://cloud.tencent.com/document/product/1278/85305
	// 密钥可前往官网控制台 https://console.cloud.tencent.com/cam/capi 进行获取
	credential := common.NewCredential(
		viper.GetString("SecretId"),
		viper.GetString("SecretKey"),
	)
	// 使用临时密钥示例
	// credential := common.NewTokenCredential("SecretId", "SecretKey", "Token")
	// 实例化一个client选项，可选的，没有特殊需求可以跳过
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "billing.tencentcloudapi.com"
	// 实例化要请求产品的client对象,clientProfile是可选的
	client, _ := billing.NewClient(credential, "", cpf)

	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := billing.NewDescribeBillDetailRequest()

	request.Offset = common.Uint64Ptr(0)
	request.Limit = common.Uint64Ptr(300)

	customDay := viper.GetString("billCustomDay")
	var billingDate string

	if customDay == "" {
		cstSh, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			log.Printf("加载时区失败: %v", err)
		}
		billingDate = time.Now().AddDate(0, 0, -1).In(cstSh).Format("2006-01-02") // 查询昨天的账单
	} else {
		if len(customDay) < 7 {
			log.Printf("自定义日期格式错误，需要至少7位字符: %s", customDay)
		}
		billingDate = customDay
	}

	log.Printf("billingDate: %s", billingDate)

	request.BeginTime = common.StringPtr(billingDate + " 00:00:00")
	request.EndTime = common.StringPtr(billingDate + " 23:59:59")
	// 返回的resp是一个DescribeBillDetailResponse的实例，与请求对象对应
	response, err := client.DescribeBillDetail(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
		return ""
	}
	if err != nil {
		panic(err)
	}

	if len(response.Response.DetailSet) > 300 {
		log.Fatalln("获取账单失败: 单次请求账单数量超过300")
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

	rows, err := db.Query(`select count(*) from pg_class where relname = 'tencentDescribeBillDetail';`)
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
			log.Println("创建表：tencentDescribeBillDetail")
			sqlData := `
				CREATE TABLE "public"."tencentDescribeBillDetail" (
				"ActionType" VARCHAR(200) NOT NULL,
				"ActionTypeName" VARCHAR(200) NOT NULL,
				"BillDay" VARCHAR(200) NOT NULL,
				"BillId" VARCHAR(200) NOT NULL,
				"BillMonth" VARCHAR(200) NOT NULL,
				"BusinessCode" VARCHAR(200) NOT NULL,
				"BusinessCodeName" VARCHAR(200) NOT NULL,
				"BlendedDiscount" NUMERIC NOT NULL,
				"CashPayAmount" NUMERIC NOT NULL,
				"ComponentCode" VARCHAR(200) NOT NULL,
				"ComponentCodeName" VARCHAR(200) NOT NULL,
				"ContractPrice" NUMERIC NOT NULL,
				"Cost" NUMERIC NOT NULL,
				"DeductedMeasure" VARCHAR(200) NOT NULL,
				"Discount" VARCHAR(200) NOT NULL,
				"IncentivePayAmount" VARCHAR(200) NOT NULL,
				"InstanceType" VARCHAR(200) NOT NULL,
				"ItemCode" VARCHAR(200) NOT NULL,
				"ItemCodeName" VARCHAR(200) NOT NULL,
				"OriginalCostWithRI" NUMERIC NOT NULL,
				"OriginalCostWithSP" NUMERIC NOT NULL,
				"PriceUnit" VARCHAR(200) NOT NULL,
				"RealCost" NUMERIC NOT NULL,
				"RealTotalMeasure" VARCHAR(200) NOT NULL,
				"ReduceType" VARCHAR(200) NOT NULL,
				"RiTimeSpan" NUMERIC NOT NULL,
				"SPDeductionRate" NUMERIC NOT NULL,
				"SinglePrice" NUMERIC NOT NULL,
				"TimeSpan" VARCHAR(200) NOT NULL,
				"TimeUnitName" VARCHAR(200) NOT NULL,
				"TransferPayAmount" VARCHAR(200) NOT NULL,
				"UsedAmount" VARCHAR(200) NOT NULL,
				"UsedAmountUnit" VARCHAR(200) NOT NULL,
				"VoucherPayAmount" VARCHAR(200) NOT NULL,
				"DiscountContent" VARCHAR(200) NOT NULL,
				"DiscountObject" VARCHAR(200) NOT NULL,
				"DiscountType" VARCHAR(200) NOT NULL,
				"FeeBeginTime" VARCHAR(200) NOT NULL,
				"FeeEndTime" VARCHAR(200) NOT NULL,
				"Formula" VARCHAR(200) NOT NULL,
				"FormulaUrl" VARCHAR(200) NOT NULL,
				"Id" VARCHAR(200) NOT NULL,
				"OperateUin" VARCHAR(200) NOT NULL,
				"OrderId" VARCHAR(200) NOT NULL,
				"OwnerUin" VARCHAR(200) NOT NULL,
				"PayModeName" VARCHAR(200) NOT NULL,
				"PayTime" VARCHAR(200) NOT NULL,
				"PayerUin" VARCHAR(200) NOT NULL,
				"PriceInfo" VARCHAR(200) NOT NULL,
				"ProductCode" VARCHAR(200) NOT NULL,
				"ProductCodeName" VARCHAR(200) NOT NULL,
				"ProjectId" VARCHAR(200) NOT NULL,
				"ProjectName" VARCHAR(200) NOT NULL,
				"RegionId" VARCHAR(200) NOT NULL,
				"RegionName" VARCHAR(200) NOT NULL,
				"RegionType" VARCHAR(200) NOT NULL,
				"RegionTypeName" VARCHAR(200) NOT NULL,
				"ReserveDetail" VARCHAR(200) NOT NULL,
				"ResourceId" VARCHAR(200) NOT NULL,
				"ResourceName" VARCHAR(200) NOT NULL,
				"Tags" VARCHAR(1000) NOT NULL,
				"ZoneName" VARCHAR(200) NOT NULL
				);`
			rows2, err := db.Query(sqlData)
			if err != nil {
				log.Fatalln("创建表失败：", err)
			}
			rows2.Close()

			commitList := [][]string{
				{"ActionType", "操作类型"},
				{"ActionTypeName", "操作类型名称"},
				{"BillDay", "账单日"},
				{"BillId", "账单ID"},
				{"BillMonth", "账单月份"},
				{"BusinessCode", "产品名称代码"},
				{"BusinessCodeName", "产品名称"},
				{"BlendedDiscount", "混合折扣率"},
				{"CashPayAmount", "现金支付"},
				{"ComponentCode", "组件代码"},
				{"ComponentCodeName", "组件代码名称"},
				{"ContractPrice", "合同价格"},
				{"Cost", "成本"},
				{"DeductedMeasure", "抵扣用量"},
				{"Discount", "折扣"},
				{"IncentivePayAmount", "激励支付金额"},
				{"InstanceType", "实例类型"},
				{"ItemCode", "组件编码"},
				{"ItemCodeName", "组件名称"},
				{"OriginalCostWithRI", "预留实例原始成本"},
				{"OriginalCostWithSP", "节省计划原始成本"},
				{"PriceUnit", "价格单位"},
				{"RealCost", "实际成本"},
				{"RealTotalMeasure", "实际总用量"},
				{"ReduceType", "减免类型"},
				{"RiTimeSpan", "预留实例时间跨度"},
				{"SPDeductionRate", "节省计划抵扣率"},
				{"SinglePrice", "单价"},
				{"TimeSpan", "时间跨度"},
				{"TimeUnitName", "时间单位名称"},
				{"TransferPayAmount", "转账支付金额"},
				{"UsedAmount", "使用量"},
				{"UsedAmountUnit", "使用量单位"},
				{"VoucherPayAmount", "代金券支付金额"},
				{"DiscountContent", "折扣内容"},
				{"DiscountObject", "折扣对象"},
				{"DiscountType", "折扣类型"},
				{"FeeBeginTime", "费用开始时间"},
				{"FeeEndTime", "费用结束时间"},
				{"Formula", "公式"},
				{"FormulaUrl", "公式URL"},
				{"Id", "ID"},
				{"OperateUin", "操作UIN"},
				{"OrderId", "订单ID"},
				{"OwnerUin", "所有者UIN"},
				{"PayModeName", "支付方式名称"},
				{"PayTime", "支付时间"},
				{"PayerUin", "付款人UIN"},
				{"PriceInfo", "价格信息"},
				{"ProductCode", "子产品编码"},
				{"ProductCodeName", "子产品名称"},
				{"ProjectId", "项目ID"},
				{"ProjectName", "项目名称"},
				{"RegionId", "地域ID"},
				{"RegionName", "地域名称"},
				{"RegionType", "地域类型"},
				{"RegionTypeName", "地域类型名称"},
				{"ReserveDetail", "预留详情"},
				{"ResourceId", "资源ID"},
				{"ResourceName", "资源名称"},
				{"Tags", "标签"},
				{"ZoneName", "可用区名称"},
			}
			for _, v := range commitList {
				sqlData := fmt.Sprintf(`COMMENT ON COLUMN "tencentDescribeBillDetail"."%s" IS '%s'`, v[0], v[1])
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
	var response struct {
		Response struct {
			DetailSet []struct {
				ActionType       string        `json:"ActionType"`
				ActionTypeName   string        `json:"ActionTypeName"`
				BillDay          string        `json:"BillDay"`
				BillId           string        `json:"BillId"`
				BillMonth        string        `json:"BillMonth"`
				BusinessCode     string        `json:"BusinessCode"`
				BusinessCodeName string        `json:"BusinessCodeName"`
				PayModeName      string        `json:"PayModeName"`
				ProjectName      string        `json:"ProjectName"`
				RegionName       string        `json:"RegionName"`
				ZoneName         string        `json:"ZoneName"`
				ResourceId       string        `json:"ResourceId"`
				ResourceName     string        `json:"ResourceName"`
				OrderId          string        `json:"OrderId"`
				PayTime          string        `json:"PayTime"`
				FeeBeginTime     string        `json:"FeeBeginTime"`
				FeeEndTime       string        `json:"FeeEndTime"`
				PayerUin         string        `json:"PayerUin"`
				OwnerUin         string        `json:"OwnerUin"`
				OperateUin       string        `json:"OperateUin"`
				ProductCode      string        `json:"ProductCode"`
				ProductCodeName  string        `json:"ProductCodeName"`
				RegionId         string        `json:"RegionId"`
				ProjectId        int64         `json:"ProjectId"`
				Formula          string        `json:"Formula"`
				FormulaUrl       string        `json:"FormulaUrl"`
				Id               string        `json:"Id"`
				RegionType       string        `json:"RegionType"`
				RegionTypeName   string        `json:"RegionTypeName"`
				ReserveDetail    string        `json:"ReserveDetail"`
				Tags             []interface{} `json:"Tags"`
				ComponentSet     []struct {
					ComponentCode      string `json:"ComponentCode"`
					ComponentCodeName  string `json:"ComponentCodeName"`
					ItemCode           string `json:"ItemCode"`
					ItemCodeName       string `json:"ItemCodeName"`
					SinglePrice        string `json:"SinglePrice"`
					PriceUnit          string `json:"PriceUnit"`
					UsedAmount         string `json:"UsedAmount"`
					UsedAmountUnit     string `json:"UsedAmountUnit"`
					RealTotalMeasure   string `json:"RealTotalMeasure"`
					DeductedMeasure    string `json:"DeductedMeasure"`
					TimeSpan           string `json:"TimeSpan"`
					TimeUnitName       string `json:"TimeUnitName"`
					Cost               string `json:"Cost"`
					Discount           string `json:"Discount"`
					ReduceType         string `json:"ReduceType"`
					RealCost           string `json:"RealCost"`
					VoucherPayAmount   string `json:"VoucherPayAmount"`
					CashPayAmount      string `json:"CashPayAmount"`
					IncentivePayAmount string `json:"IncentivePayAmount"`
					TransferPayAmount  string `json:"TransferPayAmount"`
					ContractPrice      string `json:"ContractPrice"`
					InstanceType       string `json:"InstanceType"`
					RiTimeSpan         string `json:"RiTimeSpan"`
					OriginalCostWithRI string `json:"OriginalCostWithRI"`
					SPDeductionRate    string `json:"SPDeductionRate"`
					OriginalCostWithSP string `json:"OriginalCostWithSP"`
					BlendedDiscount    string `json:"BlendedDiscount"`
				} `json:"ComponentSet"`
			} `json:"DetailSet"`
		} `json:"Response"`
	}

	err := json.Unmarshal(data, &response)
	if err != nil {
		log.Printf("解析返回值失败: %v", err)
		return
	}

	if len(response.Response.DetailSet) == 0 {
		log.Println("没有数据需要写入")
		return
	}

	db, err := sql.Open("postgres", sqlConnStr)
	if err != nil {
		log.Fatalln("连接数据库失败", err)
	}
	defer db.Close()

	// 使用参数化查询防止SQL注入，字段顺序与创建表的顺序一致
	sqlQuery := `INSERT INTO "tencentDescribeBillDetail"(
		"ActionType", "ActionTypeName", "BillDay", "BillId", "BillMonth", 
		"BusinessCode", "BusinessCodeName", "BlendedDiscount", "CashPayAmount", "ComponentCode", 
		"ComponentCodeName", "ContractPrice", "Cost", "DeductedMeasure", "Discount", 
		"IncentivePayAmount", "InstanceType", "ItemCode", "ItemCodeName", "OriginalCostWithRI", 
		"OriginalCostWithSP", "PriceUnit", "RealCost", "RealTotalMeasure", "ReduceType", 
		"RiTimeSpan", "SPDeductionRate", "SinglePrice", "TimeSpan", "TimeUnitName", 
		"TransferPayAmount", "UsedAmount", "UsedAmountUnit", "VoucherPayAmount", "DiscountContent", 
		"DiscountObject", "DiscountType", "FeeBeginTime", "FeeEndTime", "Formula", 
		"FormulaUrl", "Id", "OperateUin", "OrderId", "OwnerUin", 
		"PayModeName", "PayTime", "PayerUin", "PriceInfo", "ProductCode", 
		"ProductCodeName", "ProjectId", "ProjectName", "RegionId", "RegionName", 
		"RegionType", "RegionTypeName", "ReserveDetail", "ResourceId", "ResourceName", 
		"Tags", "ZoneName"
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
		$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
		$31, $32, $33, $34, $35, $36, $37, $38, $39, $40,
		$41, $42, $43, $44, $45, $46, $47, $48, $49, $50,
		$51, $52, $53, $54, $55, $56, $57, $58, $59, $60,
		$61, $62
	)`

	successCount := 0
	totalRecords := 0

	// 遍历每个账单项目
	for _, detail := range response.Response.DetailSet {
		// 将Tags转换为JSON字符串
		tagsJson, _ := json.Marshal(detail.Tags)

		// 遍历每个组件
		for _, component := range detail.ComponentSet {
			totalRecords++

			// 转换数值字段，如果转换失败则使用0
			blendedDiscount := parseFloat(component.BlendedDiscount)
			cashPayAmount := parseFloat(component.CashPayAmount)
			contractPrice := parseFloat(component.ContractPrice)
			cost := parseFloat(component.Cost)
			originalCostWithRI := parseFloat(component.OriginalCostWithRI)
			originalCostWithSP := parseFloat(component.OriginalCostWithSP)
			realCost := parseFloat(component.RealCost)
			riTimeSpan := parseFloat(component.RiTimeSpan)
			spDeductionRate := parseFloat(component.SPDeductionRate)
			singlePrice := parseFloat(component.SinglePrice)

			_, err := db.Exec(sqlQuery,
				detail.ActionType,                   // $1 - ActionType
				detail.ActionTypeName,               // $2 - ActionTypeName
				detail.BillDay,                      // $3 - BillDay
				detail.BillId,                       // $4 - BillId
				detail.BillMonth,                    // $5 - BillMonth
				detail.BusinessCode,                 // $6 - BusinessCode
				detail.BusinessCodeName,             // $7 - BusinessCodeName
				blendedDiscount,                     // $8 - BlendedDiscount
				cashPayAmount,                       // $9 - CashPayAmount
				component.ComponentCode,             // $10 - ComponentCode
				component.ComponentCodeName,         // $11 - ComponentCodeName
				contractPrice,                       // $12 - ContractPrice
				cost,                                // $13 - Cost
				component.DeductedMeasure,           // $14 - DeductedMeasure
				component.Discount,                  // $15 - Discount
				component.IncentivePayAmount,        // $16 - IncentivePayAmount
				component.InstanceType,              // $17 - InstanceType
				component.ItemCode,                  // $18 - ItemCode
				component.ItemCodeName,              // $19 - ItemCodeName
				originalCostWithRI,                  // $20 - OriginalCostWithRI
				originalCostWithSP,                  // $21 - OriginalCostWithSP
				component.PriceUnit,                 // $22 - PriceUnit
				realCost,                            // $23 - RealCost
				component.RealTotalMeasure,          // $24 - RealTotalMeasure
				component.ReduceType,                // $25 - ReduceType
				riTimeSpan,                          // $26 - RiTimeSpan
				spDeductionRate,                     // $27 - SPDeductionRate
				singlePrice,                         // $28 - SinglePrice
				component.TimeSpan,                  // $29 - TimeSpan
				component.TimeUnitName,              // $30 - TimeUnitName
				component.TransferPayAmount,         // $31 - TransferPayAmount
				component.UsedAmount,                // $32 - UsedAmount
				component.UsedAmountUnit,            // $33 - UsedAmountUnit
				component.VoucherPayAmount,          // $34 - VoucherPayAmount
				"",                                  // $35 - DiscountContent (示例数据中没有此字段)
				"",                                  // $36 - DiscountObject (示例数据中没有此字段)
				"",                                  // $37 - DiscountType (示例数据中没有此字段)
				detail.FeeBeginTime,                 // $38 - FeeBeginTime
				detail.FeeEndTime,                   // $39 - FeeEndTime
				detail.Formula,                      // $40 - Formula
				detail.FormulaUrl,                   // $41 - FormulaUrl
				detail.Id,                           // $42 - Id
				detail.OperateUin,                   // $43 - OperateUin
				detail.OrderId,                      // $44 - OrderId
				detail.OwnerUin,                     // $45 - OwnerUin
				detail.PayModeName,                  // $46 - PayModeName
				detail.PayTime,                      // $47 - PayTime
				detail.PayerUin,                     // $48 - PayerUin
				"",                                  // $49 - PriceInfo (示例数据中为空数组)
				detail.ProductCode,                  // $50 - ProductCode
				detail.ProductCodeName,              // $51 - ProductCodeName
				fmt.Sprintf("%d", detail.ProjectId), // $52 - ProjectId
				detail.ProjectName,                  // $53 - ProjectName
				detail.RegionId,                     // $54 - RegionId
				detail.RegionName,                   // $55 - RegionName
				detail.RegionType,                   // $56 - RegionType
				detail.RegionTypeName,               // $57 - RegionTypeName
				detail.ReserveDetail,                // $58 - ReserveDetail
				detail.ResourceId,                   // $59 - ResourceId
				detail.ResourceName,                 // $60 - ResourceName
				string(tagsJson),                    // $61 - Tags
				detail.ZoneName)                     // $62 - ZoneName

			if err != nil {
				log.Printf("插入数据失败: %v, 资源ID: %s, 组件: %s", err, detail.ResourceId, component.ComponentCodeName)
			} else {
				successCount++
			}
		}
	}
	log.Printf("写入数据完成，成功插入 %d/%d 条记录", successCount, totalRecords)
}

// parseFloat 辅助函数，用于安全地将字符串转换为float64
func parseFloat(s string) float64 {
	if s == "" || s == "-" {
		return 0.0
	}

	result, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0
	}
	return result
}
