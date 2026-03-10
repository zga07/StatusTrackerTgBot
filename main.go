package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"statusTracker/internal/postgresDB"
	"strconv"
	"strings"
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

	bot.Handle("/start", func(c telebot.Context) error {
		return c.Send("Здравствуйте, для отслеживания заказа введите трек-код отправленный вам по почте")
	})

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
		text := c.Text()
		if c.Sender().ID == adminID {
			memory, exists := adminStates[adminID]
			if exists {
				switch memory.State {
				case "waiting_desc":
					var trackCode string
					var err error
					for range 5 {
						trackCode = generateTrackCode()
						err = postgresDB.AddOrder(db, trackCode, text)
						if err == nil {
							break
						}
					}
					if err != nil {
						delete(adminStates, adminID)
						return c.Send("Ошибка коллизий (wtf), попробуйте ещё раз")
					}
					delete(adminStates, adminID)
					return c.Send(fmt.Sprintf("Заказ создан!\nОписание: %s\nТрек-код: %s", text, trackCode))
				case "waiting_track":
					memory.TrackCode = strings.TrimSpace(text)
					memory.State = "waiting_status"
					return c.Send(fmt.Sprintf("Заказ: %s\nНапишите новый статус для заказа:", memory.TrackCode))
				case "waiting_status":
					var status string = text
					chatID, err := postgresDB.UpdateOrderStatus(db, memory.TrackCode, status)
					delete(adminStates, adminID)
					if err != nil {
						return c.Send("Ошибка, заказ с таким трек-кодом не найден")
					}

					if chatID != 0 {
						bot.Send(&telebot.User{ID: chatID}, fmt.Sprintf("Статус вашего заказа %s обновился:\n%s", memory.TrackCode, status))
					}
					return c.Send(fmt.Sprintf("Статус заказа %s успешно изменен на %s", memory.TrackCode, status))
				}
			}
		}
		trackCode := strings.TrimSpace(text)

		status, err := postgresDB.GetOrderStatus(db, trackCode)
		if err != nil {
			return c.Send("Заказ с таким трек-кодом не найден")
		}

		history, _ := postgresDB.GetOrderHistory(db, trackCode)
		var result string
		result = "История статусов:\n" + strings.Join(history, "\n")

		postgresDB.RegisterUser(db, trackCode, c.Sender().ID)

		message := fmt.Sprintf("Заказ: %s\nТекущий статус: %s\n%s", trackCode, status, result)
		return c.Send(message)
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
