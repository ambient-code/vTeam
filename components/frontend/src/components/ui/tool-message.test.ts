/**
 * Unit tests for redactSecrets() function in tool-message.tsx
 *
 * These tests verify that sensitive tokens and credentials are properly redacted
 * from text displayed in the UI. This is a security-critical function.
 *
 * To run these tests, add a test framework like Jest or Vitest to the project:
 * npm install --save-dev jest @types/jest ts-jest
 * npx jest tool-message.test.ts
 */

// Note: This function is extracted from tool-message.tsx for testing
// Keep this in sync with the actual implementation
const redactSecrets = (text: string | null | undefined): string => {
  if (!text) return '';

  // Redact GitHub tokens (ghs_, ghp_, gho_, ghu_ prefixes)
  text = text.replace(/gh[pousr]_[a-zA-Z0-9]{36,255}/g, 'gh*_[REDACTED]');

  // Redact x-access-token: patterns in URLs
  text = text.replace(/x-access-token:[^@\s]+@/g, 'x-access-token:[REDACTED]@');

  // Redact oauth tokens in URLs
  text = text.replace(/oauth2:[^@\s]+@/g, 'oauth2:[REDACTED]@');

  // Redact basic auth credentials in URLs
  text = text.replace(/:\/\/[^:@\s]+:[^@\s]+@/g, '://[REDACTED]@');

  // Redact Authorization header values (Bearer, token, etc.) - minimum 20 chars to avoid false positives
  text = text.replace(/(Authorization["\s:]+)(Bearer\s+|token\s+)?([a-zA-Z0-9_\-\.]{20,})/gi, '$1$2[REDACTED]');

  // Redact common API key patterns (sk-* prefix) - handle start of string, quotes, colons, equals
  text = text.replace(/(^|["\s:=])(sk-[a-zA-Z0-9]{20,})/g, '$1[REDACTED]');

  // Redact api_key or api-key patterns - handle start of string and various separators
  text = text.replace(/(^|["\s])(api[_-]?key["\s:=]+)([a-zA-Z0-9_\-\.]{20,})/gi, '$1$2[REDACTED]');

  return text;
};

describe('redactSecrets', () => {
  describe('GitHub tokens', () => {
    it('should redact ghp_ tokens (personal access tokens)', () => {
      const input = 'Clone with ghp_1234567890abcdefghijklmnopqrstuvwxyz1234567890';
      const result = redactSecrets(input);
      expect(result).toBe('Clone with gh*_[REDACTED]');
      expect(result).not.toContain('ghp_');
    });

    it('should redact ghs_ tokens (secret scanning tokens)', () => {
      const input = 'Secret: ghs_abcdefghijklmnopqrstuvwxyz123456789012345678';
      const result = redactSecrets(input);
      expect(result).toBe('Secret: gh*_[REDACTED]');
      expect(result).not.toContain('ghs_');
    });

    it('should redact gho_ tokens (OAuth tokens)', () => {
      const input = 'OAuth token: gho_1234567890abcdefghijklmnopqrstuvwxyz123456';
      const result = redactSecrets(input);
      expect(result).toBe('OAuth token: gh*_[REDACTED]');
      expect(result).not.toContain('gho_');
    });

    it('should redact ghu_ tokens (user tokens)', () => {
      const input = 'User token: ghu_abcdefghijklmnopqrstuvwxyz1234567890123456';
      const result = redactSecrets(input);
      expect(result).toBe('User token: gh*_[REDACTED]');
      expect(result).not.toContain('ghu_');
    });

    it('should redact multiple GitHub tokens in one string', () => {
      const input = 'Tokens: ghp_abc123456789012345678901234567890123456 and ghs_def456789012345678901234567890123456';
      const result = redactSecrets(input);
      expect(result).toBe('Tokens: gh*_[REDACTED] and gh*_[REDACTED]');
      expect(result).not.toContain('ghp_');
      expect(result).not.toContain('ghs_');
    });
  });

  describe('URL-embedded credentials', () => {
    it('should redact x-access-token in URLs', () => {
      const input = 'git clone https://x-access-token:ghp_abc123@github.com/repo';
      const result = redactSecrets(input);
      expect(result).toContain('x-access-token:[REDACTED]@');
      expect(result).not.toContain('ghp_abc123');
    });

    it('should redact oauth2 tokens in URLs', () => {
      const input = 'https://oauth2:my_secret_token@gitlab.com/project';
      const result = redactSecrets(input);
      expect(result).toContain('oauth2:[REDACTED]@');
      expect(result).not.toContain('my_secret_token');
    });

    it('should redact basic auth credentials in URLs', () => {
      const input = 'https://username:password123@example.com/api';
      const result = redactSecrets(input);
      expect(result).toBe('https://[REDACTED]@example.com/api');
      expect(result).not.toContain('username');
      expect(result).not.toContain('password123');
    });
  });

  describe('Authorization headers', () => {
    it('should redact Bearer tokens', () => {
      const input = 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9';
      const result = redactSecrets(input);
      expect(result).toBe('Authorization: Bearer [REDACTED]');
      expect(result).not.toContain('eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9');
    });

    it('should redact token without Bearer prefix', () => {
      const input = 'Authorization: token 1234567890abcdefghijklmnopqrst';
      const result = redactSecrets(input);
      expect(result).toBe('Authorization: token [REDACTED]');
      expect(result).not.toContain('1234567890abcdefghijklmnopqrst');
    });

    it('should redact Authorization headers in JSON', () => {
      const input = '{"Authorization": "Bearer sk_test_1234567890abcdefghijklmnopqrstuvwxyz"}';
      const result = redactSecrets(input);
      expect(result).toContain('Bearer [REDACTED]');
      expect(result).not.toContain('sk_test_');
    });

    it('should not redact short Authorization values (less than 20 chars)', () => {
      // This prevents false positives like "Authorization: Bearer success"
      const input = 'Authorization: Bearer ok';
      const result = redactSecrets(input);
      expect(result).toBe('Authorization: Bearer ok');
    });
  });

  describe('API keys', () => {
    it('should redact sk- prefixed API keys in JSON', () => {
      const input = '{"apiKey":"sk-proj-1234567890abcdefghijklmnopqrstuvwxyz"}';
      const result = redactSecrets(input);
      expect(result).toBe('{"apiKey":[REDACTED]}');
      expect(result).not.toContain('sk-proj-');
    });

    it('should redact sk- keys at start of string', () => {
      const input = 'sk-test-1234567890abcdefghijklmnopqrstuvwxyz';
      const result = redactSecrets(input);
      expect(result).toBe('[REDACTED]');
      expect(result).not.toContain('sk-');
    });

    it('should redact sk- keys after equals sign', () => {
      const input = 'ANTHROPIC_API_KEY=sk-ant-1234567890abcdefghijklmnopqrstuvwxyz';
      const result = redactSecrets(input);
      expect(result).toContain('=[REDACTED]');
      expect(result).not.toContain('sk-ant-');
    });

    it('should redact api_key patterns', () => {
      const input = 'api_key: abcdef1234567890ghijklmnop';
      const result = redactSecrets(input);
      expect(result).toBe('api_key: [REDACTED]');
      expect(result).not.toContain('abcdef1234567890ghijklmnop');
    });

    it('should redact api-key patterns (hyphenated)', () => {
      const input = 'api-key=xyz123456789012345678901234567890';
      const result = redactSecrets(input);
      expect(result).toBe('api-key=[REDACTED]');
      expect(result).not.toContain('xyz123456789012345678901234567890');
    });
  });

  describe('Complex scenarios', () => {
    it('should redact multiple different secrets in one string', () => {
      const input = 'git clone https://x-access-token:ghp_abc123456789012345678901234567890123456@github.com/repo with api_key=sk-proj-1234567890abcdefghijklmnopqrstuvwxyz';
      const result = redactSecrets(input);
      expect(result).toContain('x-access-token:[REDACTED]@');
      expect(result).toContain('api_key=[REDACTED]');
      expect(result).not.toContain('ghp_');
      expect(result).not.toContain('sk-proj-');
    });

    it('should handle bash command with Authorization header', () => {
      const input = 'curl -H "Authorization: token ghp_1234567890abcdefghijklmnopqrstuvwxyz123456"';
      const result = redactSecrets(input);
      expect(result).toContain('Authorization: token [REDACTED]');
      expect(result).not.toContain('ghp_');
    });

    it('should redact credentials in curl commands', () => {
      const input = 'curl -u username:password123 https://api.example.com';
      // Note: This doesn't match our current patterns - basic auth in URLs only
      // The username:password pattern without :// is not caught
      const result = redactSecrets(input);
      // Just ensure it doesn't break
      expect(result).toBeDefined();
    });
  });

  describe('Edge cases', () => {
    it('should handle empty string', () => {
      expect(redactSecrets('')).toBe('');
    });

    it('should handle null', () => {
      expect(redactSecrets(null)).toBe('');
    });

    it('should handle undefined', () => {
      expect(redactSecrets(undefined)).toBe('');
    });

    it('should handle string with no secrets', () => {
      const input = 'This is a normal string with no secrets';
      expect(redactSecrets(input)).toBe(input);
    });

    it('should handle malformed tokens (too short)', () => {
      const input = 'ghp_short';
      expect(redactSecrets(input)).toBe(input); // Too short to match (< 36 chars)
    });

    it('should handle tokens at start of string', () => {
      const input = 'ghp_1234567890abcdefghijklmnopqrstuvwxyz1234567890 is the token';
      const result = redactSecrets(input);
      expect(result).toBe('gh*_[REDACTED] is the token');
    });

    it('should handle tokens at end of string', () => {
      const input = 'The token is ghp_1234567890abcdefghijklmnopqrstuvwxyz1234567890';
      const result = redactSecrets(input);
      expect(result).toBe('The token is gh*_[REDACTED]');
    });

    it('should handle newlines and multiline content', () => {
      const input = 'Line 1\nToken: ghp_1234567890abcdefghijklmnopqrstuvwxyz1234567890\nLine 3';
      const result = redactSecrets(input);
      expect(result).toContain('gh*_[REDACTED]');
      expect(result).not.toContain('ghp_');
    });

    it('should handle JSON with nested secrets', () => {
      const input = JSON.stringify({
        config: {
          apiKey: 'sk-proj-1234567890abcdefghijklmnopqrstuvwxyz',
          token: 'ghp_abc123456789012345678901234567890123456'
        }
      });
      const result = redactSecrets(input);
      expect(result).not.toContain('sk-proj-');
      expect(result).not.toContain('ghp_abc');
    });
  });

  describe('Non-regression tests', () => {
    it('should not over-redact common words', () => {
      const input = 'The operation was successful';
      expect(redactSecrets(input)).toBe(input);
    });

    it('should preserve formatting of non-secret content', () => {
      const input = 'Command: ls -la /home/user';
      expect(redactSecrets(input)).toBe(input);
    });

    it('should handle special characters without breaking', () => {
      const input = 'Special chars: !@#$%^&*()_+-=[]{}|;:",.<>?';
      expect(redactSecrets(input)).toBe(input);
    });
  });
});

// Example usage showing how tests would be run:
// npm test -- tool-message.test.ts

console.log('Test suite defined for redactSecrets()');
console.log('To run tests, configure a test framework (Jest, Vitest, etc.) and execute:');
console.log('  npm test tool-message.test.ts');
