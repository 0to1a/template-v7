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

// null = the user closed the popup. The Firebase session is thrown away after
// the exchange; our tokens live only in auth.ts
export async function getGoogleIdToken(config: FirebaseConfig): Promise<string | null> {
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
	try {
		const { user } = await signInWithPopup(auth, new GoogleAuthProvider());
		return await user.getIdToken();
	} catch (err) {
		const code = (err as { code?: string } | null)?.code;
		if (code === 'auth/popup-closed-by-user' || code === 'auth/cancelled-popup-request')
			return null;
		throw err;
	} finally {
		await signOut(auth);
	}
}
