package main

import (
	"fmt"
)

func main() {

	//ConnectMongo()

	ConfData = configurationDataStruct{}
	JapanAis = JpData{}
	fmt.Println("AIP Downloader is starting")
	ConfData.LoadConfigurationFile("./aipdownloader.json")
	fmt.Printf("Data will be stored in %s \n", ConfData.MainLocalDir)

	JapanAis.LoadJsonFile("./japan.json")
	JapanAis.Process()
	fmt.Println("AIP Downloader - End of process")


}
