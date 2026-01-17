package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/gofiber/fiber/v2"
    "go.mau.fi/whatsmeow"
    "go.mau.fi/whatsmeow/store/sqlstore"
    "go.mau.fi/whatsmeow/types"
    waLog "go.mau.fi/whatsmeow/util/log"
    _ "github.com/mattn/go-sqlite3"
)

var (
    client *whatsmeow.Client
    qrCode string
    isConnected bool
    n8nWebhook = getEnv("N8N_WEBHOOK", "http://n8n-lg4s0cw48w4g08gwk0w4o8g8:5678/webhook/baileys")
)

func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}

func main() {
    // Setup database
    container, err := sqlstore.New("sqlite3", "file:whatsmeow.db?_foreign_keys=on", waLog.Stdout("Database", "INFO", true))
    if err != nil {
        log.Fatal(err)
    }

    deviceStore, err := container.GetFirstDevice()
    if err != nil {
        log.Fatal(err)
    }

    client = whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "INFO", true))

    // Event handlers
    client.AddEventHandler(func(evt interface{}) {
        switch v := evt.(type) {
        case *events.Message:
            handleMessage(v)
        case *events.QR:
            qrCode = v.Codes[len(v.Codes)-1]
            log.Println("📱 QR Code gerado! Acesse /qr")
        case *events.Connected:
            isConnected = true
            log.Println("✅ WhatsApp conectado!")
        case *events.Disconnected:
            isConnected = false
            log.Println("🔴 WhatsApp desconectado")
        }
    })

    // Conectar
    if client.Store.ID == nil {
        qrChan, _ := client.GetQRChannel(context.Background())
        err = client.Connect()
        if err != nil {
            log.Fatal(err)
        }
        
        for evt := range qrChan {
            if evt.Event == "code" {
                qrCode = evt.Code
                log.Println("📱 Novo QR Code disponível")
            }
        }
    } else {
        err = client.Connect()
        if err != nil {
            log.Fatal(err)
        }
    }

    // API REST
    app := fiber.New()

    app.Get("/", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{
            "service": "WhatsApp API - OLAMAESTRO",
            "status": map[string]bool{"connected": isConnected},
            "version": "2.0.0",
        })
    })

    app.Get("/qr", func(c *fiber.Ctx) error {
        if qrCode != "" {
            return c.JSON(fiber.Map{"qr": qrCode, "status": "scan_needed"})
        }
        if isConnected {
            return c.JSON(fiber.Map{"status": "connected"})
        }
        return c.JSON(fiber.Map{"status": "disconnected"})
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

        jid, err := types.ParseJID(body.Number + "@s.whatsapp.net")
        if err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Número inválido"})
        }

        _, err = client.SendMessage(context.Background(), jid, &waProto.Message{
            Conversation: &body.Message,
        })

        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }

        return c.JSON(fiber.Map{"success": true})
    })

    log.Println("🚀 WhatsApp API rodando na porta 3000")
    log.Fatal(app.Listen(":3000"))
}

func handleMessage(evt *events.Message) {
    if evt.Info.IsFromMe {
        return
    }

    msg := evt.Message.GetConversation()
    if msg == "" && evt.Message.ExtendedTextMessage != nil {
        msg = evt.Message.ExtendedTextMessage.GetText()
    }

    log.Printf("📩 Mensagem de %s: %s", evt.Info.Sender.User, msg)

    // Enviar para n8n
    payload := map[string]string{
        "from":    evt.Info.Sender.String(),
        "message": msg,
        "timestamp": time.Now().Format(time.RFC3339),
    }

    // HTTP request para n8n (implementar com http.Post)
}
