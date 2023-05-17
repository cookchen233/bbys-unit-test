package main

import (
	_ "github.com/joho/godotenv/autoload"
	"os"
	"testing"
)

func main() {
	args := os.Args
	switch args[1] {
	case "ExecTempCrontab":
		ExecTempCrontab(&testing.T{})
	}
}
