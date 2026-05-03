const net = require('net');
const tls = require('tls');
const { CryptoEngine } = require('./crypto-engine');
const { StealthLayer } = require('./stealth-layer');
const { DoHTunnel } = require('./doh-tunnel');
const { PortScanner } = require('./port-scanner');

class AdaptiveClient {
  constructor(serverHost, psk, aiEndpoint) {
    this.serverHost = serverHost;
    this.crypto = new CryptoEngine(psk);
    this.stealth = new StealthLayer();
    this.dohTunnel = new DoHTunnel();
    this.scanner = new PortScanner(aiEndpoint);
    
    this.currentPath = null;
    this.connection = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.failureCount = 0;
  }

  async connect() {
    console.log('[CLIENT] Starting adaptive connection...');
    
    // مرحله 1: اسکن و پیدا کردن بهترین مسیر
    this.currentPath = await this.scanner.scanHost(this.serverHost);
    
    if (!this.currentPath) {
      throw new Error('No viable path found to server');
    }
    
    console.log(`[CLIENT] Using ${this.currentPath.protocol.toUpperCase()} on port ${this.currentPath.port}`);
    
    // مرحله 2: اتصال بر اساس پروتکل انتخاب شده
    switch (this.currentPath.protocol) {
      case 'tls':
        return await this.connectTLS();
      case 'https':
        return await this.connectHTTPS();
      case 'websocket':
        return await this.connectWebSocket();
      case 'doh':
        return await this.connectDoH();
      default:
        return await this.connectTLS(); // fallback
    }
  }

  async connectTLS() {
    return new Promise((resolve, reject) => {
      const fingerprint = this.crypto.generateTLSFingerprint('chrome');
      
      const options = {
        host: this.serverHost,
        port: this.currentPath.port,
        rejectUnauthorized: false,
        minVersion: 'TLSv1.3',
        maxVersion: 'TLSv1.3',
        ciphers: fingerprint.ciphers.join(':'),
        ecdhCurve: fingerprint.curves.join(':'),
        // Domain fronting
        servername: this.stealth.domains[0], // SNI مختلف
        ALPNProtocols: ['h2', 'http/1.1']
      };

      this.connection = tls.connect(options);

      this.connection.on('secureConnect', () => {
        console.log('[CLIENT] ✓ TLS connection established');
        this.reconnectAttempts = 0;
        this.setupConnection();
        resolve();
      });

      this.connection.on('error', (err) => {
        console.error('[CLIENT] ✗ TLS error:', err.message);
        this.handleConnectionFailure();
        reject(err);
      });

      this.connection.on('end', () => {
        console.log('[CLIENT] Connection closed');
        this.handleConnectionFailure();
      });
    });
  }

  async connectHTTPS() {
    // HTTP/2 tunneling با steganography
    return new Promise((resolve, reject) => {
      const headers = this.stealth.hideInHeaders(Buffer.from('CONNECT'));
      
      const options = {
        host: this.serverHost,
        port: this.currentPath.port,
        method: 'CONNECT',
        path: `${this.serverHost}:${this.currentPath.port}`,
        headers: headers,
        rejectUnauthorized: false
      };

      const req = require('https').request(options);

      req.on('connect', (res, socket) => {
        console.log('[CLIENT] ✓ HTTPS tunnel established');
        this.connection = socket;
        this.reconnectAttempts = 0;
        this.setupConnection();
        resolve();
      });

      req.on('error', (err) => {
        console.error('[CLIENT] ✗ HTTPS error:', err.message);
        this.handleConnectionFailure();
        reject(err);
      });

      req.end();
    });
  }

  async connectWebSocket() {
    return new Promise((resolve, reject) => {
      const socket = net.connect({
        host: this.serverHost,
        port: this.currentPath.port
      });

      socket.on('connect', () => {
        // WebSocket handshake با steganography
        const headers = this.stealth.hideInHeaders(Buffer.from('tunnel-key'));
        
        const handshake = [
          `GET /tunnel HTTP/1.1`,
          `Host: ${this.serverHost}`,
          `Upgrade: websocket`,
          `Connection: Upgrade`,
          `Sec-WebSocket-Key: ${headers['X-Request-ID']}`,
          `Sec-WebSocket-Version: 13`,
          `User-Agent: ${headers['User-Agent']}`,
          `\r\n`
        ].join('\r\n');

        socket.write(handshake);
      });

      socket.on('data', (data) => {
        const response = data.toString();
        if (response.includes('101') || response.includes('Upgrade')) {
          console.log('[CLIENT] ✓ WebSocket tunnel established');
          this.connection = socket;
          this.reconnectAttempts = 0;
          this.setupConnection();
          resolve();
        } else {
          reject(new Error('WebSocket handshake failed'));
        }
      });

      socket.on('error', (err) => {
        console.error('[CLIENT] ✗ WebSocket error:', err.message);
        this.handleConnectionFailure();
        reject(err);
      });
    });
  }

  async connectDoH() {
    console.log('[CLIENT] ✓ DoH tunnel mode activated');
    // DoH tunneling - داده‌ها از طریق DNS queries ارسال می‌شوند
    this.connection = {
      isDohMode: true,
      write: async (data) => {
        await this.dohTunnel.sendViaDNS(data, this.serverHost);
      }
    };
    this.reconnectAttempts = 0;
    return Promise.resolve();
  }

  setupConnection() {
    if (!this.connection || this.connection.isDohMode) return;

    this.connection.on('data', (data) => {
      try {
        // رمزگشایی داده
        const decrypted = this.crypto.decrypt(data);
        this.emit('data', decrypted);
      } catch (err) {
        console.error('[CLIENT] Decryption failed:', err.message);
      }
    });

    this.connection.on('error', (err) => {
      console.error('[CLIENT] Connection error:', err.message);
      this.handleConnectionFailure();
    });

    this.connection.on('end', () => {
      console.log('[CLIENT] Connection ended');
      this.handleConnectionFailure();
    });
  }

  async send(data) {
    if (!this.connection) {
      throw new Error('Not connected');
    }

    // اضافه کردن random delay برای obfuscation
    await this.stealth.addRandomDelay(10, 50);

    // رمزنگاری داده
    const encrypted = this.crypto.encrypt(data);

    if (this.connection.isDohMode) {
      await this.connection.write(encrypted);
    } else {
      this.connection.write(encrypted);
    }
  }

  async handleConnectionFailure() {
    this.failureCount++;
    
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log('[CLIENT] Max reconnect attempts reached, rescanning...');
      this.reconnectAttempts = 0;
      
      // اسکن مجدد برای پیدا کردن مسیر جدید
      setTimeout(() => this.connect(), 5000);
      return;
    }

    this.reconnectAttempts++;
    console.log(`[CLIENT] Reconnecting (attempt ${this.reconnectAttempts})...`);
    
    setTimeout(() => this.connect(), 2000);
  }

  emit(event, data) {
    if (this.onData && event === 'data') {
      this.onData(data);
    }
  }

  close() {
    if (this.connection && !this.connection.isDohMode) {
      this.connection.end();
    }
    this.connection = null;
  }
}

module.exports = { AdaptiveClient };
