// Stored only via central auth module, never touched here
import { setAccessToken } from './auth';

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
export type SubmitLoginFn = (req: {
	email: string;
	code: string;
}) => Promise<{ access_token: string }>;

// Injected so tests can pass a spy for navigate
export async function completeLogin(
	email: string,
	code: string,
	navigate: (path: string) => Promise<void> | void,
	submitLogin: SubmitLoginFn
): Promise<void> {
	const response = await submitLogin({ email, code });
	setAccessToken(response.access_token);
	clearPendingEmail();
	await navigate(postLoginPath);
}
