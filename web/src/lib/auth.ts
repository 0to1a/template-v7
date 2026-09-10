// Only module for the token; localStorage has XSS risk
const ACCESS_TOKEN_KEY = 'template-v7.access_token';

// In-memory fallback for non-browser environments (tests)
let memoryToken: string | null = null;

function hasLocalStorage(): boolean {
	return typeof localStorage !== 'undefined';
}

export function getAccessToken(): string | null {
	if (!hasLocalStorage()) return memoryToken;
	return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function setAccessToken(token: string): void {
	if (!hasLocalStorage()) {
		memoryToken = token;
		return;
	}
	localStorage.setItem(ACCESS_TOKEN_KEY, token);
}

export function clearAccessToken(): void {
	if (!hasLocalStorage()) {
		memoryToken = null;
		return;
	}
	localStorage.removeItem(ACCESS_TOKEN_KEY);
}

export function isAuthenticated(): boolean {
	return getAccessToken() !== null;
}

// No server session; JWT stays valid but unused after this
export function logout(): void {
	clearAccessToken();
}
