// Stored only via central auth module, never touched here
import { setTokens } from './auth';
import type { Tokens } from './gen/api';

let pendingEmail = '';

export function setPendingEmail(email: string): void {
	pendingEmail = email;
}

export function getPendingEmail(): string {
	return pendingEmail;
}

export function clearPendingEmail(): void {
	pendingEmail = '';
}

export const postLoginPath = '/' as const;

// App passes api.submitLogin; tests pass a fake
export type SubmitLoginFn = (req: { email: string; code: string }) => Promise<Tokens>;

// Injected so tests can pass a spy for navigate
export async function completeLogin(
	email: string,
	code: string,
	navigate: (path: string) => Promise<void> | void,
	submitLogin: SubmitLoginFn
): Promise<void> {
	const response = await submitLogin({ email, code });
	await finishLogin(response, navigate);
}

// Shared tail of every login method
export async function finishLogin(
	tokens: Tokens,
	navigate: (path: string) => Promise<void> | void
): Promise<void> {
	setTokens(tokens);
	clearPendingEmail();
	await navigate(postLoginPath);
}
