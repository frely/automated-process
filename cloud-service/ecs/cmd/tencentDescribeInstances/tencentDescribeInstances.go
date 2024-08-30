package tencentDescribeInstances

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/frely/automated-process/cloud-service/ecs/cmd/tencentDescribeRegions"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

var (
	sqlConnStr string
)

type AutoGenerated struct {
	Response struct {
		TotalCount  int `json:"TotalCount"`
		InstanceSet []struct {
			Placement struct {
				Zone      string `json:"Zone"`
				ProjectId int    `json:"ProjectId"`
			} `json:"Placement"`
			InstanceId         string `json:"InstanceId"`
			InstanceType       string `json:"InstanceType"`
			CPU                int    `json:"CPU"`
			Memory             int    `json:"Memory"`
			RestrictState      string `json:"RestrictState"`
			InstanceName       string `json:"InstanceName"`
			InstanceChargeType string `json:"InstanceChargeType"`
			SystemDisk         struct {
				CdcID    any    `json:"CdcId"`
				DiskType string `json:"DiskType"`
				DiskId   string `json:"DiskId"`
				DiskSize int    `json:"DiskSize"`
			} `json:"SystemDisk"`
			DataDisks []struct {
				DiskSize              int    `json:"DiskSize"`
				DiskType              string `json:"DiskType"`
				DiskID                string `json:"DiskId"`
				DeleteWithInstance    bool   `json:"DeleteWithInstance"`
				Encrypt               bool   `json:"Encrypt"`
				ThroughputPerformance int    `json:"ThroughputPerformance"`
			} `json:"DataDisks"`
			PrivateIpAddresses []string `json:"PrivateIpAddresses"`
			PublicIpAddresses  []string `json:"PublicIpAddresses"`
			InternetAccessible struct {
				InternetChargeType      string `json:"InternetChargeType"`
				InternetMaxBandwidthOut int    `json:"InternetMaxBandwidthOut"`
			} `json:"InternetAccessible"`
			VirtualPrivateCloud struct {
				VpcId        string `json:"VpcId"`
				SubnetId     string `json:"SubnetId"`
				AsVpcGateway bool   `json:"AsVpcGateway"`
			} `json:"VirtualPrivateCloud"`
			ImageId          string    `json:"ImageId"`
			RenewFlag        string    `json:"RenewFlag"`
			CreatedTime      time.Time `json:"CreatedTime"`
			ExpiredTime      time.Time `json:"ExpiredTime"`
			OsName           string    `json:"OsName"`
			SecurityGroupIds []string  `json:"SecurityGroupIds"`
			LoginSettings    struct {
			} `json:"LoginSettings"`
			InstanceState            string `json:"InstanceState"`
			Tags                     []any  `json:"Tags"`
			StopChargingMode         string `json:"StopChargingMode"`
			UUID                     string `json:"Uuid"`
			LatestOperation          string `json:"LatestOperation"`
			LatestOperationState     string `json:"LatestOperationState"`
			LatestOperationRequestId string `json:"LatestOperationRequestId"`
			DisasterRecoverGroupId   string `json:"DisasterRecoverGroupId"`
			CamRoleName              string `json:"CamRoleName"`
			HpcClusterId             string `json:"HpcClusterId"`
			DedicatedClusterId       string `json:"DedicatedClusterId"`
			IsolatedSource           string `json:"IsolatedSource"`
			LicenseType              string `json:"LicenseType"`
			DisableAPITermination    bool   `json:"DisableApiTermination"`
			DefaultLoginUser         string `json:"DefaultLoginUser"`
			DefaultLoginPort         int    `json:"DefaultLoginPort"`
		} `json:"InstanceSet"`
		RequestId string `json:"RequestId"`
	} `json:"Response"`
}

// 写入cvm信息到数据库
func ToSql() {
	customRegion := viper.GetString("customRegion")
	if customRegion != "" {
		// 查询指定Region
		viper.Set("RegionId", customRegion)
		cvmList := getEcs()
		writeSql(cvmList)
	} else {
		for _, v := range tencentDescribeRegions.Get() {
			viper.Set("RegionId", v)
			cvmList := getEcs()
			writeSql(cvmList)
			time.Sleep(1 * time.Second)
		}
	}
}

