package aliyunDescribeInstanceBill

import (
  "encoding/json"
  "strings"
  "fmt"
  "os"
  bssopenapi20171214  "github.com/alibabacloud-go/bssopenapi-20171214/v6/client"
  openapi  "github.com/alibabacloud-go/darabonba-openapi/v2/client"
  util  "github.com/alibabacloud-go/tea-utils/v2/service"
  credential  "github.com/aliyun/credentials-go/credentials"
  "github.com/alibabacloud-go/tea/tea"
)


// Description:
// 
// 使用凭据初始化账号Client
// 
// @return Client
// 
// @throws Exception
func CreateClient () (_result *bssopenapi20171214.Client, _err error) {
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

var resStr *string

func _main (args []*string) (_err error) {
  client, _err := CreateClient()
  if _err != nil {
    return _err
  }

  cstSh, _ := time.LoadLocation("Asia/Shanghai")
  currentMonth := time.Now().In(cstSh).Format("2006-01")
  billingDate := time.Now().AddDate(0, 0, -1).In(cstSh).Format("2006-01-02") // 查询昨天的账单

  describeInstanceBillRequest := &bssopenapi20171214.DescribeInstanceBillRequest{
    BillingCycle: tea.String(currentMonth),
    BillingDate: tea.String(billingDate),
    Granularity: tea.String("DAILY"),
    MaxResults: tea.Int32(50),
  }
  runtime := &util.RuntimeOptions{}
  tryErr := func()(_e error) {
    defer func() {
      if r := tea.Recover(recover()); r != nil {
        _e = r
      }
    }()
    // 复制代码运行请自行打印 API 的返回值
    res, _err = client.DescribeInstanceBillWithOptions(describeInstanceBillRequest, runtime)
    if _err != nil {
      return _err
    }

    resStr = res
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
    log.Println(err)
  }
  return *resStr
}