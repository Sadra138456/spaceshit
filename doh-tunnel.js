const https = require('https');
const dgram = require('dgram');

class DoHTunnel {
  constructor(dohServer = 'https://1.1.1.1/dns-query') {
    this.dohServer = dohServer;
    this.cache = new Map();
  }

  // تونل کردن داده از طریق DNS queries
  async sendViaDNS(data, targetDomain) {
    // تقسیم داده به چانک‌های کوچک (max 63 bytes per label)
    const chunks = this.splitToChunks(data, 50);
    const results = [];
    
    for (let i = 0; i < chunks.length; i++) {
      const encoded = Buffer.from(chunks[i]).toString('base64')
        .replace(/\+/g, '-')
        .replace(/\//g, '_')
        .replace(/=/g, '');
      
      // ساخت subdomain
      const subdomain = `${i}-${encoded}.${targetDomain}`;
      
      try {
        const result = await this.queryDoH(subdomain, 'TXT');
        results.push(result);
      } catch (err) {
        console.error(`[DoH] Query failed for chunk ${i}:`, err.message);
      }
    }
    
    return results;
  }

  // Query به DoH server
  async queryDoH(domain, type = 'A') {
    const cacheKey = `${domain}:${type}`;
    
    if (this.cache.has(cacheKey)) {
      return this.cache.get(cacheKey);
    }
    
    return new Promise((resolve, reject) => {
      const dnsMessage = this.buildDNSQuery(domain, type);
      
      const options = {
        method: 'POST',
        headers: {
          'Content-Type': 'application/dns-message',
          'Content-Length': dnsMessage.length,
          'Accept': 'application/dns-message'
        }
      };
      
      const req = https.request(this.dohServer, options, (res) => {
        const chunks = [];
        
        res.on('data', chunk => chunks.push(chunk));
        
        res.on('end', () => {
          const response = Buffer.concat(chunks);
          const parsed = this.parseDNSResponse(response);
          
          this.cache.set(cacheKey, parsed);
          setTimeout(() => this.cache.delete(cacheKey), 300000); // 5 min TTL
          
          resolve(parsed);
        });
      });
      
      req.on('error', reject);
      req.write(dnsMessage);
      req.end();
    });
  }

  // ساخت DNS query message
  buildDNSQuery(domain, type) {
    const buf = Buffer.alloc(512);
    let offset = 0;
    
    // Header
    buf.writeUInt16BE(Math.floor(Math.random() * 65535), offset); // ID
    offset += 2;
    buf.writeUInt16BE(0x0100, offset); // Flags: standard query
    offset += 2;
    buf.writeUInt16BE(1, offset); // Questions: 1
    offset += 2;
    buf.writeUInt16BE(0, offset); // Answer RRs
    offset += 2;
    buf.writeUInt16BE(0, offset); // Authority RRs
    offset += 2;
    buf.writeUInt16BE(0, offset); // Additional RRs
    offset += 2;
    
    // Question
    const labels = domain.split('.');
    for (const label of labels) {
      buf.writeUInt8(label.length, offset++);
      buf.write(label, offset);
      offset += label.length;
    }
    buf.writeUInt8(0, offset++); // End of domain
    
    const typeCode = type === 'TXT' ? 16 : 1; // A=1, TXT=16
    buf.writeUInt16BE(typeCode, offset); // Type
    offset += 2;
    buf.writeUInt16BE(1, offset); // Class: IN
    offset += 2;
    
    return buf.slice(0, offset);
  }

  // Parse DNS response
  parseDNSResponse(buffer) {
    try {
      // ساده‌سازی شده - فقط برای نمایش
      const id = buffer.readUInt16BE(0);
      const flags = buffer.readUInt16BE(2);
      const questions = buffer.readUInt16BE(4);
      const answers = buffer.readUInt16BE(6);
      
      return {
        id,
        success: (flags & 0x8000) !== 0,
        answers
      };
    } catch (err) {
      return { success: false, error: err.message };
    }
  }

  splitToChunks(data, size) {
    const chunks = [];
    const buffer = Buffer.from(data);
    
    for (let i = 0; i < buffer.length; i += size) {
      chunks.push(buffer.slice(i, i + size));
    }
    
    return chunks;
  }
}

module.exports = { DoHTunnel };
