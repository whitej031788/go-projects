package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

/* Fill out your configuration options below, and then you can run the script from the command line */
const (
	PaddleAPIURL     = "https://sandbox-vendors.paddle.com"
	PaddleVendorID   = "7"
	PaddleAuthKey    = "bacdaf1fa8dcacd80bcc9829ed5fefaca409cf6121da4aa423"
	ProfitwellAPIKey = "2F077A41764EA0ACDEE97A4C6BD613F6"
	EndDate          = "2020-01-30"
)

func main() {
	apiResults := callPaddleListUsers()

	if apiResults["success"].(bool) {
		for _, element := range apiResults["response"].([]interface{}) {
			theMap := element.(map[string]interface{})
			theCurrency, theValue := getMRRValue(theMap)
			if theCurrency == "INVALID" {
				fmt.Println("The last amount and next amount aren't the same, so we aren't sure what the MRR value is:", theMap["subscription_id"].(float64))
				continue
			}

			planInterval, planId := getPlanInfo(int(theMap["plan_id"].(float64)))

			jsonData := map[string]interface{}{
				"user_alias":         md5Hash(int(theMap["user_id"].(float64))),
				"subscription_alias": md5Hash(int(theMap["subscription_id"].(float64))),
				"email":              theMap["user_email"].(string),
				"plan_id":            planId,
				"plan_interval":      planInterval,
				"plan_currency":      theCurrency,
				"status":             "active",
				"value":              theValue,
				"effective_date":     convertToUnixTimeStamp(theMap["signup_date"].(string), true, false),
			}
			bolB, _ := json.Marshal(jsonData)
			fmt.Println(string(bolB))
		}
	} else {
		fmt.Println("The List Users API call failed")
	}
}

func md5Hash(id int) string {
	data := make([]byte, id)

	return fmt.Sprintf("%x", md5.Sum(data))
}

func callPaddleListUsers() map[string]interface{} {
	var result map[string]interface{}
	jsonData := map[string]string{"vendor_id": PaddleVendorID, "vendor_auth_code": PaddleAuthKey, "state": "active"}
	jsonValue, _ := json.Marshal(jsonData)
	response, err := http.Post(PaddleAPIURL+"/api/2.0/subscription/users", "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("The List Users API call request failed with error %s\n", err)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		json.Unmarshal([]byte(data), &result)
	}
	return result
}

// DEPRECATED FUNCTION BELOW //
func getSubscriptionCurrency(SubID int) string {
	currency := ""
	var result map[string]interface{}
	jsonData := map[string]interface{}{"vendor_id": PaddleVendorID, "vendor_auth_code": PaddleAuthKey, "subscription_id": SubID}
	jsonValue, _ := json.Marshal(jsonData)
	response, err := http.Post(PaddleAPIURL+"/api/2.0/subscription/payments", "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("The List Payments API call request failed with error %s\n", err)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		json.Unmarshal([]byte(data), &result)
	}

	if result["success"].(bool) {
		for _, element := range result["response"].([]interface{}) {
			theMap := element.(map[string]interface{})
			currency = theMap["currency"].(string)
			break
		}
	} else {
		fmt.Printf("The List Payments API call request failed with error %s\n", err)
	}

	return currency
}

func getPlanInfo(planID int) (string, string) {
	planIDMap := map[int]map[string]string{
		902:  {"name": "Test My Plan", "interval": "year"},
		919:  {"name": "New Planz", "interval": "month"},
		920:  {"name": "Alternate Year", "interval": "year"},
		1018: {"name": "Free Plan Daily", "interval": "month"},
		1153: {"name": "Quantity Test", "interval": "year"},
		1195: {"name": "Framer Monthy Seats", "interval": "month"},
		1196: {"name": "Framer Annual Seats", "interval": "year"},
	}

	planIDName := planIDMap[planID]["name"]
	planInterval := planIDMap[planID]["interval"]
	return planInterval, planIDName
}

func convertToUnixTimeStamp(dateString string, withHoursMinutesSeconds bool, withMicroseconds bool) int64 {
	layout := "2006-01-02"

	if withHoursMinutesSeconds {
		layout = layout + " 15:04:05"
	}

	if withMicroseconds {
		layout = layout + ".000Z"
	}

	timeStamp, err := time.Parse(layout, dateString)
	if err != nil {
		return 0
	}

	return timeStamp.Unix()
}

func getMRRValue(theMap map[string]interface{}) (string, float64) {
	nextPayment := theMap["next_payment"].(map[string]interface{})
	lastPayment := theMap["last_payment"].(map[string]interface{})
	nextAmount := nextPayment["amount"].(float64)
	lastAmount := lastPayment["amount"].(float64)
	theCurr := nextPayment["currency"].(string)

	if nextAmount != lastAmount {
		theCurr = "INVALID"
	}

	return theCurr, (nextAmount + lastAmount) / float64(2) * 100
}
