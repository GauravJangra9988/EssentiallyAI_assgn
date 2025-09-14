package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	URL string
	DB_Username string
	DB_Password string
	DB_Name string
}

var AppConfig *Config 

func LoadConfig(){
	
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env")
	}

	AppConfig = &Config {
		URL: os.Getenv("URL"),
		DB_Username: os.Getenv("DB_Username"),
		DB_Password: os.Getenv("DB_Password"),
		DB_Name: os.Getenv("DB_Name"),
	}
}