// Only module for the tokens; localStorage has XSS risk
const ACCESS_TOKEN_KEY = 'template-v7.access_token';
const REFRESH_TOKEN_KEY = 'template-v7.refresh_token';

// In-memory fallback for non-browser environments (tests)
const memory = new Map<string, string>();

function hasLocalStorage(): boolean {
	return typeof localStorage !== 'undefined';
}

function read(key: string): string | null {
	return hasLocalStorage() ? localStorage.getItem(key) : (memory.get(key) ?? null);
}

function write(key: string, value: string): void {
	if (hasLocalStorage()) localStorage.setItem(key, value);
	else memory.set(key, value);
}

function remove(key: string): void {
	if (hasLocalStorage()) localStorage.removeItem(key);
	else memory.delete(key);
}

export function getAccessToken(): string | null {
	return read(ACCESS_TOKEN_KEY);
}

export function getRefreshToken(): string | null {
	return read(REFRESH_TOKEN_KEY);
}

export function setTokens(tokens: { access_token: string; refresh_token: string }): void {
	write(ACCESS_TOKEN_KEY, tokens.access_token);
	write(REFRESH_TOKEN_KEY, tokens.refresh_token);
}

export function clearTokens(): void {
	remove(ACCESS_TOKEN_KEY);
	remove(REFRESH_TOKEN_KEY);
}

// An expired access token still counts: the API client refreshes it on first 401
export function isAuthenticated(): boolean {
	return getAccessToken() !== null;
}
