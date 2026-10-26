// Only place that builds the API client and attaches the bearer header
import type { ReactNode } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { clearTokens, getAccessToken, getRefreshToken, setTokens } from './auth';
import { ApiError, createClient } from './gen/api';

const client = createClient({
	headers: () => {
		const token = getAccessToken();
		return token ? { Authorization: `Bearer ${token}` } : {};
	}
});

type Client = typeof client;

// UNAUTHENTICATED from these is about the request body, not the bearer token
const noRetry = new Set<keyof Client>(['submitLogin', 'googleLogin', 'refresh', 'logout']);

let refreshing: Promise<boolean> | null = null;

// One refresh per tab at a time; the Web Lock also serialises tabs, which share one
// stored refresh token. A reused (already rotated) token makes the server end every session.
function refreshTokens(staleAccessToken: string): Promise<boolean> {
	refreshing ??= withRefreshLock(async () => {
		const current = getAccessToken();
		// Another request or tab already rotated (or logged out) while we waited
		if (current !== staleAccessToken) return current !== null;
		const refresh_token = getRefreshToken();
		try {
			if (!refresh_token) throw new ApiError('no refresh token', 'UNAUTHENTICATED', 401);
			setTokens(await client.refresh({ refresh_token }));
			return true;
		} catch (err) {
			// Dropping the tokens sends route guards to /login; network errors keep them for a retry
			if (err instanceof ApiError && err.code === 'UNAUTHENTICATED') clearTokens();
			return false;
		}
	}).finally(() => {
		refreshing = null;
	});
	return refreshing;
}

function withRefreshLock(fn: () => Promise<boolean>): Promise<boolean> {
	if (typeof navigator === 'undefined' || !navigator.locks) return fn();
	return navigator.locks.request('template-v7.refresh', fn);
}

// On a 401, refresh once and replay the call with the new access token
function withRefresh<F extends (req: never) => Promise<unknown>>(fn: F): F {
	return (async (req: never) => {
		const accessToken = getAccessToken();
		try {
			return await fn(req);
		} catch (err) {
			const expired = err instanceof ApiError && err.code === 'UNAUTHENTICATED';
			if (!expired || accessToken === null || !(await refreshTokens(accessToken))) throw err;
			return fn(req);
		}
	}) as F;
}

export const api = Object.fromEntries(
	Object.entries(client).map(([name, fn]) => [
		name,
		noRetry.has(name as keyof Client) ? fn : withRefresh(fn)
	])
) as Client;

// Revocation is best effort: the local tokens are gone either way
export function logout(): void {
	const refresh_token = getRefreshToken();
	clearTokens();
	if (refresh_token) client.logout({ refresh_token }).catch(() => {});
}

export const queryClient = new QueryClient();

// Mount AppProviders once at the root, above the router.
export function AppProviders({ children }: { children: ReactNode }) {
	return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}
