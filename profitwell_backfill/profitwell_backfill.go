package main

import (
	"bytes"
	"crypto/md5"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	pwConfig "github.com/whitej031788/profitwell_backfill/config"
)

func main() {
	apiResults := callPaddleListUsers()

	if apiResults["success"].(bool) {
		// Keep track of any subscriptions that we can't determine MRR based on last / next payment
		file, err := os.Create("bad_subscriptions.csv")
		checkError("Cannot create file", err)
		defer file.Close()
		invalidLines := 0
		writer := csv.NewWriter(file)

		// Header row
		err = writeCsvLine([]string{"subscription_id", "email", "last_payment", "next_payment", "message"}, writer)
		checkError("Cannot write to file", err)

		// Populate Paddle Plan Info
		planIDMap := callPaddleListPlans()

		for _, element := range apiResults["response"].([]interface{}) {
			theMap := element.(map[string]interface{})

			theCurrency, theValue, lastAmt, nextAmt := getMRRValue(theMap)
			if theCurrency == "INVALID" {
				var data = []string{fmt.Sprintf("%.0f", theMap["subscription_id"].(float64)), theMap["user_email"].(string), fmt.Sprintf("%.2f", lastAmt), fmt.Sprintf("%.2f", nextAmt), "INV_MRR_VALUE"}
				invalidLines++
				err := writeCsvLine(data, writer)
				checkError("Cannot write to file", err)
				continue
			}

			planInterval, planID := getPlanInfo(int(theMap["plan_id"].(float64)), planIDMap)

			if theValue == 0 {
				var data = []string{fmt.Sprintf("%.0f", theMap["subscription_id"].(float64)), theMap["user_email"].(string), fmt.Sprintf("%.2f", lastAmt), fmt.Sprintf("%.2f", nextAmt), "ZERO_VALUE"}
				invalidLines++
				err := writeCsvLine(data, writer)
				checkError("Cannot write to file", err)
				continue
			}

			signupDate := convertToUnixTimeStamp(theMap["signup_date"].(string), true, false)

			endDateInUnix := convertToUnixTimeStamp(pwConfig.EndDate, true, false)

			if signupDate > endDateInUnix {
				var data = []string{fmt.Sprintf("%.0f", theMap["subscription_id"].(float64)), theMap["user_email"].(string), fmt.Sprintf("%.2f", lastAmt), fmt.Sprintf("%.2f", nextAmt), "NEWER_SUB"}
				invalidLines++
				err := writeCsvLine(data, writer)
				checkError("Cannot write to file", err)
				continue
			}

			jsonData := map[string]interface{}{
				"user_alias":         md5Hash(int(theMap["user_id"].(float64))),
				"subscription_alias": md5Hash(int(theMap["subscription_id"].(float64))),
				"email":              theMap["user_email"].(string),
				"plan_id":            planID,
				"plan_interval":      planInterval,
				"plan_currency":      theCurrency,
				"status":             "active",
				"value":              theValue,
				"effective_date":     signupDate,
			}

			if !pwConfig.DryRun {
				pwSuccess, pwMsg := callProfitwellAPI(jsonData)
				if !pwSuccess {
					var data = []string{fmt.Sprintf("%.0f", theMap["subscription_id"].(float64)), theMap["user_email"].(string), fmt.Sprintf("%.2f", lastAmt), fmt.Sprintf("%.2f", nextAmt), pwMsg}
					invalidLines++
					err := writeCsvLine(data, writer)
					checkError("Cannot write to file", err)
				}
			} else {

			}
		}

		defer writer.Flush()

		if pwConfig.DryRun {
			fmt.Println("This was a dry run; any errors are reported in the console, or written to 'bad_subscriptions.csv' file")
		}

		if invalidLines > 0 {
			fmt.Printf("\n")
			fmt.Println("You had some subscriptions we could not determine the MRR value of, or failed to process into Profitwell. Please consult the 'bad_subscriptions.csv' file in this directory")
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
	jsonData := map[string]string{"vendor_id": pwConfig.PaddleVendorID, "vendor_auth_code": pwConfig.PaddleAuthKey, "state": "active"}
	jsonValue, _ := json.Marshal(jsonData)
	response, err := http.Post(pwConfig.PaddleAPIURL+"/api/2.0/subscription/users", "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("The List Users API call request failed with error %s\n", err)
		log.Fatal("")
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		json.Unmarshal([]byte(data), &result)
	}
	return result
}

// DEPRECATED FUNCTION START //
func getSubscriptionCurrency(SubID int) string {
	currency := ""
	var result map[string]interface{}
	jsonData := map[string]interface{}{"vendor_id": pwConfig.PaddleVendorID, "vendor_auth_code": pwConfig.PaddleAuthKey, "subscription_id": SubID}
	jsonValue, _ := json.Marshal(jsonData)
	response, err := http.Post(pwConfig.PaddleAPIURL+"/api/2.0/subscription/payments", "application/json", bytes.NewBuffer(jsonValue))
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

// This needs to be a map of Paddle Plan ID to the name and interval of that plan
func getPlanInfo(planID int, planIDMap map[int]map[string]string) (string, string) {
	planIDName := planIDMap[planID]["name"]
	planInterval := planIDMap[planID]["interval"]
	return planInterval, planIDName
}

func callPaddleListPlans() map[int]map[string]string {
	finalResult := map[int]map[string]string{}
	var apiResults map[string]interface{}
	jsonData := map[string]string{"vendor_id": pwConfig.PaddleVendorID, "vendor_auth_code": pwConfig.PaddleAuthKey}
	jsonValue, _ := json.Marshal(jsonData)
	response, err := http.Post(pwConfig.PaddleAPIURL+"/api/2.0/subscription/plans", "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("The List Plans API call request failed with error %s\n", err)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		json.Unmarshal([]byte(data), &apiResults)
		if apiResults["success"].(bool) {
			for _, element := range apiResults["response"].([]interface{}) {
				theMap := element.(map[string]interface{})
				m := make(map[string]string)
				m["name"] = theMap["name"].(string)
				m["interval"] = theMap["billing_type"].(string)
				finalResult[int(theMap["id"].(float64))] = m
			}
		} else {
			log.Fatal("Paddle List Plans API faied, terminating script. Please make sure your config file is correct")
		}
	}
	return finalResult
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

func getMRRValue(theMap map[string]interface{}) (string, float64, float64, float64) {
	nextPayment := theMap["next_payment"].(map[string]interface{})
	lastPayment := theMap["last_payment"].(map[string]interface{})
	nextAmount := nextPayment["amount"].(float64)
	lastAmount := lastPayment["amount"].(float64)
	theCurr := nextPayment["currency"].(string)

	if nextAmount != lastAmount {
		theCurr = "INVALID"
	}

	return theCurr, ((nextAmount + lastAmount) / float64(2) * 100), lastAmount, nextAmount
}

func checkError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}

func callProfitwellAPI(apiData map[string]interface{}) (success bool, message string) {
	result := false
	returnMessage := ""

	jsonString, _ := json.Marshal(apiData)
	client := &http.Client{}

	req, _ := http.NewRequest("POST", "https://api.profitwell.com/v2/subscriptions/", bytes.NewBuffer(jsonString))
	req.Header.Set("Authorization", pwConfig.ProfitwellAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, respErr := client.Do(req)

	data, _ := ioutil.ReadAll(resp.Body)
	// Profitwell returns a 201 on success
	if respErr != nil || resp.StatusCode != 201 {
		fmt.Printf("The Profitwell API call failed %s\n", string(data))
		returnMessage = "PRF_FAIL_API"
		result = false
	} else {
		json.Unmarshal([]byte(data), &returnMessage)
		fmt.Printf(returnMessage)
		result = true
	}
	return result, returnMessage
}

func writeCsvLine(data []string, writer *csv.Writer) (err error) {
	theError := writer.Write(data)
	checkError("Cannot write to file", err)
	return theError
}
