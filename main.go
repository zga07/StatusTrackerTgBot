package main

import (
	"log"
	"math/rand"
	"os"
	"statusTracker/internal/postgresDB"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gopkg.in/telebot.v3"
)

type AdminState struct {
	State     string
	TrackCode string
}

var adminStates = make(map[int64]*AdminState)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка подгрузки окружения: ", err)
	}

	adminID, _ := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)

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

	bot.Handle("/cancel", func(c telebot.Context) error {
		if c.Sender().ID == adminID {
			delete(adminStates, adminID)
			return c.Send("Действия отменены")
		}
		return nil
	})

	bot.Handle("/add", func(c telebot.Context) error {
		if c.Sender().ID != adminID {
			return c.Send("Ошибка, ваш ID не совпадает с ID администратора")
		}
		adminStates[adminID] = &AdminState{State: "waiting_desc"}
		return c.Send("Напишите описание заказа \nЛибо напишите /cancel для отмены")
	})

	bot.Handle("/update", func(c telebot.Context) error {
		if c.Sender().ID != adminID {
			return c.Send("Ошибка, ваш ID не совпадает с ID администратора")
		}
		adminStates[adminID] = &AdminState{State: "waiting_track"}
		return c.Send("Напишите трек-код заказа для обновления его статуса \nЛибо напишите /cancel для отмены")

	})

	bot.Handle(telebot.OnText, func(c telebot.Context) error {
		//TODO
	})

	bot.Start()
}

func generateTrackCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var randomizer = rand.New(rand.NewSource(time.Now().UnixNano()))
	byteSlice := make([]byte, 6)
	for i := range byteSlice {
		byteSlice[i] = charset[randomizer.Intn(len(charset))]
	}
	return string(byteSlice)
}
