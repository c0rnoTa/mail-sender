// Sending Email Using Smtp in Golang
package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/smtp"
	"os"
	"time"
)

// Здесь все активные хэндлеры приложения
type MyApp struct {
	config   Config
	logLevel log.Level
	auth     smtp.Auth
}

// Main function
func main() {

	var App MyApp

	// Читаем конфиг
	App.GetConfigYaml("conf.yml")

	// Устанавливаем уровень журналирования событий приложения
	log.SetLevel(App.logLevel)

	// This is the message to send in the mail
	subject := "Письмо от mail-sender"
	msg := "Проверка доставки сообщения."

	// PlainAuth uses the given username and password to
	// authenticate to host and act as identity.
	// Usually identity should be the empty string,
	// to act as username.
	App.auth = smtp.PlainAuth("", App.config.Smtp.Username, App.config.Smtp.Password, App.config.Smtp.Server)
	log.Info("Use auth ", App.config.Smtp.Username, " at ", App.config.Smtp.Server, ":", App.config.Smtp.Port)

	for _, toAddr := range App.config.ToList {

		go App.sendEmail(toAddr, subject, msg)
	}

	time.Sleep(15 * time.Second)
	log.Info("Successfully sent mail to all user in toList")
}

func (a MyApp) sendEmail(toAddr string, subject string, message string) {
	log.Info("Sending e-mail to ", toAddr)
	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", a.config.Smtp.Server, a.config.Smtp.Port),
		a.auth,
		a.config.Smtp.FromAddr,
		[]string{toAddr},
		[]byte(fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\n\r\n%s\r\n", toAddr, a.config.Smtp.FromAddr, subject, message)),
	)
	log.Info("Message successfully send to ", toAddr)

	// handling the errors
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
