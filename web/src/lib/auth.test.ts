import { describe, expect, it, beforeEach } from 'vitest';

import { clearTokens, getAccessToken, getRefreshToken, isAuthenticated, setTokens } from './auth';

describe('auth token storage', () => {
	beforeEach(() => {
		clearTokens();
	});

	it('reports not authenticated with no token', () => {
		expect(isAuthenticated()).toBe(false);
		expect(getAccessToken()).toBeNull();
		expect(getRefreshToken()).toBeNull();
	});

	it('reports authenticated once tokens are set', () => {
		setTokens({ access_token: 'a-token', refresh_token: 'r-token' });
		expect(isAuthenticated()).toBe(true);
		expect(getAccessToken()).toBe('a-token');
		expect(getRefreshToken()).toBe('r-token');
	});

	it('clearTokens drops both tokens', () => {
		setTokens({ access_token: 'a-token', refresh_token: 'r-token' });
		clearTokens();
		expect(isAuthenticated()).toBe(false);
		expect(getRefreshToken()).toBeNull();
	});
});
