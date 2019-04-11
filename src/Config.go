package config

import (
	"strconv"
	"os"
)

type Config struct {
	useHttps bool
	port int
}

func GetConfig() Config {
	newConf := Config{}
	newConf.useHttps = false
	if(os.Getenv("USE_HTTPS") == "YES") {
		newConf.useHttps = true
	}	
	if(os.Getenv("PORT") != "") {
		newConf.port, _ = strconv.Atoi(os.Getenv("PORT"))
	}
	return newConf
}

