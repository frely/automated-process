package checkEnv

import (
	"fmt"
	"os"
)

func checkEnv() {
	if os.Getenv("POSTGRES_HOST") == "" {
		fmt.Println("POSTGRES_HOST Not Find")
		os.Exit(0)
	}
	if os.Getenv("POSTGRES_PORT") == "" {
		fmt.Println("POSTGRES_PORT Not Find")
		os.Exit(0)
	}
	if os.Getenv("POSTGRES_DB") == "" {
		fmt.Println("POSTGRES_DB Not Find")
		os.Exit(0)
	}
	if os.Getenv("POSTGRES_USER") == "" {
		fmt.Println("POSTGRES_USER Not Find")
		os.Exit(0)
	}
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		fmt.Println("POSTGRES_PASSWORD Not Find")
		os.Exit(0)
	}
}

func CheckAliyun() {
	checkEnv()
	if os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_ID") == "" {
		fmt.Println("ALIBABA_CLOUD_ACCESS_KEY_ID Not Find")
		os.Exit(0)
	}
	if os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET") == "" {
		fmt.Println("ALIBABA_CLOUD_ACCESS_KEY_SECRET Not Find")
		os.Exit(0)
	}
}

func CheckTencent() {
	checkEnv()
	if os.Getenv("SecretId") == "" {
		fmt.Println("SecretId Not Find")
		os.Exit(0)
	}
	if os.Getenv("SecretKey") == "" {
		fmt.Println("SecretKey Not Find")
		os.Exit(0)
	}
}
