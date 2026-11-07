import type { useNavigate } from '@tanstack/react-router';

import { setTokens } from './auth';
import type { Tokens } from './gen/api';

// Carried from /login to /login/otp in memory only; a reload restarts the flow
let pendingEmail = '';

export function setPendingEmail(email: string): void {
	pendingEmail = email;
}

export function getPendingEmail(): string {
	return pendingEmail;
}

// Shared tail of every login method
export async function finishLogin(
	tokens: Tokens,
	navigate: ReturnType<typeof useNavigate>
): Promise<void> {
	setTokens(tokens);
	pendingEmail = '';
	await navigate({ to: '/' });
}
