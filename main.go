// Sending Email Using Smtp in Golang
package main

import (
	b64 "encoding/base64"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/smtp"
	"sync"
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

	var wg sync.WaitGroup

	// Читаем конфиг
	App.GetConfigYaml("conf.yml")

	// Устанавливаем уровень журналирования событий приложения
	log.SetLevel(App.logLevel)

	// This is the message to send in the mail
	subject := "Проверка отправки сообщения"
	msg := "<html><h1>Привет!</h1>Это проверка доставки сообщения.<br>С уважением,<br>Ваш скрипт!</html>"

	// PlainAuth uses the given username and password to
	// authenticate to host and act as identity.
	// Usually identity should be the empty string,
	// to act as username.
	App.auth = smtp.PlainAuth("", App.config.Smtp.Username, App.config.Smtp.Password, App.config.Smtp.Server)
	log.Info("Use auth ", App.config.Smtp.Username, " at ", App.config.Smtp.Server, ":", App.config.Smtp.Port)
	log.Info("Start senders")
	for _, toAddr := range App.config.ToList {
		wg.Add(1)
		go App.sendEmail(&wg, toAddr, subject, msg)
	}
	log.Info("Wait until all senders done")
	wg.Wait()
	log.Info("Successfully sent mail to all user in toList")
}

func (a MyApp) sendEmail(wg *sync.WaitGroup, toAddr string, subject string, message string) {
	defer wg.Done()
	log.Info("Sending e-mail to ", toAddr)
	body := fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: =?utf-8?B?%s?=\r\n\r\n%s\r\n", toAddr, a.config.Smtp.FromAddr, b64.StdEncoding.EncodeToString([]byte(subject)), message)
	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", a.config.Smtp.Server, a.config.Smtp.Port),
		a.auth,
		a.config.Smtp.FromAddr,
		[]string{toAddr},
		[]byte("MIME-version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n"+body),
	)

	// handling the errors
	if err != nil {
		log.Error(err)
		log.Error("Message failed to ", toAddr, " at ", time.Now())
		return
	}
	log.Info("Message successfully send to ", toAddr, " at ", time.Now())
}
