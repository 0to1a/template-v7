// Only place that builds the API client and attaches the bearer header
import type { ReactNode } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { clearAccessToken, getAccessToken } from './auth';
import { createClient } from './gen/api';

export const api = createClient({
	headers: () => {
		const token = getAccessToken();
		return token ? { Authorization: `Bearer ${token}` } : {};
	},
	onError: (err) => {
		// Expired or revoked token: drop it so route guards send the user to /login
		if (err.code === 'UNAUTHENTICATED') clearAccessToken();
	}
});

export const queryClient = new QueryClient();

// Mount AppProviders once at the root, above the router.
export function AppProviders({ children }: { children: ReactNode }) {
	return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}
