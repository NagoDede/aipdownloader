package main

func main() {

	ConfData.LoadConfigurationFile("./aipdownloader.json")

	
	JapanAis.LoadJsonFile("./japan.json")
	JapanAis.Process()

}
