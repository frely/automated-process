package sslCheck

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

var (
	sqlConnStr string
)

func Check() {
	sqlConnStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		viper.GetString("POSTGRES_USER"),
		viper.GetString("POSTGRES_PASSWORD"),
		viper.GetString("POSTGRES_HOST"),
		viper.GetString("POSTGRES_PORT"),
		viper.GetString("POSTGRES_DB"))
	data := getCheckList()
	check(data)
}

func getCheckList() []string {
	db, err := sql.Open("postgres", sqlConnStr)
	if err != nil {
		log.Fatalln("连接数据库失败", err)
	}
	defer db.Close()

	rows, err := db.Query(`select "Record" from public."tencentDomainList" where "Status" = 'ENABLE'`)
	defer rows.Close()
	if err != nil {
		log.Fatalln("查询表失败：", err)
	}
	recordList := []string{}
	for rows.Next() {
		var record string
		err := rows.Scan(&record)
		if err != nil {
			log.Fatalln("获取表失败", err)
		}
		recordList = append(recordList, record)
	}
	if err := rows.Err(); err != nil {
		log.Fatalln("查询sql失败", err)
	}

	return recordList
}

func check(recordList []string) {
	db, err := sql.Open("postgres", sqlConnStr)
	if err != nil {
		log.Fatalln("连接数据库失败", err)
	}
	defer db.Close()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			// 跳过证书验证
			InsecureSkipVerify: true,
			// 配置使用的协议类型
			CipherSuites: []uint16{
				// TLS 1.0 - 1.2 cipher suites.
				tls.TLS_RSA_WITH_RC4_128_SHA,
				tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
				tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
				// TLS 1.3 cipher suites.
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			},
		},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	for _, record := range recordList {

		resp, err := client.Get("https://" + record)
		if err != nil {
			fmt.Println(err)
		}
		resp.Body.Close()
		if err != nil {
			log.Println("连接错误或未配置证书：", record)
		} else {
			var cstSh, _ = time.LoadLocation("Asia/Shanghai")
			// nowTime, _ := time.Parse("2006-01-02 15:04:05", time.Now().In(cstSh).Format("2006-01-02 15:04:05"))
			// endTime, _ := time.Parse("2006-01-02 15:04:05", resp.TLS.PeerCertificates[0].NotAfter.In(cstSh).Format("2006-01-02 15:04:05"))
			// d := endTime.Sub(nowTime).Hours() / 24
			// dStr := strings.Split(strconv.FormatFloat(d, 'g', -1, 64), ".")
			// dInt, _ := strconv.Atoi(dStr[0])

			// if dInt < 0 {
			// 	log.Printf("%s：证书已到期", record)
			// } else {
			// 	log.Printf("%s：到期时间还有%d天", record, dInt)
			// }

			sqlData := fmt.Sprintf(`UPDATE public."tencentDomainList" SET "NotBefore" = '%s', "NotAfter" = '%s', "Subject" = '%s' WHERE "Domain"='%s'`,
				resp.TLS.PeerCertificates[0].NotBefore.In(cstSh).Format("2006-01-02 15:04:05"),
				resp.TLS.PeerCertificates[0].NotAfter.In(cstSh).Format("2006-01-02 15:04:05"),
				resp.TLS.PeerCertificates[0].Subject,
				record)
			rows, err := db.Query(sqlData)
			if err != nil {
				log.Fatalln("写入表失败：", err, sqlData)
			}
			rows.Close()
		}
	}
}
