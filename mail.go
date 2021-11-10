package main

import (
	"database/sql"
	b64 "encoding/base64"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	_ "github.com/go-sql-driver/mysql"
	"github.com/m1/gospin"
	log "github.com/sirupsen/logrus"
	"math/rand"
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
	// Добавляем текстовый спинер
	spinner := gospin.New(nil)

	if App.config.Smtp.Warm {
		log.Info("Starting infinity loop for SMTP warmup sending messages to random toList")
		rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
		for range time.NewTicker(time.Duration(int64(rand.Intn(App.config.Smtp.WarmFreq))+5) * time.Second).C {
			// Берём случайный e-mail из списка получателей
			toAddr := App.config.ToList[rand.Intn(len(App.config.ToList))]
			// Ратируем заголовок сообщения
			spinSubject, err := spinner.Spin(subject)
			if err != nil {
				log.Error("Could not spin Subject: ", err)
				spinSubject = subject
			}
			// Ратируем текст сообщения
			spinMsg, err := spinner.Spin(msg)
			if err != nil {
				log.Error("Could not spin Message: ", err)
				spinMsg = msg
			}
			wg.Add(1)
			go App.sendEmail(&wg, toAddr, spinSubject, spinMsg, DBStmt)
		}

	}

	for _, toAddr := range App.config.ToList {
		// Ратируем заголовок сообщения
		spinSubject, err := spinner.Spin(subject)
		if err != nil {
			log.Error("Could not spin Subject: ", err)
			spinSubject = subject
		}
		// Ратируем текст сообщения
		spinMsg, err := spinner.Spin(msg)
		if err != nil {
			log.Error("Could not spin Message: ", err)
			spinMsg = msg
		}
		wg.Add(1)
		go App.sendEmail(&wg, toAddr, spinSubject, spinMsg, DBStmt)
	}
	log.Info("Wait until all senders done")
	wg.Wait()
	log.Info("Successfully sent mail to all user in toList")
}

func (App *MyApp) sendEmail(wg *sync.WaitGroup, toAddr string, subject string, message string, DBStmt *sql.Stmt) {
	defer wg.Done()
	var status int
	log.Info("Sending e-mail to ", toAddr)
	dateHeader := time.Now().Format(time.RFC1123Z)
	body := fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: =?utf-8?B?%s?=\r\n\r\n%s\r\n", toAddr, App.config.Smtp.FromAddr, b64.StdEncoding.EncodeToString([]byte(subject)), message)
	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", App.config.Smtp.Server, App.config.Smtp.Port),
		App.auth,
		App.config.Smtp.FromAddr,
		[]string{toAddr},
		[]byte("MIME-version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\nDate: "+dateHeader+"\r\n"+body),
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
		log.Error("Could not execute UPDATE statement for `sender`: ", err)
	}
}

// Запуск получения почты
func (a *MyApp) RunReceiver(i int) {

	log.Debug("Start mail receiver", i, ":", a.config.Imap.Receivers[i].Mail)

	// Подключаемся к серверу IMAP
	log.Debug("Receiver [", a.config.Imap.Receivers[i].Mail, "] connecting to imap://", a.config.Imap.Receivers[i].Server)
	tmpClinet, err := client.DialTLS(a.config.Imap.Receivers[i].Server, nil)
	if err != nil {
		log.Error("IMAP TLS connection returned error: ", err)
		return
	}

	a.imapClient[i] = tmpClinet

	log.Debug("Receiver [", a.config.Imap.Receivers[i].Mail, "] IMAP Connected")

	// Don't forget to logout from IMAP server
	defer func() {
		err = a.imapClient[i].Logout()
		if err != nil {
			log.Error("Receiver [", a.config.Imap.Receivers[i].Mail, "] IMAP Logout error: ", err)
		}
		err = a.imapClient[i].Terminate()
		if err != nil {
			log.Error("Receiver [", a.config.Imap.Receivers[i].Mail, "] IMAP Terminate error: ", err)
		}
	}()

	// Login
	err = a.imapClient[i].Login(a.config.Imap.Receivers[i].Username, a.config.Imap.Receivers[i].Password)
	if err != nil {
		log.Error("Receiver [", a.config.Imap.Receivers[i].Mail, "] IMAP login returned error: ", err)
		return
	}
	log.Info("Receiver [", a.config.Imap.Receivers[i].Mail, "] IMAP Logged in as ", a.config.Imap.Receivers[i].Username)

	// Отображаем списки папок на сервере
	mailboxes := make(chan *imap.MailboxInfo, 10)

	err = a.imapClient[i].List("", "*", mailboxes)
	if err != nil {
		log.Error("Receiver [", a.config.Imap.Receivers[i].Mail, "] IMAP Could not list mailboxes ", err)
	}
	for m := range mailboxes {
		log.Debug("Receiver [", a.config.Imap.Receivers[i].Mail, "] IMAP Mailbox: "+m.Name)
	}

	// Выбираем папку INBOX на почтовом сервере
	log.Infof("Receiver [%s] Select %s mailbox", a.config.Imap.Receivers[i].Mail, a.config.Imap.Folder)
	_, err = a.imapClient[i].Select(a.config.Imap.Folder, false)
	if err != nil {
		log.Error("Receiver [", a.config.Imap.Receivers[i].Mail, "] IMAP Mailbox folder select returned error: ", err)
		return
	}

	// Дальше в бесконечном цикле ищем новые сообщения и сохраняем время получения письма
	a.ReadNewMail(i)
	return
}

