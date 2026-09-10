// Shared guard for _authenticated layout routes
import { redirect } from '@tanstack/react-router';

import { isAuthenticated } from './auth';

export function requireAuthBeforeLoad(): void {
	if (!isAuthenticated()) {
		throw redirect({ to: '/login' });
	}
}
