import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { RouterProvider } from '@tanstack/react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { router } from './router';
import './index.css';

const rootEl = document.getElementById('root');
if (!rootEl) {
	throw new Error('main: #root element not found');
}

createRoot(rootEl).render(
	<StrictMode>
		<QueryClientProvider client={new QueryClient()}>
			<RouterProvider router={router} />
		</QueryClientProvider>
	</StrictMode>
);
