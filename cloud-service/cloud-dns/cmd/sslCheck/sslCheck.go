package sslCheck

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net"
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

	dialer := net.Dialer{Timeout: time.Second * 3}

	for _, record := range recordList {
		conn, err := tls.DialWithDialer(&dialer, "tcp", record+":443", nil)
		if err != nil {
			log.Println("连接错误或未配置证书：", record)
		} else {
			cert := conn.ConnectionState().PeerCertificates[0]

			// 时间信息
			//fmt.Printf("NotBefore: %v\n", cert.NotBefore.In(cstSh))
			//fmt.Printf("NotAfter: %v\n", cert.NotAfter.In(cstSh))
			//fmt.Printf("Issuer: %v\n", cert.Issuer)
			//fmt.Printf("Subject: %v\n", cert.Subject)

			var cstSh, _ = time.LoadLocation("Asia/Shanghai")
			// nowTime, _ := time.Parse("2006-01-02 15:04:05", time.Now().In(cstSh).Format("2006-01-02 15:04:05"))
			// endTime, _ := time.Parse("2006-01-02 15:04:05", cert.NotAfter.In(cstSh).Format("2006-01-02 15:04:05"))
			// d := endTime.Sub(nowTime).Hours() / 24
			// dStr := strings.Split(strconv.FormatFloat(d, 'g', -1, 64), ".")
			// dInt, _ := strconv.Atoi(dStr[0])

			// if dInt < 0 {
			// 	log.Printf("%s：证书已到期", record)
			// } else {
			// 	log.Printf("%s：到期时间还有%d天", record, dInt)
			// }

			sqlData := fmt.Sprintf(`UPDATE public."tencentDomainList" SET "NotBefore" = '%s', "NotAfter" = '%s', "Subject" = '%s' WHERE "Domain"='%s'`, cert.NotBefore.In(cstSh), cert.NotAfter.In(cstSh), cert.Subject, record)
			rows, err := db.Query(sqlData)
			if err != nil {
				log.Fatalln("写入表失败：", err, sqlData)
			}
			rows.Close()
		}
	}
}
