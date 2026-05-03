const crypto = require('crypto');

class StealthLayer {
  constructor() {
    this.domains = [
      'www.google.com',
      'www.cloudflare.com',
      'www.microsoft.com',
      'api.github.com',
      'www.wikipedia.org'
    ];
    
    this.userAgents = [
      'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0',
      'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Safari/537.36',
      'Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/121.0'
    ];
  }

  // مخفی کردن داده در HTTP headers
  hideInHeaders(data) {
    const encoded = Buffer.from(data).toString('base64');
    const chunks = this.splitString(encoded, 32);
    
    const headers = {
      'User-Agent': this.randomChoice(this.userAgents),
      'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
      'Accept-Language': 'en-US,en;q=0.5',
      'Accept-Encoding': 'gzip, deflate, br',
      'DNT': '1',
      'Connection': 'keep-alive',
      'Upgrade-Insecure-Requests': '1',
      'Sec-Fetch-Dest': 'document',
      'Sec-Fetch-Mode': 'navigate',
      'Sec-Fetch-Site': 'none',
      'Cache-Control': 'max-age=0',
      // مخفی کردن داده در custom headers
      'X-Request-ID': chunks[0] || this.randomHex(32),
      'X-Trace-ID': chunks[1] || this.randomHex(32),
      'X-Session-Token': chunks[2] || this.randomHex(32)
    };
    
    return headers;
  }

  // استخراج داده از headers
  extractFromHeaders(headers) {
    try {
      const parts = [
        headers['x-request-id'],
        headers['x-trace-id'],
        headers['x-session-token']
      ].filter(Boolean);
      
      const encoded = parts.join('');
      return Buffer.from(encoded, 'base64');
    } catch (err) {
      return null;
    }
  }

  // تولید ترافیک fake
  generateFakeTraffic() {
    const types = ['GET', 'POST', 'OPTIONS'];
    const paths = ['/api/v1/status', '/health', '/metrics', '/favicon.ico'];
    
    return {
      method: this.randomChoice(types),
      path: this.randomChoice(paths),
      host: this.randomChoice(this.domains),
      headers: {
        'User-Agent': this.randomChoice(this.userAgents),
        'Accept': '*/*',
        'Connection': 'keep-alive'
      }
    };
  }

  // Domain fronting
  createFrontedRequest(realHost, frontDomain) {
    return {
      headers: {
        'Host': realHost,              // واقعی
        'X-Forwarded-Host': frontDomain, // ظاهری
        'User-Agent': this.randomChoice(this.userAgents)
      },
      sni: frontDomain  // SNI مختلف از Host
    };
  }

  // Timing obfuscation
  async addRandomDelay(min = 10, max = 100) {
    const delay = Math.floor(Math.random() * (max - min + 1)) + min;
    await new Promise(resolve => setTimeout(resolve, delay));
  }

  // Helper functions
  splitString(str, size) {
    const chunks = [];
    for (let i = 0; i < str.length; i += size) {
      chunks.push(str.slice(i, i + size));
    }
    return chunks;
  }

  randomChoice(arr) {
    return arr[Math.floor(Math.random() * arr.length)];
  }

  randomHex(length) {
    return crypto.randomBytes(length / 2).toString('hex');
  }
}

module.exports = { StealthLayer };
