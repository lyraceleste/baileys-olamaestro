package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	waProto "go.mau.fi/whatsmeow/binary/proto"
)

var (
	client      *whatsmeow.Client
	qrCode      string
	isConnected bool
	n8nWebhook  = getEnv("N8N_WEBHOOK", "http://n8n-lg4s0cw48w4g08gwk0w4o8g8:5678/webhook/whatsapp")
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	dbLog := waLog.Stdout("Database", "INFO", true)
	container, err := sqlstore.New("sqlite3", "file:whatsmeow.db?_foreign_keys=on", dbLog)
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		log.Fatal("Error getting device:", err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)

	client.AddEventHandler(handleEvent)

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			log.Fatal("Error connecting:", err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				qrCode = evt.Code
				log.Println("📱 QR Code gerado! Acesse /qr para escanear")
			} else {
				log.Println("QR channel event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			log.Fatal("Error connecting:", err)
		}
	}

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service": "WhatsApp API - OLAMAESTRO",
			"status":  isConnected,
			"version": "2.0.0-whatsmeow",
		})
	})

	app.Get("/qr", func(c *fiber.Ctx) error {
		if qrCode != "" {
			return c.JSON(fiber.Map{
				"qr":     qrCode,
				"status": "scan_needed",
			})
		}
		if isConnected {
			return c.JSON(fiber.Map{
				"status": "connected",
			})
		}
		return c.JSON(fiber.Map{
			"status": "disconnected",
		})
	})

	app.Get("/status", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"connected": isConnected,
			"timestamp": time.Now(),
		})
	})

	app.Post("/send", func(c *fiber.Ctx) error {
		var body struct {
			Number  string `json:"number"`
			Message string `json:"message"`
		}

		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
		}

		if !isConnected {
			return c.Status(503).JSON(fiber.Map{"error": "WhatsApp não conectado"})
		}

		jid := body.Number
		if !contains(jid, "@") {
			jid = jid + "@s.whatsapp.net"
		}

		msg := &waProto.Message{
			Conversation: &body.Message,
		}

		_, err := client.SendMessage(context.Background(), parseJID(jid), msg)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"success": true, "to": jid})
	})

	log.Println("🚀 WhatsApp API rodando na porta 3000")
	log.Println("📡 Webhook n8n:", n8nWebhook)
	log.Fatal(app.Listen(":3000"))
}

func handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if !v.Info.IsFromMe && v.Message != nil {
			msg := v.Message.GetConversation()
			if msg == "" && v.Message.ExtendedTextMessage != nil {
				msg = v.Message.ExtendedTextMessage.GetText()
			}
			log.Printf("📩 Mensagem de %s: %s", v.Info.Sender.User, msg)
		}

	case *events.Connected:
		isConnected = true
		qrCode = ""
		log.Println("✅ WhatsApp conectado com sucesso!")

	case *events.Disconnected:
		isConnected = false
		log.Println("🔴 WhatsApp desconectado")
	}
}

func parseJID(jid string) types.JID {
	parsed, _ := types.ParseJID(jid)
	return parsed
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
