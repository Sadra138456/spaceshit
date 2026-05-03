const crypto = require('crypto');

class CryptoEngine {
  constructor(psk) {
    this.psk = Buffer.from(psk, 'utf8');
    this.algorithm = 'aes-256-gcm';
  }

  // تولید کلید از PSK با HKDF
  deriveKey(salt, info = 'tunnel-key') {
    return crypto.hkdfSync('sha256', this.psk, salt, Buffer.from(info), 32);
  }

  // رمزنگاری با AES-256-GCM
  encrypt(plaintext, associatedData = null) {
    const iv = crypto.randomBytes(12);
    const salt = crypto.randomBytes(16);
    const key = this.deriveKey(salt);
    
    const cipher = crypto.createCipheriv(this.algorithm, key, iv);
    
    if (associatedData) {
      cipher.setAAD(Buffer.from(associatedData));
    }
    
    const encrypted = Buffer.concat([
      cipher.update(plaintext),
      cipher.final()
    ]);
    
    const authTag = cipher.getAuthTag();
    
    // Format: [salt(16)][iv(12)][authTag(16)][ciphertext]
    return Buffer.concat([salt, iv, authTag, encrypted]);
  }

  // رمزگشایی
  decrypt(ciphertext, associatedData = null) {
    const salt = ciphertext.slice(0, 16);
    const iv = ciphertext.slice(16, 28);
    const authTag = ciphertext.slice(28, 44);
    const encrypted = ciphertext.slice(44);
    
    const key = this.deriveKey(salt);
    
    const decipher = crypto.createDecipheriv(this.algorithm, key, iv);
    decipher.setAuthTag(authTag);
    
    if (associatedData) {
      decipher.setAAD(Buffer.from(associatedData));
    }
    
    return Buffer.concat([
      decipher.update(encrypted),
      decipher.final()
    ]);
  }

  // تولید fingerprint TLS واقعی
  generateTLSFingerprint(type = 'chrome') {
    const fingerprints = {
      chrome: {
        ciphers: [
          'TLS_AES_128_GCM_SHA256',
          'TLS_AES_256_GCM_SHA384',
          'TLS_CHACHA20_POLY1305_SHA256',
          'ECDHE-ECDSA-AES128-GCM-SHA256',
          'ECDHE-RSA-AES128-GCM-SHA256'
        ],
        curves: ['X25519', 'prime256v1', 'secp384r1'],
        extensions: [
          'server_name',
          'extended_master_secret',
          'renegotiation_info',
          'supported_groups',
          'ec_point_formats',
          'session_ticket',
          'application_layer_protocol_negotiation',
          'status_request',
          'signature_algorithms',
          'signed_certificate_timestamp',
          'key_share',
          'psk_key_exchange_modes',
          'supported_versions',
          'compress_certificate',
          'application_settings'
        ]
      },
      firefox: {
        ciphers: [
          'TLS_AES_128_GCM_SHA256',
          'TLS_CHACHA20_POLY1305_SHA256',
          'TLS_AES_256_GCM_SHA384'
        ],
        curves: ['X25519', 'prime256v1', 'secp384r1', 'secp521r1'],
        extensions: [
          'server_name',
          'extended_master_secret',
          'renegotiation_info',
          'supported_groups',
          'ec_point_formats',
          'session_ticket',
          'application_layer_protocol_negotiation',
          'status_request',
          'delegated_credentials',
          'key_share',
          'supported_versions',
          'signature_algorithms',
          'psk_key_exchange_modes',
          'record_size_limit'
        ]
      }
    };
    
    return fingerprints[type] || fingerprints.chrome;
  }
}

module.exports = { CryptoEngine };
