package main

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	godotenv "github.com/joho/godotenv"
	cron "github.com/robfig/cron/v3"
)

var (
	// Map zur Speicherung, ob ein Nutzer "gemacht" geschrieben hat.
	userDone = make(map[int64]bool)
	mu       sync.Mutex

	// Globale Variablen für die beiden Nutzer
	user1ID int64
	user2ID int64
)

// resetUserDone setzt den Status beider Nutzer auf "nicht gemacht".
func resetUserDone() {
	log.Println("Reseted User Status")
	mu.Lock()
	defer mu.Unlock()
	userDone[user1ID] = false
	userDone[user2ID] = false
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal(".env file not found!")
	}
	// Einlesen der Konfiguration aus Umgebungsvariablen
	botToken := os.Getenv("BOT_AUTHKEY")
	if botToken == "" {
		log.Fatal("BOT_TOKEN nicht gesetzt")
	}
	groupIDStr := os.Getenv("GROUPID")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		log.Fatal("Ungültige GROUP_CHAT_ID")
	}
	user1IDStr := os.Getenv("USER1ID")
	user1ID, err = strconv.ParseInt(user1IDStr, 10, 64)
	if err != nil {
		log.Fatal("Ungültige USER1_ID")
	}
	user2IDStr := os.Getenv("USER2ID")
	user2ID, err = strconv.ParseInt(user2IDStr, 10, 64)
	if err != nil {
		log.Fatal("Ungültige USER2_ID")
	}

	// Initialisierung des Telegram Bots
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Erfolgreich verbunden als %s", bot.Self.UserName)

	// Initialer Zustand für den Tag setzen
	resetUserDone()

	// Zeitzone "Europe/Berlin" laden
	berlinLoc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		log.Fatalf("Fehler beim Laden der Zeitzone: %v", err)
	}
	msgstartstr := "Bin Online"
	msgstart := tgbotapi.NewMessage(groupID, msgstartstr)

	if _, err := bot.Send(msgstart); err != nil {
		log.Printf("Fehler beim Senden der Start Nachricht!")
	} else {
		log.Printf("Start Nachricht gesendet!")
	}

	// Cron-Job initialisieren – der Job wird jeden Tag um 20:00 (Berliner Zeit) ausgeführt.
	c := cron.New(cron.WithLocation(berlinLoc))
	_, err = c.AddFunc("0 20 * * *", func() {
		// Status der Nutzer abfragen
		mu.Lock()
		user1Status := userDone[user1ID]
		user2Status := userDone[user2ID]
		mu.Unlock()

		// Falls beide Nutzer nicht "gemacht" geschrieben haben, sende eine Benachrichtigung
		if !user1Status && !user2Status {
			msgText := "Leute bitte einmal die Blumen gießen, was soll das?"
			msg := tgbotapi.NewMessage(groupID, msgText)
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Fehler beim Senden der Nachricht: %v", err)
			} else {
				log.Println("Erinnerung erfolgreich versendet.")
			}
		} else {
			log.Println("Mindestens ein Nutzer hat 'gemacht' geschrieben – keine Erinnerung gesendet.")
		}

		// Zustand für den nächsten Tag zurücksetzen
		resetUserDone()
	})
	if err != nil {
		log.Fatalf("Fehler beim Planen des Cron-Jobs: %v", err)
	}
	c.Start()
	defer c.Stop()

	// Telegram Updates abfragen
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 90
	updates := bot.GetUpdatesChan(u)

	// Endlosschleife: Nachrichten verarbeiten
	for update := range updates {
		log.Printf("update")
		if update.Message == nil {
			continue
		}

		// Nur Nachrichten aus der definierten Gruppe verarbeiten
		if update.Message.Chat.ID != groupID {
			continue
		}

		// Prüfen, ob der Text exakt "gemacht" lautet
		if update.Message.Text == "/gemacht" {
			uid := update.Message.From.ID
			// Nur die beiden beobachteten Nutzer werden berücksichtigt
			if uid == user1ID || uid == user2ID {
				mu.Lock()
				userDone[uid] = true
				mu.Unlock()
				log.Printf("Nutzer %d hat 'gemacht' geschrieben.", uid)
			}
		}
	}
}
