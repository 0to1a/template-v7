import { createFileRoute, Outlet, redirect } from '@tanstack/react-router';

import { isAuthenticated } from '@/lib/auth';

// Pathless layout: no URL segment, just the auth guard
export const Route = createFileRoute('/_authenticated')({
	beforeLoad: () => {
		if (!isAuthenticated()) throw redirect({ to: '/login' });
	},
	component: () => <Outlet />
});
