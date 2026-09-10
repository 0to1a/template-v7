import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { RouterProvider } from '@tanstack/react-router';

import { AppProviders } from './lib/client';
import { router } from './router';
import './index.css';

const rootEl = document.getElementById('root');
if (!rootEl) {
	throw new Error('main: #root element not found');
}

createRoot(rootEl).render(
	<StrictMode>
		<AppProviders>
			<RouterProvider router={router} />
		</AppProviders>
	</StrictMode>
);
