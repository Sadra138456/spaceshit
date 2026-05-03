const net = require('net');
const tls = require('tls');
const fs = require('fs');
const { CryptoEngine } = require('./crypto-engine');
const { StealthLayer } = require('./stealth-layer');

class MultiProtocolServer {
  constructor(port, psk) {
    this.port = port;
    this.crypto = new CryptoEngine(psk);
    this.stealth = new StealthLayer();
    this.servers = [];
    this.connections = new Map();
  }

  async start() {
    console.log('[SERVER] Starting multi-protocol server...');

    // TLS Server (port 443)
    await this.startTLSServer(443);
    
    // HTTPS Server (port 8443)
    await this.startHTTPSServer(8443);
    
    // WebSocket Server (port 8080)
    await this.startWebSocketServer(8080);

    console.log('[SERVER] ✓ All protocols ready');
  }

  async startTLSServer(port) {
    return new Promise((resolve, reject) => {
      const options = {
        key: fs.readFileSync('server.key'),
        cert: fs.readFileSync('server.crt'),
        minVersion: 'TLSv1.3',
        maxVersion: 'TLSv1.3',
        ciphers: [
          'TLS_AES_128_GCM_SHA256',
          'TLS_AES_256_GCM_SHA384',
          'TLS_CHACHA20_POLY1305_SHA256'
        ].join(':'),
        honorCipherOrder: true,
        ALPNProtocols: ['h2', 'http/1.1']
      };

      const server = tls.createServer(options, (socket) => {
        console.log('[SERVER] New TLS connection');
        this.handleConnection(socket, 'tls');
      });

      server.listen(port, () => {
        console.log(`[SERVER] ✓ TLS listening on port ${port}`);
        this.servers.push(server);
        resolve();
      });

      server.on('error', reject);
    });
  }

  async startHTTPSServer(port) {
    return new Promise((resolve, reject) => {
      const options = {
        key: fs.readFileSync('server.key'),
        cert: fs.readFileSync('server.crt')
      };

      const server = require('https').createServer(options, (req, res) => {
        if (req.method === 'CONNECT') {
          console.log('[SERVER] New HTTPS CONNECT tunnel');
          
          // استخراج داده مخفی شده در headers
          const hiddenData = this.stealth.extractFromHeaders(req.headers);
          
          req.socket.write('HTTP/1.1 200 Connection Established\r\n\r\n');
          this.handleConnection(req.socket, 'https');
        } else {
          // ترافیک عادی HTTP - پاسخ fake
          res.writeHead(200, { 'Content-Type': 'text/html' });
          res.end('<html><body><h1>Welcome</h1></body></html>');
        }
      });

      server.listen(port, () => {
        console.log(`[SERVER] ✓ HTTPS listening on port ${port}`);
        this.servers.push(server);
        resolve();
      });

      server.on('error', reject);
    });
  }

  async startWebSocketServer(port) {
    return new Promise((resolve, reject) => {
      const server = net.createServer((socket) => {
        let buffer = '';

        socket.once('data', (data) => {
          buffer += data.toString();

          if (buffer.includes('Upgrade: websocket')) {
            console.log('[SERVER] New WebSocket connection');
            
            // WebSocket handshake response
            const key = buffer.match(/Sec-WebSocket-Key: (.+)/)?.[1]?.trim();
            
            if (key) {
              const accept = require('crypto')
                .createHash('sha1')
                .update(key + '258EAFA5-E914-47DA-95CA-C5AB0DC85B11')
                .digest('base64');

              const response = [
                'HTTP/1.1 101 Switching Protocols',
                'Upgrade: websocket',
                'Connection: Upgrade',
                `Sec-WebSocket-Accept: ${accept}`,
                '\r\n'
              ].join('\r\n');

              socket.write(response);
              this.handleConnection(socket, 'websocket');
            }
          } else {
            socket.end('HTTP/1.1 400 Bad Request\r\n\r\n');
          }
        });
      });

      server.listen(port, () => {
        console.log(`[SERVER] ✓ WebSocket listening on port ${port}`);
        this.servers.push(server);
        resolve();
      });

      server.on('error', reject);
    });
  }

  handleConnection(socket, protocol) {
    const connId = `${protocol}-${Date.now()}`;
    this.connections.set(connId, socket);

    socket.on('data', async (data) => {
      try {
        // رمزگشایی داده
        const decrypted = this.crypto.decrypt(data);
        
        // پردازش درخواست و ارسال به مقصد
        const response = await this.forwardRequest(decrypted);
        
        // رمزنگاری پاسخ
        const encrypted = this.crypto.encrypt(response);
        
        // اضافه کردن random delay
        await this.stealth.addRandomDelay(5, 30);
        
        socket.write(encrypted);
      } catch (err) {
        console.error(`[SERVER] Error handling ${protocol} data:`, err.message);
      }
    });

    socket.on('error', (err) => {
      console.error(`[SERVER] ${protocol} socket error:`, err.message);
      this.connections.delete(connId);
    });

    socket.on('end', () => {
      console.log(`[SERVER] ${protocol} connection closed`);
      this.connections.delete(connId);
    });
  }

  async forwardRequest(data) {
    // اینجا باید درخواست را به مقصد واقعی forward کنید
    // برای سادگی، فقط echo می‌کنیم
    return data;
  }

  stop() {
    console.log('[SERVER] Stopping all servers...');
    
    for (const server of this.servers) {
      server.close();
    }
    
    for (const [id, socket] of this.connections) {
      socket.end();
    }
    
    this.servers = [];
    this.connections.clear();
    
    console.log('[SERVER] ✓ All servers stopped');
  }
}

function startServer(port = 443, psk = 'your-secret-key') {
  const server = new MultiProtocolServer(port, psk);
  server.start();
  return server;
}

module.exports = { startServer, MultiProtocolServer };
