import path from 'node:path';

import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { tanstackRouter } from '@tanstack/router-plugin/vite';
import { defineConfig } from 'vite';

// Everything under /api plus /health goes to the Go server
const backendTarget = 'http://localhost:8080';

export default defineConfig({
	plugins: [
		tanstackRouter({
			target: 'react',
			autoCodeSplitting: true,
			routesDirectory: './src/routes',
			generatedRouteTree: './src/routeTree.gen.ts'
		}),
		react(),
		tailwindcss()
	],
	resolve: {
		alias: {
			'@': path.resolve(import.meta.dirname, './src')
		}
	},
	build: {
		outDir: 'dist'
	},
	server: {
		proxy: {
			'/api': backendTarget,
			'/health': backendTarget
		}
	},
	test: {
		environment: 'jsdom',
		globals: false,
		include: ['src/**/*.{test,spec}.{js,ts,jsx,tsx}']
	}
});
