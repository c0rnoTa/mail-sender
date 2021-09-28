package main

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// Структура конфигурационного файла
type Config struct {
	Smtp struct {
		FromAddr string `yaml:"from"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Server   string `yaml:"server"`
		Port     int    `yaml:"port"`
	} `yaml:"smtp"`
	ToList   []string `yaml:"toList,flow"`
	LogLevel string   `yaml:"loglevel"`
}

// Читаем конфиг и устанавливаем параметры приложения
func (a *MyApp) GetConfigYaml(filename string) {
	log.Info("Reading config ", filename)

	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}
	err = yaml.Unmarshal(yamlFile, &a.config)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}
	a.logLevel = setLogLevel(a.config.LogLevel)
}

// Устанавливаем уровень журналирования событий в приложении
func setLogLevel(confLogLevel string) log.Level {
	var result log.Level
	switch confLogLevel {
	case "debug":
		result = log.DebugLevel
	case "info":
		result = log.InfoLevel
	case "warn":
		result = log.WarnLevel
	case "error":
		result = log.ErrorLevel
	case "fatal":
		result = log.FatalLevel
	default:
		result = log.InfoLevel
	}

	log.Info("Application logging level: ", result)

	return result
}
