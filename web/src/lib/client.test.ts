import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { clearTokens, getAccessToken, getRefreshToken, setTokens } from './auth';
import { api, logout } from './client';

type Handler = (url: string, init: RequestInit) => Response;

function json(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), { status });
}

const unauthenticated = () => json(401, { status: 'UNAUTHENTICATED', message: 'unauthenticated' });
const profile = { email: 'a@example.com', display_name: 'A' };

function stubFetch(handler: Handler) {
	const calls: { url: string; auth: string | null; body: unknown }[] = [];
	vi.stubGlobal(
		'fetch',
		vi.fn(async (url: string, init: RequestInit) => {
			calls.push({
				url,
				auth: new Headers(init.headers).get('Authorization'),
				body: init.body ? JSON.parse(init.body as string) : undefined
			});
			return handler(url, init);
		})
	);
	return calls;
}

describe('api client refresh', () => {
	beforeEach(() => {
		clearTokens();
		setTokens({ access_token: 'old-access', refresh_token: 'old-refresh' });
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('refreshes once on 401 and replays concurrent calls with the new token', async () => {
		const calls = stubFetch((url, init) => {
			if (url.endsWith('/refresh')) {
				return json(200, { data: { access_token: 'new-access', refresh_token: 'new-refresh' } });
			}
			const auth = new Headers(init.headers).get('Authorization');
			return auth === 'Bearer new-access' ? json(200, { data: profile }) : unauthenticated();
		});

		const results = await Promise.all([api.getProfile({}), api.getProfile({})]);

		expect(results).toEqual([profile, profile]);
		const refreshes = calls.filter((c) => c.url.endsWith('/refresh'));
		expect(refreshes).toHaveLength(1);
		expect(refreshes[0].body).toEqual({ refresh_token: 'old-refresh' });
		expect(getAccessToken()).toBe('new-access');
		expect(getRefreshToken()).toBe('new-refresh');
	});

	it('drops the tokens when the refresh token is rejected', async () => {
		stubFetch(() => unauthenticated());

		await expect(api.getProfile({})).rejects.toMatchObject({ code: 'UNAUTHENTICATED' });
		expect(getAccessToken()).toBeNull();
		expect(getRefreshToken()).toBeNull();
	});

	it('does not refresh when a login code is rejected', async () => {
		const calls = stubFetch(() => unauthenticated());

		await expect(api.submitLogin({ email: 'a@example.com', code: '000000' })).rejects.toThrow();
		expect(calls).toHaveLength(1);
	});

	it('logout clears locally and revokes the refresh token', async () => {
		const calls = stubFetch(() => new Response(null, { status: 204 }));

		logout();

		expect(getAccessToken()).toBeNull();
		expect(calls[0]).toMatchObject({
			url: '/api/auth/logout',
			body: { refresh_token: 'old-refresh' }
		});
	});
});
