// Sending Email Using Smtp in Golang
package main

import (
	"database/sql"
	"flag"
	"github.com/emersion/go-imap/client"
	log "github.com/sirupsen/logrus"
	"net/smtp"
	"os"
	"time"
)

// Здесь все активные хэндлеры приложения
type MyApp struct {
	config     Config
	logLevel   log.Level
	auth       smtp.Auth
	db         *sql.DB
	imapClient []*client.Client
}

// Main function
func main() {

	var err error

	var App MyApp

	// Читаем конфиг
	configPath := flag.String("c", "", "Path to conf.yml")
	flag.Parse()
	if *configPath == "" {
		log.Info("Path to config.yaml not defined. Use -c option")
		*configPath = "conf.yml"
	}
	App.GetConfigYaml(*configPath)

	// Устанавливаем уровень журналирования событий приложения
	log.SetLevel(App.logLevel)
	// Set logger channel
	log.SetOutput(os.Stdout)

	// This is the message to send in the mail
	subject := "Проверка отправки сообщения"
	msg := "<html><h1>Привет!</h1>Это проверка доставки сообщения.<br>С уважением,<br>Ваш скрипт!</html>"

	// Подключение к базе данных
	err = App.ConncetDB()
	if err != nil {
		os.Exit(1)
	}

	if App.config.Imap.Enable {
		// Запускаем получение почты
		for i, _ := range App.config.Imap.Receivers {
			App.imapClient = append(App.imapClient, nil)
			i := i
			go func() {
				// Будет переподключаться, если разорвалось соединение
				for {
					App.RunReceiver(i)
					log.Warn("Receiver [", App.config.Imap.Receivers[i].Mail, "] will reconnect to imap://", App.config.Imap.Receivers[i].Server)
					time.Sleep(10 * time.Second)
				}
			}()
		}
		log.Info("All receivers are started")
	}

	if App.config.Smtp.Enable {
		// Запускаем отправку писем
		App.RunSender(subject, msg)
	}

	// TODO Сюда можно добавить проверку статуса подключений и их восстановление в цикле
	ch := make(chan bool)
	<-ch
}
