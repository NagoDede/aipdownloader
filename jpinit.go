package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"

	"golang.org/x/net/publicsuffix"
)

var JapanAis JpData

type JpLoginFormData struct {
	FormName string `json:"formName"`
	Password string `json:"password"`
	UserID   string `json:"userID"`
}

type JpData struct {
	MainDataConfig
	LoginData        JpLoginFormData `json:"loginData"`
	LoginPage        string          `json:"loginPage"`
	AipIndexPageName string
}

type MainDataConfig struct {
	CountryDir       string `json:"countryDir"`
	MainAipPage      string
	MainAipActiveURL string
}

func (jpd *JpData) LoadJsonFile(path string) {
	// Open our jsonFile
	jsonFile, err := os.Open(path)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, jpd)
	if err != nil {
		fmt.Println("error:", err)
	}
}

func (jpd *JpData) Process() {
	client := jpd.initClient()

	//retrieve the  AIP document and the active one
	var aipDocsList AipDocs
	fmt.Println("Retrieve the AIP Documents")
	aipDocsList = getAipDocuments(&client)
	fmt.Println("Retrieve the Active Document")
	activeAipDoc := aipDocsList.getActiveAipDoc()
	activeAipDoc.countryCode = jpd.CountryDir
	fmt.Println("   Active Document Effective Date:" + activeAipDoc.effectiveDate.Format("02-Jan-2006") +
		" Publication Date: " + activeAipDoc.publicationDate.Format("02-Jan-2006"))
	fmt.Println("   " + activeAipDoc.fullURLDir)
	fmt.Println("Retrieve the Airports List")
	activeAipDoc.GetAirports(&client)
	fmt.Println("Number of identified airports: ")
	fmt.Println("Download the Airports Data")
	activeAipDoc.DownloadAllAiportsData(&client)
	activeAipDoc.SentToFtp()
}

/**
 * initClient inits an http client to connect to the website  by sending the
 * data to the formular.
 */
func (jpd *JpData) initClient() http.Client {
	frmData := jpd.LoginData
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}

	var client = http.Client{Jar: jar}

	v := url.Values{"formName": {frmData.FormName},
		"password": {frmData.Password},
		"userID":   {frmData.UserID}}

	//connect to the website
	resp, err := client.PostForm(JapanAis.LoginPage, v)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	return client
}
