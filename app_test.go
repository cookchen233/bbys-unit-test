package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/tidwall/gjson"
	"testing"
)

var appApi = NewAppApi()

var appDataError *AppDataError

type CreatePrintOrder struct {
	location     gjson.Result
	locationId   string
	locationName string
	applySn      string
}

func (bind *CreatePrintOrder) Run(t *testing.T) {
	appApi.CreatePrintOrder()

}
