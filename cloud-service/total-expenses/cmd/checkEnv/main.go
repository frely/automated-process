package checkEnv

import (
	"fmt"
	"os"
)

func Check() {
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