// ReadNewMail Уведомляем о новых письмах
func (a *MyApp) ReadNewMail(i int) {
	log.Debug("Receiver [", a.config.Imap.Receivers[i].Mail, "] mailbox pooler starting")
	/*
		// Глубокий дэбаг
		if a.config.LogLevel == "debug" {
			a.imapClient[i].SetDebug(log.StandardLogger().Writer())
		}
	*/
	// Установка критериев отбора писем в папке
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Seen"}

	// Регистрируем statement для отметки полученных писем
	sqlMailReceive := fmt.Sprintf("INSERT INTO %s (receiver,receive_status,receive_time) VALUES (?,?,NOW()) ON DUPLICATE KEY UPDATE receive_status=?, receive_time=NOW()", a.config.DB.Table)
	DBStmt, err := a.db.Prepare(sqlMailReceive)
	defer DBStmt.Close()
	if err != nil {
		log.Error("Could not register INSERT statement `sqlMailReceive`: ", err)
		return
	}

	// В бесконечном цикле проверяем почтовый ящик на новые письма
	for range time.NewTicker(time.Duration(a.getTimeout(i)) * time.Second).C {
		// Проверяем новые письма
		err := a.imapClient[i].Noop()
		if err != nil {
			log.Error("Receiver [", a.config.Imap.Receivers[i].Mail, "] IMAP Mailbox refresh returned error: ", err)
			log.Debug("Receiver [", a.config.Imap.Receivers[i].Mail, "] IMAP connection status: ", a.imapClient[i].State())
			return
		}

		// Получаем UID-ы непрочитанных писем
		log.Debug("Receiver [", a.config.Imap.Receivers[i].Mail, "] Search Unseen messages... ")
		uids, err := a.imapClient[i].Search(criteria)
		if err != nil {
			log.Error("Receiver [", a.config.Imap.Receivers[i].Mail, "] IMAP mail search returned error: ", err)
			return
		}
		// Если UID-ов нет, то новых писем нет
		if len(uids) == 0 {
			log.Info("Receiver [", a.config.Imap.Receivers[i].Mail, "] No new messages yet.")
			continue
		}

		log.Info("Receiver [", a.config.Imap.Receivers[i].Mail, "] There are ", len(uids), " new messages")
		seqset := new(imap.SeqSet)
		// Читаем письма по одному
		seqset.AddNum(uids[0])

		// Инициализируем канал обработки полученных писем
		messages := make(chan *imap.Message, 10)
		// Отдельным потоком отгружаем найденные письма в канал
		go func() {
			log.Debug("Receiver [", a.config.Imap.Receivers[i].Mail, "] Fetching new messages... ")
			err := a.imapClient[i].Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
			if err != nil {
				log.Error("Receiver [", a.config.Imap.Receivers[i].Mail, "] IMAP mail fetch error: ", err)
			}
		}()

		// Обрабатываем каждое новое письмо
		for msg := range messages {
			log.Info("Receiver [", a.config.Imap.Receivers[i].Mail, "] * "+msg.Envelope.Subject)
			// Сохраняем информацию о получении письма
			_, err = DBStmt.Exec(a.config.Imap.Receivers[i].Mail, 1, 1)
			if err != nil {
				log.Error("Could not execute UPDATE statement for `receiver`: ", err)
			}
			// Помечаем письмо как прочитанное
			curSeq := new(imap.SeqSet)
			curSeq.AddNum(msg.SeqNum)
			markFlag := imap.SeenFlag
			// или удаляем письмо
			if a.config.Imap.DeleteMessages {
				markFlag = imap.DeletedFlag
			}
			log.Debug("Receiver [", a.config.Imap.Receivers[i].Mail, "] Mark message '", msg.Envelope.Subject, "' as ", markFlag)
			err := a.imapClient[i].Store(curSeq, imap.FormatFlagsOp(imap.AddFlags, true), []interface{}{markFlag}, nil)
			if err != nil {
				log.Error("Receiver [", a.config.Imap.Receivers[i].Mail, "] IMAP mark mail as ", markFlag, " error: ", err)
			}
			log.Debug("Receiver [", a.config.Imap.Receivers[i].Mail, "] Message '", msg.Envelope.Subject, "' done.")
		}
	}

}

func (a *MyApp) getTimeout(i int) int64 {
	timeout := a.config.Imap.RefreshTimeout
	if a.config.Imap.RefreshRandomize {
		rand.Seed(time.Now().UnixNano())
		timeout = timeout*int64(rand.Intn(60)) + 1
	}
	log.Info("Receiver [", a.config.Imap.Receivers[i].Mail, "] Will check ", a.config.Imap.Folder, " mail every ", timeout, " seconds")
	return timeout
}
