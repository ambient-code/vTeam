import { getApiBaseUrl } from '../client';

describe('API Client', () => {
  describe('getApiBaseUrl', () => {
    const originalEnv = process.env;

    beforeEach(() => {
      jest.resetModules();
      process.env = { ...originalEnv };
    });

    afterEach(() => {
      process.env = originalEnv;
    });

    it('returns default /api when no env var is set', () => {
      delete process.env.NEXT_PUBLIC_API_URL;
      expect(getApiBaseUrl()).toBe('/api');
    });

    it('returns custom URL when NEXT_PUBLIC_API_URL is set', () => {
      process.env.NEXT_PUBLIC_API_URL = 'https://api.example.com';
      expect(getApiBaseUrl()).toBe('https://api.example.com');
    });
  });
});

