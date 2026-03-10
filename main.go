package main

import (
	"fmt"
	"log"
	"os"
	"statusTracker/internal/postgresDB"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gopkg.in/telebot.v3"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка подгрузки окружения: ", err)
	}

	db := postgresDB.InitDB()
	defer db.Close()
	postgresDB.CreateTable(db)

	pref := telebot.Settings{
		Token:  os.Getenv("BOT_TOKEN"),
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		log.Fatal("Ошибка создания бота: ", err)
	}

	bot.Handle(telebot.OnText, func(c telebot.Context) error {
		status, err := postgresDB.GetOrderStatus(db, c.Text())
		if err != nil {
			log.Println("Ошибка получения статуса: ", err)
			return c.Send("Заказ с таким номером не найден. Проверьте правильность кода.")
		}
		postgresDB.RegisterUser(db, c.Text(), c.Sender().ID)
		return c.Send(fmt.Sprintf("Статус вашего заказа: %s", status))
	})

	bot.Start()
}
