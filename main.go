package main

import (
	"fmt"
	generic "aiploader/generic"
)

func main() {

	//ConnectMongo()

	generic.ConfData = configurationDataStruct{}
	JapanAis = JpData{}
	fmt.Println("AIP Downloader is starting")
	generic.ConfData.LoadConfigurationFile("./aipdownloader.json")
	fmt.Printf("Data will be stored in %s \n", ConfData.MainLocalDir)

	JapanAis.LoadJsonFile("./japan.json")
	JapanAis.Process()
	fmt.Println("AIP Downloader - End of process")


}
