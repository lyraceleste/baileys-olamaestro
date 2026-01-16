const { default: makeWASocket, DisconnectReason, useMultiFileAuthState } = require('@whiskeysockets/baileys');
const express = require('express');
const axios = require('axios');
const app = express();

app.use(express.json());

let sock;
let qrCode = null;
let isConnected = false;

const N8N_WEBHOOK = process.env.N8N_WEBHOOK || 'http://n8n-lg4s0cw48w4g08gwk0w4o8g8:5678/webhook/baileys';

async function connectToWhatsApp() {
    const { state, saveCreds } = await useMultiFileAuthState('auth_info');
    
    sock = makeWASocket({
        auth: state,
        printQRInTerminal: true
    });

    sock.ev.on('creds.update', saveCreds);

    sock.ev.on('connection.update', (update) => {
        const { connection, lastDisconnect, qr } = update;
        
        if (qr) {
            qrCode = qr;
            console.log('📱 QR Code disponível em /qr');
        }
        
        if (connection === 'close') {
            const shouldReconnect = lastDisconnect?.error?.output?.statusCode !== DisconnectReason.loggedOut;
            console.log('🔴 Conexão fechada. Reconectando:', shouldReconnect);
            if (shouldReconnect) {
                setTimeout(connectToWhatsApp, 3000);
            }
            isConnected = false;
        } else if (connection === 'open') {
            console.log('✅ WhatsApp conectado!');
            isConnected = true;
            qrCode = null;
        }
    });

    sock.ev.on('messages.upsert', async ({ messages, type }) => {
        if (type === 'notify') {
            for (const msg of messages) {
                if (!msg.key.fromMe && msg.message) {
                    const from = msg.key.remoteJid;
                    const text = msg.message.conversation || 
                                msg.message.extendedTextMessage?.text || '';
                    
                    const payload = {
                        from: from,
                        message: text,
                        timestamp: new Date().toISOString()
                    };
                    
                    console.log(`📩 ${from}: ${text}`);
                    
                    try {
                        await axios.post(N8N_WEBHOOK, payload);
                    } catch (error) {
                        console.error('Erro webhook:', error.message);
                    }
                }
            }
        }
    });
}

app.get('/', (req, res) => {
    res.json({ service: 'Baileys OLAMAESTRO', status: isConnected ? 'connected' : 'disconnected' });
});

app.get('/qr', (req, res) => {
    if (qrCode) {
        res.json({ qr: qrCode, status: 'scan_needed' });
    } else if (isConnected) {
        res.json({ status: 'connected' });
    } else {
        res.json({ status: 'disconnected' });
    }
});

app.get('/status', (req, res) => {
    res.json({ connected: isConnected, timestamp: new Date().toISOString() });
});

app.post('/send', async (req, res) => {
    const { number, message } = req.body;
    
    if (!isConnected) {
        return res.status(503).json({ error: 'WhatsApp desconectado' });
    }
    
    try {
        const jid = number.includes('@') ? number : `${number}@s.whatsapp.net`;
        await sock.sendMessage(jid, { text: message });
        res.json({ success: true });
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

const PORT = 3000;
app.listen(PORT, '0.0.0.0', () => {
    console.log(`🚀 Baileys rodando porta ${PORT}`);
    connectToWhatsApp();
});
```

4. **Commit**

---

**Pronto! Repositório criado com 4 arquivos.**

---

## 🔧 PASSO 3: DEPLOY NO COOLIFY (5min)

### **No Coolify**:

1. **+ New** → **Resource**
2. Seleciona: **Public Repository**
3. **Repository URL**: `https://github.com/SEU_USER/baileys-olamaestro`
4. **Branch**: `main`
5. **Build Pack**: Docker Compose
6. **Network**: `coolify` (selecionar no dropdown)

---

### **Configurações importantes**:

**General**:
- **Name**: `baileys-whatsapp`
- **Port**: `3000` (Exposed Ports)

**Domains** (opcional agora, pode fazer depois):
- Add domain: `baileys.olamaestro.com`

**Environment Variables**:
- `N8N_WEBHOOK` = `http://n8n-lg4s0cw48w4g08gwk0w4o8g8:5678/webhook/baileys`

---

### **Deploy**:

1. Botão **Deploy** (canto superior direito)
2. Aguardar build (2-3 min)
3. Ver logs em tempo real

---

## 📱 PASSO 4: VER QR CODE

### **Opção A: Logs do Coolify**

1. Coolify → Baileys → **Logs**
2. Procurar texto grande ASCII (QR Code)
3. Copiar todo o QR

### **Opção B: API Endpoint**

**Se configurou domínio**:
```
https://baileys.olamaestro.com/qr
