package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/garyburd/go-oauth/oauth"
)

const (
	XeroRequestTokenURL   = "https://api.xero.com/oauth/RequestToken"
	XeroAuthorizeTokenURL = "https://api.xero.com/oauth/Authorize"
	XeroAccessTokenURL    = "https://api.xero.com/oauth/AccessToken"
	XeroApiEndpointURL    = "https://api.xero.com/api.xro/2.0/"
)

func main() {
	//parse your private key (see http://developer.xero.com/documentation/getting-started/private-applications/)
	path := "../keys/xero_paddle.pem"
	pemBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	block, extra := pem.Decode(pemBytes)
	if block == nil || len(extra) > 0 {
		log.Fatal("Failed to decode PEM")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}
	//create the oauth1 client using this key (consumer secret not required)
	client := oauth.Client{
		Credentials:                   oauth.Credentials{Token: "D9WY2YJOVKUUDCKNYTRNICUDGWMWFY"},
		TemporaryCredentialRequestURI: XeroRequestTokenURL,
		ResourceOwnerAuthorizationURI: XeroAuthorizeTokenURL,
		TokenRequestURI:               XeroAccessTokenURL,
		SignatureMethod:               oauth.RSASHA1,
		PrivateKey:                    privateKey,
		Header:                        http.Header{"Accept": []string{"application/json"}}, //noone likes XML
	}
	//list all invoices
	response, err := client.Get(http.DefaultClient, &client.Credentials, XeroApiEndpointURL+"invoices/", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	//show response
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(contents))
}
