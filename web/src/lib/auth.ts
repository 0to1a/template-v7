// Only module for the tokens; localStorage has XSS risk
export const ACCESS_TOKEN_KEY = 'template-v7.access_token';
const REFRESH_TOKEN_KEY = 'template-v7.refresh_token';

export function getAccessToken(): string | null {
	return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function getRefreshToken(): string | null {
	return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function setTokens(tokens: { access_token: string; refresh_token: string }): void {
	localStorage.setItem(ACCESS_TOKEN_KEY, tokens.access_token);
	localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refresh_token);
}

export function clearTokens(): void {
	localStorage.removeItem(ACCESS_TOKEN_KEY);
	localStorage.removeItem(REFRESH_TOKEN_KEY);
}

// An expired access token still counts: the API client refreshes it on first 401
export function isAuthenticated(): boolean {
	return getAccessToken() !== null;
}