func getEcs() string {
	// 实例化一个认证对象，入参需要传入腾讯云账户 SecretId 和 SecretKey，此处还需注意密钥对的保密
	// 代码泄露可能会导致 SecretId 和 SecretKey 泄露，并威胁账号下所有资源的安全性。以下代码示例仅供参考，建议采用更安全的方式来使用密钥，请参见：https://cloud.tencent.com/document/product/1278/85305
	// 密钥可前往官网控制台 https://console.cloud.tencent.com/cam/capi 进行获取
	credential := common.NewCredential(
		viper.GetString("SecretId"),
		viper.GetString("SecretKey"),
	)
	// 实例化一个client选项，可选的，没有特殊需求可以跳过
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cvm.tencentcloudapi.com"
	// 实例化要请求产品的client对象,clientProfile是可选的
	client, _ := cvm.NewClient(credential, viper.GetString("RegionId"), cpf)

	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := cvm.NewDescribeInstancesRequest()

	// 返回的resp是一个DescribeInstancesResponse的实例，与请求对象对应
	response, err := client.DescribeInstances(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		log.Fatalln("An API error has returned: %s", err)
	}
	if err != nil {
		panic(err)
	}
	// 输出json格式的字符串回包
	//fmt.Printf("%s", response.ToJsonString())
	return response.ToJsonString()
}

// 如果不存在表，则新建。
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

	rows, err := db.Query(`select count(*) from pg_class where relname = 'tencentCvm';`)
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
			sqlData := `TRUNCATE TABLE "tencentCvm"`
			rows2, err := db.Query(sqlData)
			if err != nil {
				log.Fatalln("清空表失败：", err)
			}
			rows2.Close()
		} else {
			log.Println("创建表：tencentCvm")
			sqlData := `
				CREATE TABLE "public"."tencentCvm" (
				"InstanceId" VARCHAR(200) NOT NULL,
				"InstanceName" VARCHAR(200) NOT NULL,
				"InstanceState" VARCHAR(200) NOT NULL,
				"InstanceType" VARCHAR(200) NOT NULL,
				"CPU" INTEGER NOT NULL,
				"Memory" INTEGER NOT NULL,
				"OsName" VARCHAR(200) NOT NULL,
				"SystemDisk" VARCHAR(200) NOT NULL
				"DataDisks" VARCHAR(200) NOT NULL,
				"PrivateIpAddresses" VARCHAR(200) NOT NULL,
				"PublicIpAddresses" VARCHAR(200) NOT NULL,
				"Placement" VARCHAR(200) NOT NULL);`
			rows2, err := db.Query(sqlData)
			if err != nil {
				log.Fatalln("创建表失败：", err)
			}
			rows2.Close()

			commitList := [][]string{
				{"InstanceId", "实例ID"},
				{"InstanceName", "实例名称"},
				{"InstanceState", "实例状态"},
				{"InstanceType", "实例规格"},
				{"CPU", "vCPU数"},
				{"Memory", "内存，单位GB"},
				{"OsName", "操作系统"},
				{"SystemDisk", "系统盘"},
				{"DataDisks", "数据盘"},
				{"PrivateIpAddresses", "实例的私网IP信息"},
				{"PublicIpAddresses", "实例公网IP地址"},
				{"Placement", "可用区"}}
			for _, v := range commitList {
				sqlData := fmt.Sprintf(`COMMENT ON COLUMN "tencentCvm"."%s" IS '%s'`, v[0], v[1])
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

// 写入数据库
func writeSql(cvmList string) {
	if cvmList == "" {
		log.Fatalln("cvm is null")
	}
	var p AutoGenerated
	err := json.Unmarshal([]byte(cvmList), &p)
	if err != nil {
		log.Fatalln("解析返回值失败", err)
	}

	db, err := sql.Open("postgres", sqlConnStr)
	if err != nil {
		log.Fatalln("连接数据库失败 func writeSql()", err)
	}
	defer db.Close()
	for _, v := range p.Response.InstanceSet {
		strSystemDisk := fmt.Sprintf(`["DiskSize": %s, "DiskType": %s]`, strconv.Itoa(v.SystemDisk.DiskSize), v.SystemDisk.DiskType)
		strDataDisks := []string{}
		for _, value := range v.DataDisks {
			tmpStr := fmt.Sprintf(`["DiskSize": %s, "DiskType": %s]`, strconv.Itoa(value.DiskSize), value.DiskType)
			strDataDisks = append(strDataDisks, tmpStr)
		}
		sqlData := fmt.Sprintf(`INSERT INTO "tencentCvm"("CPU","DataDisks","InstanceId","InstanceName","InstanceState","InstanceType","Memory","OsName","Placement","PrivateIpAddresses","PublicIpAddresses","SystemDisk") VALUES ('%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s')`,
			strconv.Itoa(v.CPU),
			strDataDisks,
			v.InstanceId,
			v.InstanceName,
			v.InstanceState,
			v.InstanceType,
			strconv.Itoa(v.Memory),
			v.OsName,
			v.Placement.Zone,
			v.PrivateIpAddresses,
			v.PublicIpAddresses,
			strSystemDisk)
		rows, err := db.Query(sqlData)
		if err != nil {
			log.Fatalln("sql执行失败 func writeSql(): ", err, sqlData)
		}
		rows.Close()
	}
	log.Println("写入数据完成")
}
