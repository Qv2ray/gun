package main

import "log"

func main() {
	log.Println("gun is running in SIP003 mode.")
	//options := map[string]string{}
	//_, sip003 := os.LookupEnv("SS_LOCAL_HOST")
	//if sip003 {
	//	log.Println("start as SIP003 plugin, command line parameter is ignored.")
	//	*LocalAddr = fmt.Sprintf("%s:%s", os.Getenv("SS_LOCAL_HOST"), os.Getenv("SS_LOCAL_PORT"))
	//	*RemoteAddr = fmt.Sprintf("%s:%s", os.Getenv("SS_REMOTE_HOST"), os.Getenv("SS_REMOTE_PORT"))
	//	optionArr := strings.Split(os.Getenv("SS_PLUGIN_OPTIONS"), ";")
	//	if optionArr[0] != "server" {
	//		*RunMode = "client"
	//	} else {
	//		*RunMode = "server"
	//	}
	//	optionArr = optionArr[1:]
	//	for _, s := range optionArr {
	//		kv := strings.Split(s, "=")
	//		if len(kv) != 2 {
	//			log.Println(s)
	//			log.Fatalln("Can't parse plugin option")
	//		}
	//		options[kv[0]] = options[kv[1]]
	//	}
	//
	//	*ServerName = readOption(options, "sni")
	//	*CertPath = readOption(options, "cert")
	//	*KeyPath = readOption(options, "key")
	//} else {
	//	flag.Parse()
	//}

}
