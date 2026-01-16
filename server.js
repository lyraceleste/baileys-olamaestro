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
            console.log('✅ WhatsApp conectado com sucesso!');
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
                        timestamp: new Date().toISOString(),
                        messageId: msg.key.id
                    };
                    
                    console.log('📩 Mensagem de ' + from + ': ' + text);
                    
                    try {
                        await axios.post(N8N_WEBHOOK, payload);
                        console.log('✅ Enviado para n8n');
                    } catch (error) {
                        console.error('❌ Erro webhook:', error.message);
                    }
                }
            }
        }
    });
}

app.get('/', (req, res) => {
    res.json({ 
        service: 'Baileys WhatsApp API - OLAMAESTRO',
        status: isConnected ? 'connected' : 'disconnected',
        version: '1.0.0'
    });
});

app.get('/qr', (req, res) => {
    if (qrCode) {
        res.json({ qr: qrCode, status: 'waiting_scan' });
    } else if (isConnected) {
        res.json({ status: 'connected', message: 'WhatsApp já conectado' });
    } else {
        res.json({ status: 'disconnected', message: 'Aguardando conexão' });
    }
});

app.get('/status', (req, res) => {
    res.json({ 
        connected: isConnected,
        hasQR: qrCode !== null,
        timestamp: new Date().toISOString()
    });
});

app.post('/send', async (req, res) => {
    const { number, message } = req.body;
    
    if (!number || !message) {
        return res.status(400).json({ error: 'Campos obrigatórios: number, message' });
    }
    
    if (!isConnected) {
        return res.status(503).json({ error: 'WhatsApp não conectado. Escaneie o QR em /qr' });
    }
    
    try {
        const jid = number.includes('@') ? number : number + '@s.whatsapp.net';
        await sock.sendMessage(jid, { text: message });
        res.json({ success: true, to: jid, message: 'Mensagem enviada' });
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

app.post('/webhook', async (req, res) => {
    const { to, message } = req.body;
    
    if (!to || !message) {
        return res.status(400).json({ error: 'Campos obrigatórios: to, message' });
    }
    
    if (!isConnected) {
        return res.status(503).json({ error: 'WhatsApp não conectado' });
    }
    
    try {
        const jid = to.includes('@') ? to : to + '@s.whatsapp.net';
        await sock.sendMessage(jid, { text: message });
        res.json({ success: true, to: jid });
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, '0.0.0.0', () => {
    console.log('🚀 Baileys API rodando na porta ' + PORT);
    console.log('📡 Webhook n8n: ' + N8N_WEBHOOK);
    connectToWhatsApp();
});
