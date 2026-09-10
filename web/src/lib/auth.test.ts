import { describe, expect, it, beforeEach } from 'vitest';

import { clearAccessToken, getAccessToken, isAuthenticated, logout, setAccessToken } from './auth';

describe('auth token storage', () => {
	beforeEach(() => {
		clearAccessToken();
	});

	it('reports not authenticated with no token', () => {
		expect(isAuthenticated()).toBe(false);
		expect(getAccessToken()).toBeNull();
	});

	it('reports authenticated once a token is set', () => {
		setAccessToken('a-token');
		expect(isAuthenticated()).toBe(true);
		expect(getAccessToken()).toBe('a-token');
	});

	it('logout clears the held token', () => {
		setAccessToken('a-token');
		logout();
		expect(isAuthenticated()).toBe(false);
		expect(getAccessToken()).toBeNull();
	});
});
