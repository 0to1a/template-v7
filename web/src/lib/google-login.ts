import { getApps, initializeApp } from 'firebase/app';
import {
	getAuth,
	GoogleAuthProvider,
	inMemoryPersistence,
	setPersistence,
	signInWithPopup,
	signOut
} from 'firebase/auth';

import type { FirebaseConfig } from './gen/api';

// Firebase session is thrown away after the exchange; our JWT lives only in auth.ts
export async function getGoogleIdToken(config: FirebaseConfig): Promise<string> {
	const app =
		getApps()[0] ??
		initializeApp({
			apiKey: config.api_key,
			authDomain: config.auth_domain,
			projectId: config.project_id,
			appId: config.app_id
		});
	const auth = getAuth(app);
	await setPersistence(auth, inMemoryPersistence);
	const { user } = await signInWithPopup(auth, new GoogleAuthProvider());
	try {
		return await user.getIdToken();
	} finally {
		await signOut(auth);
	}
}

// User closing the popup is not an error worth showing
export function isPopupDismissed(err: unknown): boolean {
	const code = (err as { code?: string } | null)?.code;
	return code === 'auth/popup-closed-by-user' || code === 'auth/cancelled-popup-request';
}
