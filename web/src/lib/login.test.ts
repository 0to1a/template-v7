import { describe, expect, it, beforeEach } from 'vitest';

import { clearAccessToken, getAccessToken } from './auth';
import {
	completeLogin,
	getPendingEmail,
	postLoginPath,
	setPendingEmail,
	type SubmitLoginFn
} from './login';

describe('login flow', () => {
	beforeEach(() => {
		clearAccessToken();
		setPendingEmail('');
	});

	it('TC-001-5 stores the token through the central auth module and redirects home', async () => {
		const submitLogin: SubmitLoginFn = async (req) => {
			expect(req).toEqual({ email: 'admin@localhost', code: '123456' });
			return { access_token: 'issued-token' };
		};
		const navigatedTo: string[] = [];
		const navigate = async (path: string) => {
			navigatedTo.push(path);
		};

		setPendingEmail('admin@localhost');
		await completeLogin('admin@localhost', '123456', navigate, submitLogin);

		expect(getAccessToken()).toBe('issued-token');
		expect(getPendingEmail()).toBe('');
		expect(navigatedTo).toEqual([postLoginPath]);
	});

	it('TC-001-4 keeps no token and does not navigate when the code is rejected', async () => {
		const submitLogin: SubmitLoginFn = async () => {
			throw new Error('unauthenticated');
		};
		const navigatedTo: string[] = [];
		const navigate = async (path: string) => {
			navigatedTo.push(path);
		};

		await expect(
			completeLogin('admin@localhost', '000000', navigate, submitLogin)
		).rejects.toThrow();
		expect(getAccessToken()).toBeNull();
		expect(navigatedTo).toEqual([]);
	});
});
