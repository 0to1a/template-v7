import { createFileRoute, Outlet } from '@tanstack/react-router';

import { requireAuthBeforeLoad } from '@/lib/route-guards';

// Pathless layout: no URL segment, just auth guard
export const Route = createFileRoute('/_authenticated')({
	beforeLoad: requireAuthBeforeLoad,
	component: () => <Outlet />
});
