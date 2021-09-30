package main

import (
	"database/sql"
	b64 "encoding/base64"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"net/smtp"
	"sync"
	"time"
)

func (App MyApp) RunSender(subject string, msg string) {
	var wg sync.WaitGroup
	// PlainAuth uses the given username and password to
	// authenticate to host and act as identity.
	// Usually identity should be the empty string,
	// to act as username.
	App.auth = smtp.PlainAuth("", App.config.Smtp.Username, App.config.Smtp.Password, App.config.Smtp.Server)
	log.Info("Use auth ", App.config.Smtp.Username, " at ", App.config.Smtp.Server, ":", App.config.Smtp.Port)
	log.Info("Start senders")
	sqlMailSend := fmt.Sprintf("INSERT INTO %s (receiver,send_status,send_time) VALUES (?,?,NOW()) ON DUPLICATE KEY UPDATE send_status=?, send_time=NOW()", App.config.DB.Table)
	DBStmt, err := App.db.Prepare(sqlMailSend)
	defer DBStmt.Close()
	if err != nil {
		log.Error("Could not register INSERT statement `sqlMailSend`: ", err)
		return
	}
	for _, toAddr := range App.config.ToList {
		wg.Add(1)
		go App.sendEmail(&wg, toAddr, subject, msg, DBStmt)
	}
	log.Info("Wait until all senders done")
	wg.Wait()
	log.Info("Successfully sent mail to all user in toList")
}

func (App *MyApp) sendEmail(wg *sync.WaitGroup, toAddr string, subject string, message string, DBStmt *sql.Stmt) {
	defer wg.Done()
	var status int
	log.Info("Sending e-mail to ", toAddr)
	body := fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: =?utf-8?B?%s?=\r\n\r\n%s\r\n", toAddr, App.config.Smtp.FromAddr, b64.StdEncoding.EncodeToString([]byte(subject)), message)
	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", App.config.Smtp.Server, App.config.Smtp.Port),
		App.auth,
		App.config.Smtp.FromAddr,
		[]string{toAddr},
		[]byte("MIME-version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n"+body),
	)

	// handling the errors
	if err != nil {
		log.Error(err)
		log.Error("Message failed to ", toAddr, " at ", time.Now())
		status = 0
	} else {
		log.Info("Message successfully send to ", toAddr, " at ", time.Now())
		status = 1
	}

	_, err = DBStmt.Exec(toAddr, status, status)
	if err != nil {
		log.Error("Could not execute UPDATE statement for `ad_task`: ", err)
	}
}
