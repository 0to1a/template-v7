import { createRootRoute, Outlet } from '@tanstack/react-router';

// Pure SPA: no server shell; React 19 hoists <title>
export const Route = createRootRoute({
	component: RootComponent
});

function RootComponent() {
	return <Outlet />;
}
