const net = require('net');
const tls = require('tls');
const http = require('http');
const https = require('https');
const { DoHTunnel } = require('./doh-tunnel');

class PortScanner {
  constructor(aiEndpoint = 'https://gapgpt.app/api/chat') {
    this.aiEndpoint = aiEndpoint;
    this.protocols = ['tls', 'http', 'https', 'websocket', 'doh', 'quic'];
    this.commonPorts = [443, 80, 8080, 8443, 53, 853, 5353];
    this.history = [];
    this.dohTunnel = new DoHTunnel();
  }

  async scanHost(host) {
    console.log(`[SCANNER] Scanning ${host}...`);
    
    const results = [];
    
    for (const port of this.commonPorts) {
      for (const protocol of this.protocols) {
        const result = await this.testProtocol(host, port, protocol);
        
        if (result.success) {
          console.log(`[SCANNER] ✓ ${protocol.toUpperCase()} on port ${port} works!`);
          results.push({ port, protocol, latency: result.latency });
        }
        
        this.history.push({
          host,
          port,
          protocol,
          success: result.success,
          timestamp: Date.now()
        });
      }
    }
    
    if (results.length === 0) {
      console.log('[SCANNER] ✗ No open paths found');
      return null;
    }
    
    // استفاده از AI برای انتخاب بهترین مسیر
    const best = await this.selectBestPath(results);
    console.log(`[SCANNER] AI selected: ${best.protocol.toUpperCase()} on port ${best.port}`);
    
    return best;
  }

  async testProtocol(host, port, protocol) {
    const start = Date.now();
    
    try {
      switch (protocol) {
        case 'tls':
          return await this.testTLS(host, port, start);
        case 'http':
          return await this.testHTTP(host, port, start);
        case 'https':
          return await this.testHTTPS(host, port, start);
        case 'websocket':
          return await this.testWebSocket(host, port, start);
        case 'doh':
          return await this.testDoH(host, port, start);
        case 'quic':
          return await this.testQUIC(host, port, start);
        default:
          return { success: false };
      }
    } catch (err) {
      return { success: false, error: err.message };
    }
  }

  async testTLS(host, port, start) {
    return new Promise((resolve) => {
      const socket = tls.connect({
        host,
        port,
        rejectUnauthorized: false,
        minVersion: 'TLSv1.3'
      });

      socket.on('secureConnect', () => {
        socket.end();
        resolve({ success: true, protocol: 'tls', latency: Date.now() - start });
      });

      socket.on('error', () => {
        resolve({ success: false });
      });

      setTimeout(() => {
        socket.destroy();
        resolve({ success: false });
      }, 3000);
    });
  }

  async testHTTP(host, port, start) {
    return new Promise((resolve) => {
      const req = http.request({
        host,
        port,
        method: 'HEAD',
        path: '/',
        timeout: 3000
      }, (res) => {
        resolve({ success: true, protocol: 'http', latency: Date.now() - start });
      });

      req.on('error', () => resolve({ success: false }));
      req.end();
    });
  }

  async testHTTPS(host, port, start) {
    return new Promise((resolve) => {
      const req = https.request({
        host,
        port,
        method: 'HEAD',
        path: '/',
        rejectUnauthorized: false,
        timeout: 3000
      }, (res) => {
        resolve({ success: true, protocol: 'https', latency: Date.now() - start });
      });

      req.on('error', () => resolve({ success: false }));
      req.end();
    });
  }

  async testWebSocket(host, port, start) {
    return new Promise((resolve) => {
      const socket = net.connect({ host, port });

      socket.on('connect', () => {
        const upgrade = [
          `GET / HTTP/1.1`,
          `Host: ${host}`,
          `Upgrade: websocket`,
          `Connection: Upgrade`,
          `Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==`,
          `Sec-WebSocket-Version: 13`,
          `\r\n`
        ].join('\r\n');

        socket.write(upgrade);
      });

      socket.on('data', (data) => {
        socket.end();
        const response = data.toString();
        if (response.includes('101') || response.includes('Upgrade')) {
          resolve({ success: true, protocol: 'websocket', latency: Date.now() - start });
        } else {
          resolve({ success: false });
        }
      });

      socket.on('error', () => resolve({ success: false }));

      setTimeout(() => {
        socket.destroy();
        resolve({ success: false });
      }, 3000);
    });
  }

  async testDoH(host, port, start) {
    try {
      const result = await this.dohTunnel.queryDoH('google.com', 'A');
      return {
        success: result.success,
        protocol: 'doh',
        latency: Date.now() - start
      };
    } catch (err) {
      return { success: false };
    }
  }

  async testQUIC(host, port, start) {
    // QUIC نیاز به کتابخانه خاص دارد - اینجا placeholder است
    // در production باید از quic-js یا node-quic استفاده کنید
    return { success: false }; // فعلاً غیرفعال
  }

  async selectBestPath(results) {
    if (results.length === 1) return results[0];
    
    try {
      const prompt = `Based on these available paths in Iran (DPI-heavy environment):
${JSON.stringify(results, null, 2)}

Recent history (last 10):
${JSON.stringify(this.history.slice(-10), null, 2)}

Which path is LEAST likely to be detected by DPI? Consider:
1. Protocol fingerprinting resistance
2. Common traffic patterns
3. Historical success rate

Respond with JSON: {"port": number, "protocol": "string", "reason": "string"}`;

      const response = await fetch(this.aiEndpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          messages: [{ role: 'user', content: prompt }],
          temperature: 0.3
        })
      });

      const data = await response.json();
      const aiChoice = JSON.parse(data.choices[0].message.content);
      
      const selected = results.find(r => 
        r.port === aiChoice.port && r.protocol === aiChoice.protocol
      );
      
      return selected || results[0];
      
    } catch (err) {
      console.log('[SCANNER] AI selection failed, using fastest path');
      return results.sort((a, b) => a.latency - b.latency)[0];
    }
  }
}

module.exports = { PortScanner };
