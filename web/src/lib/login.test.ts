import { describe, expect, it, vi } from 'vitest';

import { clearTokens, getAccessToken, getRefreshToken } from './auth';
import { finishLogin, getPendingEmail, setPendingEmail } from './login';

describe('finishLogin', () => {
	it('TC-001-5 stores the tokens, forgets the pending email and goes home', async () => {
		clearTokens();
		setPendingEmail('admin@localhost');
		const navigate = vi.fn();

		await finishLogin({ access_token: 'issued-token', refresh_token: 'issued-refresh' }, navigate);

		expect(getAccessToken()).toBe('issued-token');
		expect(getRefreshToken()).toBe('issued-refresh');
		expect(getPendingEmail()).toBe('');
		expect(navigate).toHaveBeenCalledWith({ to: '/' });
	});
});
