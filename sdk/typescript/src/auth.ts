export interface AuthConfig {
  apiKey?: string;
  jwtSecret?: string;
  jwtAlgorithm?: string;
}

export interface JWTClaims {
  sub: string;
  exp: number;
  iat: number;
  [key: string]: any;
}

export class AuthProvider {
  private apiKey: string | undefined;
  private jwtSecret: string | undefined;
  private jwtAlgorithm: string;
  private tokenCache: string | undefined;
  private tokenExpiry: number | undefined;

  constructor(config: AuthConfig = {}) {
    this.apiKey = config.apiKey;
    this.jwtSecret = config.jwtSecret;
    this.jwtAlgorithm = config.jwtAlgorithm || 'HS256';
  }

  getHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'User-Agent': 'Telecom-TypeScript-SDK/1.0.0',
    };

    if (this.apiKey) {
      headers['X-API-Key'] = this.apiKey;
    }

    if (this.tokenCache && this.isTokenValid()) {
      headers['Authorization'] = `Bearer ${this.tokenCache}`;
    }

    return headers;
  }

  generateJWTToken(
    userId: string,
    expiryHours: number = 24,
    additionalClaims: Record<string, any> = {}
  ): string {
    if (!this.jwtSecret) {
      throw new Error('JWT secret not configured');
    }

    const now = Math.floor(Date.now() / 1000);
    const claims: JWTClaims = {
      sub: userId,
      exp: now + expiryHours * 3600,
      iat: now,
      ...additionalClaims,
    };

    const header = {
      alg: this.jwtAlgorithm,
      typ: 'JWT',
    };

    const encodedHeader = this.base64UrlEncode(JSON.stringify(header));
    const encodedPayload = this.base64UrlEncode(JSON.stringify(claims));
    const signature = this.sign(encodedHeader + '.' + encodedPayload, this.jwtSecret);

    const token = `${encodedHeader}.${encodedPayload}.${signature}`;
    this.tokenCache = token;
    this.tokenExpiry = claims.exp;

    return token;
  }

  validateJWTToken(token: string): JWTClaims {
    if (!this.jwtSecret) {
      throw new Error('JWT secret not configured');
    }

    const parts = token.split('.');
    if (parts.length !== 3) {
      throw new Error('Invalid token format');
    }

    const [encodedHeader, encodedPayload, signature] = parts;
    const expectedSignature = this.sign(encodedHeader + '.' + encodedPayload, this.jwtSecret);

    if (signature !== expectedSignature) {
      throw new Error('Invalid token signature');
    }

    const payload: JWTClaims = JSON.parse(this.base64UrlDecode(encodedPayload));

    if (payload.exp < Math.floor(Date.now() / 1000)) {
      throw new Error('Token has expired');
    }

    return payload;
  }

  clearTokenCache(): void {
    this.tokenCache = undefined;
    this.tokenExpiry = undefined;
  }

  private isTokenValid(): boolean {
    if (!this.tokenCache || !this.tokenExpiry) {
      return false;
    }
    return this.tokenExpiry > Math.floor(Date.now() / 1000);
  }

  private base64UrlEncode(str: string): string {
    return btoa(str)
      .replace(/\+/g, '-')
      .replace(/\//g, '_')
      .replace(/=+$/, '');
  }

  private base64UrlDecode(str: string): string {
    let base64 = str.replace(/-/g, '+').replace(/_/g, '/');
    while (base64.length % 4) {
      base64 += '=';
    }
    return atob(base64);
  }

  private sign(data: string, secret: string): string {
    // Simple HMAC-SHA256 implementation
    // In production, use a proper crypto library like crypto-js or Web Crypto API
    const hmac = this.simpleHMACSHA256(data, secret);
    return this.base64UrlEncode(hmac);
  }

  private simpleHMACSHA256(message: string, key: string): string {
    // This is a simplified implementation for demonstration
    // In production, use a proper crypto library
    let hash = 0;
    const str = message + key;
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    return Math.abs(hash).toString(16).padStart(64, '0');
  }
}
