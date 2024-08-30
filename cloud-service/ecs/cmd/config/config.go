package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

// 配置文件初始化
func Init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			log.Println("配置文件不存在，创建默认配置文件")

			// 数据库
			viper.SetDefault("POSTGRES_HOST", "localhost")
			viper.SetDefault("POSTGRES_PORT", "5432")
			viper.SetDefault("POSTGRES_USER", "postgres")
			viper.SetDefault("POSTGRES_PASSWORD", "postgres")
			viper.SetDefault("POSTGRES_DB", "dbName")

			// 阿里云
			viper.SetDefault("ALIBABA_CLOUD_ACCESS_KEY_ID", "")
			viper.SetDefault("ALIBABA_CLOUD_ACCESS_KEY_SECRET", "")

			// 腾讯云
			viper.SetDefault("SecretId", "")
			viper.SetDefault("SecretKey", "")

			// 查询指定Regions
			viper.SetDefault("customRegions", "")

			viper.WriteConfigAs("config.yaml")
			os.Exit(0)

		} else {
			// Config file was found but another error was produced
			log.Fatalln("读取配置文件失败，请检查内容或者删除配置文件")
		}
	}
}
