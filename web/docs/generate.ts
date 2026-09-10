// make docs: renders routes from fixtures to routes.json
import { chromium, type Browser, type Page } from 'playwright';
import { createServer, type Server } from 'node:http';
import { readFile } from 'node:fs/promises';
import { existsSync, mkdirSync, readFileSync, readdirSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, extname, join, relative, sep } from 'node:path';

// Mirrors vite proxy; matches fixture APIs, not backend
const API_PATH_PATTERN = /^\/(?:api\/|health$)/;

// Must match ACCESS_TOKEN_KEY in web/src/lib/auth.ts
const ACCESS_TOKEN_KEY = 'template-v7.access_token';
const DUMMY_ACCESS_TOKEN = 'make-docs-dummy-token';

const WEB_DIR = join(import.meta.dir, '..');
const REPO_ROOT = join(WEB_DIR, '..');
const ROUTES_DIR = join(WEB_DIR, 'src/routes');
const DIST_DIR = join(WEB_DIR, 'dist');
const FIXTURES_DIR = join(import.meta.dir, 'fixtures');
const AUTH_FIXTURE_PATH = join(FIXTURES_DIR, '_auth.json');
const TEMPLATES_DIR = join(REPO_ROOT, 'internal/template');
const DOCS_DIR = join(REPO_ROOT, 'docs');
const SCREENSHOTS_DIR = join(DOCS_DIR, 'screenshots');
const ROUTES_JSON_PATH = join(DOCS_DIR, 'routes.json');
const TEMPLATES_JSON_PATH = join(DOCS_DIR, 'templates.json');
const CATALOG_HTML_PATH = join(DOCS_DIR, 'route-catalog.html');
const PORT = 4173;
const VIEWPORT = { width: 1280, height: 800 };
// Narrower than VIEWPORT: emails are single-column, so a wide capture wastes most
// of the frame as side padding and leaves the fixed-size thumbnail box mostly empty
const TEMPLATE_VIEWPORT = { width: 600, height: 800 };

interface GenerateSpec {
	count: number;
	seed: number;
	template?: unknown;
}

interface ErrorSpec {
	code: string;
	message?: string;
}

interface StateFixture {
	api?: Record<string, unknown>;
	// Some routes only reachable via in-app navigation
	visit?: string;
	fill?: Record<string, string>;
	submit?: string;
	// Extra seed beyond the auth-required token seeding
	localStorage?: Record<string, string>;
}

interface RouteFixture {
	title: string;
	description: string;
	auth: 'public' | 'required';
	// Concrete values for $segment placeholders in the route path (e.g. { postId: "123" })
	params?: Record<string, string>;
	states: Record<string, StateFixture>;
}

interface AuthFixture {
	api?: Record<string, unknown>;
}

interface DiscoveredRoute {
	urlPath: string;
	sourceFile: string; // repo-root-relative, forward-slashed
}

function toPosix(path: string): string {
	return path.split(sep).join('/');
}

function fixtureFileFor(urlPath: string): string {
	const relativePath = urlPath === '/' ? 'index' : urlPath.slice(1);
	return join(FIXTURES_DIR, `${relativePath}.json`);
}

function screenshotDirFor(urlPath: string): string {
	const relativePath = urlPath === '/' ? 'index' : urlPath.slice(1);
	return join(SCREENSHOTS_DIR, relativePath);
}

// _dir/_file = layout, no segment; $param kept literal, resolved later via fixture.params
function discoverRoutes(dir: string, urlSegments: string[]): DiscoveredRoute[] {
	const entries = readdirSync(dir, { withFileTypes: true });
	const routes: DiscoveredRoute[] = [];

	for (const entry of entries) {
		if (entry.isDirectory()) {
			const nextSegments = entry.name.startsWith('_') ? urlSegments : [...urlSegments, entry.name];
			routes.push(...discoverRoutes(join(dir, entry.name), nextSegments));
			continue;
		}

		if (!entry.isFile() || !entry.name.endsWith('.tsx') || entry.name === '__root.tsx') continue;

		const base = entry.name.slice(0, -'.tsx'.length);
		if (base.startsWith('_')) continue; // pathless layout

		const segments = base === 'index' ? urlSegments : [...urlSegments, base];
		const urlPath = segments.length === 0 ? '/' : `/${segments.join('/')}`;
		routes.push({ urlPath, sourceFile: toPosix(relative(REPO_ROOT, join(dir, entry.name))) });
	}

	return routes;
}

// Substitutes $segment placeholders (e.g. "$postId") with values from fixture.params
function resolveDynamicPath(urlPath: string, params: Record<string, string> | undefined): string {
	return urlPath.replace(/\$([A-Za-z0-9_]+)/g, (match, name: string) => {
		const value = params?.[name];
		if (value === undefined) {
			throw new Error(
				`make docs: route "${urlPath}" has dynamic segment "${match}" but its fixture is missing params.${name}`
			);
		}
		return value;
	});
}

// Serves bun run build output, with SPA fallback
const MIME_TYPES: Record<string, string> = {
	'.html': 'text/html',
	'.js': 'text/javascript',
	'.css': 'text/css',
	'.svg': 'image/svg+xml',
	'.json': 'application/json',
	'.png': 'image/png',
	'.ico': 'image/x-icon',
	'.woff': 'font/woff',
	'.woff2': 'font/woff2'
};

function serveDist(root: string, port: number): Promise<Server> {
	const server = createServer((req, res) => {
		void (async () => {
			const url = new URL(req.url ?? '/', 'http://localhost');
			const requested = join(root, decodeURIComponent(url.pathname));
			const candidates = requested.endsWith(sep) ? [join(requested, 'index.html')] : [requested];
			for (const candidate of candidates) {
				try {
					const body = await readFile(candidate);
					res.writeHead(200, {
						'Content-Type': MIME_TYPES[extname(candidate)] ?? 'application/octet-stream'
					});
					res.end(body);
					return;
				} catch {
					// fall through, then SPA fallback
				}
			}
			const body = await readFile(join(root, 'index.html'));
			res.writeHead(200, { 'Content-Type': 'text/html' });
			res.end(body);
		})();
	});
	return new Promise((resolve) => server.listen(port, () => resolve(server)));
}

// $generate: seeded PRNG builds deterministic mock arrays
function seededRandom(seed: number): () => number {
	let state = seed >>> 0 || 0x9e3779b9;
	return () => {
		state ^= state << 13;
		state >>>= 0;
		state ^= state >>> 17;
		state ^= state << 5;
		state >>>= 0;
		return state / 0xffffffff;
	};
}

function instantiateTemplate(template: unknown, index: number, rand: () => number): unknown {
	if (typeof template === 'string') {
		return template
			.replace(/\{\{index\}\}/g, String(index + 1))
			.replace(/\{\{seed\}\}/g, () => String(Math.floor(rand() * 1_000_000)));
	}
	if (Array.isArray(template)) {
		return template.map((item) => instantiateTemplate(item, index, rand));
	}
	if (template !== null && typeof template === 'object') {
		const out: Record<string, unknown> = {};
		for (const [key, value] of Object.entries(template as Record<string, unknown>)) {
			out[key] = instantiateTemplate(value, index, rand);
		}
		return out;
	}
	return template;
}

function isGenerateNode(value: unknown): value is { $generate: GenerateSpec } {
	return (
		typeof value === 'object' &&
		value !== null &&
		!Array.isArray(value) &&
		typeof (value as Record<string, unknown>).$generate === 'object'
	);
}

function resolveGenerators(value: unknown): unknown {
	if (Array.isArray(value)) return value.map(resolveGenerators);
	if (value !== null && typeof value === 'object') {
		if (isGenerateNode(value)) {
			const spec = value.$generate;
			const rand = seededRandom(spec.seed);
			const items: unknown[] = [];
			for (let i = 0; i < spec.count; i += 1) {
				items.push(
					spec.template !== undefined
						? instantiateTemplate(spec.template, i, rand)
						: { id: i + 1, seed: Math.floor(rand() * 1_000_000) }
				);
			}
			return items;
		}
		const out: Record<string, unknown> = {};
		for (const [key, val] of Object.entries(value as Record<string, unknown>)) {
			out[key] = resolveGenerators(val);
		}
		return out;
	}
	return value;
}

// Mirrors apic's Code table (numbering per Google's canonical status codes)
const ERROR_CODES: Record<string, { code: number; status: string; http: number }> = {
	canceled: { code: 1, status: 'CANCELED', http: 499 },
	unknown: { code: 2, status: 'UNKNOWN', http: 500 },
	invalid_argument: { code: 3, status: 'INVALID_ARGUMENT', http: 400 },
	deadline_exceeded: { code: 4, status: 'DEADLINE_EXCEEDED', http: 504 },
	not_found: { code: 5, status: 'NOT_FOUND', http: 404 },
	already_exists: { code: 6, status: 'ALREADY_EXISTS', http: 409 },
	permission_denied: { code: 7, status: 'PERMISSION_DENIED', http: 403 },
	resource_exhausted: { code: 8, status: 'RESOURCE_EXHAUSTED', http: 429 },
	failed_precondition: { code: 9, status: 'FAILED_PRECONDITION', http: 400 },
	aborted: { code: 10, status: 'ABORTED', http: 409 },
	out_of_range: { code: 11, status: 'OUT_OF_RANGE', http: 400 },
	unimplemented: { code: 12, status: 'UNIMPLEMENTED', http: 501 },
	internal: { code: 13, status: 'INTERNAL', http: 500 },
	unavailable: { code: 14, status: 'UNAVAILABLE', http: 503 },
	data_loss: { code: 15, status: 'DATA_LOSS', http: 500 },
	unauthenticated: { code: 16, status: 'UNAUTHENTICATED', http: 401 }
};

function extractErrorSpec(mock: unknown): ErrorSpec | undefined {
	if (mock === null || typeof mock !== 'object' || Array.isArray(mock)) return undefined;
	const value = (mock as Record<string, unknown>).$error;
	if (value === undefined) return undefined;
	if (typeof value === 'string') return { code: value };
	if (
		typeof value === 'object' &&
		value !== null &&
		typeof (value as ErrorSpec).code === 'string'
	) {
		return value as ErrorSpec;
	}
	return { code: 'unknown' };
}

// Auto-derived from fixture; never hardcoded per route
function summarizeApiMock(apiKey: string, resolvedMock: unknown): string {
	if (resolvedMock === null) return `${apiKey} returns 204 No Content`;

	const errorSpec = extractErrorSpec(resolvedMock);
	if (errorSpec) return `${apiKey} fails with a "${errorSpec.code}" error`;

	if (resolvedMock !== null && typeof resolvedMock === 'object' && !Array.isArray(resolvedMock)) {
		const entries = Object.entries(resolvedMock as Record<string, unknown>);
		const arrayEntry = entries.find(([, v]) => Array.isArray(v));
		if (arrayEntry) {
			const [field, arr] = arrayEntry as [string, unknown[]];
			return `${apiKey} returns ${arr.length} ${field} item${arr.length === 1 ? '' : 's'}`;
		}
		if (entries.length === 0) return `${apiKey} returns an empty response`;
		return `${apiKey} returns mock data (${entries.map(([k]) => k).join(', ')})`;
	}
	return `${apiKey} returns a mock response`;
}

function mockSummaryFor(state: StateFixture, mergedApi: Record<string, unknown>): string {
	const parts: string[] = [];

	const flowSteps: string[] = [];
	if (state.visit) flowSteps.push(`visits ${state.visit}`);
	if (state.fill) flowSteps.push(`fills ${Object.keys(state.fill).length} field(s)`);
	if (state.submit) flowSteps.push('submits the form');
	if (flowSteps.length > 0) parts.push(flowSteps.join(', then '));

	if (state.localStorage) {
		parts.push(`seeds localStorage key(s) ${Object.keys(state.localStorage).sort().join(', ')}`);
	}

	const apiKeys = Object.keys(state.api ?? {}).sort();
	for (const key of apiKeys) {
		parts.push(summarizeApiMock(key, resolveGenerators(mergedApi[key])));
	}

	if (parts.length === 0) return 'No API calls; static view with no mock data.';
	const sentence = parts.join('; ');
	return `${sentence.charAt(0).toUpperCase()}${sentence.slice(1)}.`;
}

interface StateEntry {
	name: string;
	screenshot: string;
	mockSummary: string;
}

interface RouteEntry {
	path: string;
	title: string;
	description: string;
	auth: 'public' | 'required';
	sourceFile: string;
	fixture: string;
	apis: string[];
	states: StateEntry[];
}

function readJson<T>(path: string): T {
	return JSON.parse(readFileSync(path, 'utf8')) as T;
}

async function renderState(
	browser: Browser,
	route: DiscoveredRoute,
	fixture: RouteFixture,
	stateName: string,
	state: StateFixture,
	authFixture: AuthFixture
): Promise<StateEntry> {
	const mergedApi: Record<string, unknown> = {
		...(fixture.auth === 'required' ? (authFixture.api ?? {}) : {}),
		...(state.api ?? {})
	};
	const resolvedUrlPath = resolveDynamicPath(route.urlPath, fixture.params);

	const context = await browser.newContext({
		viewport: VIEWPORT,
		reducedMotion: 'reduce',
		colorScheme: 'light'
	});
	const page: Page = await context.newPage();

	await page.route(
		(url) => API_PATH_PATTERN.test(url.pathname),
		async (routeHandle) => {
			const request = routeHandle.request();
			const apiKey = `${request.method()} ${new URL(request.url()).pathname}`;
			const mock = mergedApi[apiKey];
			if (mock === undefined) {
				console.warn(
					`make docs: no fixture api response for ${apiKey} (state "${stateName}" of ${route.urlPath})`
				);
				const spec = ERROR_CODES.unimplemented;
				await routeHandle.fulfill({
					status: spec.http,
					contentType: 'application/json',
					body: JSON.stringify({
						code: spec.code,
						status: spec.status,
						message: 'no fixture response'
					})
				});
				return;
			}

			const errorSpec = extractErrorSpec(mock);
			if (errorSpec) {
				const spec = ERROR_CODES[errorSpec.code] ?? ERROR_CODES.unknown;
				await routeHandle.fulfill({
					status: spec.http,
					contentType: 'application/json',
					body: JSON.stringify({
						code: spec.code,
						status: spec.status,
						message: errorSpec.message ?? `mock ${errorSpec.code} error`
					})
				});
				return;
			}

			// null fixture value = 204 No Content (routes without resp in apic.yaml)
			if (mock === null) {
				await routeHandle.fulfill({ status: 204 });
				return;
			}

			await routeHandle.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ code: 0, status: 'OK', data: resolveGenerators(mock) })
			});
		}
	);

	if (fixture.auth === 'required') {
		await context.addInitScript(
			(seed: { key: string; token: string }) => {
				window.localStorage.setItem(seed.key, seed.token);
			},
			{ key: ACCESS_TOKEN_KEY, token: DUMMY_ACCESS_TOKEN }
		);
	}
	if (state.localStorage) {
		await context.addInitScript((seed: Record<string, string>) => {
			for (const [key, value] of Object.entries(seed)) {
				window.localStorage.setItem(key, value);
			}
		}, state.localStorage);
	}

	await page.goto(`http://localhost:${PORT}${state.visit ?? resolvedUrlPath}`, {
		waitUntil: 'networkidle'
	});
	await page.evaluate(() => document.fonts.ready);

	if (state.fill) {
		for (const [selector, value] of Object.entries(state.fill)) {
			await page.fill(selector, value);
		}
	}
	if (state.submit) {
		await page.click(state.submit);
		await page.waitForURL(`**${resolvedUrlPath}`);
		await page.waitForLoadState('networkidle');
		await page.evaluate(() => document.fonts.ready);
	}

	const screenshotAbsPath = join(screenshotDirFor(route.urlPath), `${stateName}.png`);
	mkdirSync(dirname(screenshotAbsPath), { recursive: true });
	await page.screenshot({ path: screenshotAbsPath, fullPage: true });

	await context.close();

	return {
		name: stateName,
		screenshot: toPosix(relative(REPO_ROOT, screenshotAbsPath)),
		mockSummary: mockSummaryFor(state, mergedApi)
	};
}

// Sorted everywhere for byte-identical repeat runs
function writeRoutesJson(routes: RouteEntry[]): void {
	const manifest = {
		generatedBy: 'make docs',
		routes
	};
	mkdirSync(DOCS_DIR, { recursive: true });
	writeFileSync(ROUTES_JSON_PATH, `${JSON.stringify(manifest, null, '\t')}\n`);
}

interface TemplateEntry {
	name: string;
	path: string; // repo-root-relative, forward-slashed
	variables: string[]; // kept as literal "{{.Field}}" placeholders
	screenshot: string; // repo-root-relative, forward-slashed
}

const VARIABLE_PATTERN = /\{\{\s*\.(\w+)\s*\}\}/g;

// {{.Field}} placeholders, deduped and sorted, kept in mustache syntax
function extractVariables(source: string): string[] {
	const found = new Set<string>();
	for (const match of source.matchAll(VARIABLE_PATTERN)) {
		found.add(`{{.${match[1]}}}`);
	}
	return [...found].sort();
}

// Best-effort readable stand-ins so previews aren't just "{{.Code}}" text
function sampleValueFor(field: string): string {
	if (/code/i.test(field)) return '123456';
	if (/email/i.test(field)) return 'user@example.com';
	if (/name/i.test(field)) return 'Jane Doe';
	if (/url|link/i.test(field)) return 'https://example.com';
	return `Sample ${field}`;
}

function fillTemplateSample(source: string): string {
	return source.replace(VARIABLE_PATTERN, (_match, field: string) => sampleValueFor(field));
}

interface DiscoveredTemplate {
	name: string;
	path: string;
	absPath: string;
	source: string;
}

// internal/template/*.html; each file is one named template
function discoverTemplates(): DiscoveredTemplate[] {
	if (!existsSync(TEMPLATES_DIR)) return [];
	return readdirSync(TEMPLATES_DIR, { withFileTypes: true })
		.filter((entry) => entry.isFile() && entry.name.endsWith('.html'))
		.map((entry) => {
			const absPath = join(TEMPLATES_DIR, entry.name);
			return {
				name: entry.name.slice(0, -'.html'.length),
				path: toPosix(relative(REPO_ROOT, absPath)),
				absPath,
				source: readFileSync(absPath, 'utf8')
			};
		})
		.sort((a, b) => a.name.localeCompare(b.name));
}

// Renders sample-filled HTML in a real page so the email preview is trustworthy
async function renderTemplatePreview(
	browser: Browser,
	template: DiscoveredTemplate
): Promise<TemplateEntry> {
	const context = await browser.newContext({ viewport: TEMPLATE_VIEWPORT, colorScheme: 'light' });
	const page: Page = await context.newPage();
	await page.setContent(fillTemplateSample(template.source), { waitUntil: 'networkidle' });
	await page.evaluate(() => document.fonts.ready);

	// html/body paint the full viewport regardless of content height; clip to the real content box
	const contentBox = await page.evaluate(() => {
		const rect = document.body.getBoundingClientRect();
		return { width: Math.ceil(rect.width), height: Math.ceil(rect.height) };
	});

	const screenshotAbsPath = join(SCREENSHOTS_DIR, 'templates', `${template.name}.png`);
	mkdirSync(dirname(screenshotAbsPath), { recursive: true });
	await page.screenshot({
		path: screenshotAbsPath,
		clip: { x: 0, y: 0, width: contentBox.width, height: contentBox.height }
	});

	await context.close();

	return {
		name: template.name,
		path: template.path,
		variables: extractVariables(template.source),
		screenshot: toPosix(relative(REPO_ROOT, screenshotAbsPath))
	};
}

function writeTemplatesJson(templates: TemplateEntry[]): void {
	const manifest = {
		generatedBy: 'make docs',
		templates
	};
	mkdirSync(DOCS_DIR, { recursive: true });
	writeFileSync(TEMPLATES_JSON_PATH, `${JSON.stringify(manifest, null, '\t')}\n`);
}

// img src must be relative to docs/, not repo root
function toDocsRelative(repoRootRelativePath: string): string {
	return toPosix(relative(DOCS_DIR, join(REPO_ROOT, repoRootRelativePath)));
}

// Styled after v5's catalog; only visual language reused
function renderCatalogHtml(routes: RouteEntry[], templates: TemplateEntry[]): string {
	const catalogRoutes = routes.map((route) => ({
		...route,
		states: route.states.map((state) => ({
			...state,
			screenshot: toDocsRelative(state.screenshot)
		}))
	}));
	const routesJson = JSON.stringify(catalogRoutes).replace(/</g, '\\u003c');
	const catalogTemplates = templates.map((template) => ({
		...template,
		screenshot: toDocsRelative(template.screenshot)
	}));
	const templatesJson = JSON.stringify(catalogTemplates).replace(/</g, '\\u003c');
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Route Catalog</title>
<style>
  * { box-sizing: border-box; }
  body {
    margin: 0;
    font: 14px/1.6 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    color: #1a1a1a;
    background: #fff;
  }
  #app { display: flex; min-height: 100vh; }

  aside {
    width: 260px;
    flex: none;
    border-right: 1px solid #e2e2e2;
    padding: 16px;
    background: #fafafa;
    position: sticky;
    top: 0;
    height: 100vh;
    overflow-y: auto;
  }
  aside h1 { font-size: 13px; text-transform: uppercase; letter-spacing: .08em; color: #777; margin: 0 0 12px; }
  #search {
    width: 100%; padding: 6px 8px; margin-bottom: 14px;
    border: 1px solid #d4d4d4; border-radius: 4px; font: inherit; font-size: 13px;
  }
  .group-label { font-size: 11px; text-transform: uppercase; letter-spacing: .06em; color: #999; margin: 16px 0 6px; }
  aside a {
    display: block; padding: 5px 8px; border-radius: 4px;
    color: #333; text-decoration: none; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12.5px;
  }
  aside a:hover { background: #ececec; }
  aside a.active { background: #1a1a1a; color: #fff; }

  main { flex: 1; padding: 32px 40px 80px; max-width: 1100px; }
  .route { padding-bottom: 48px; margin-bottom: 48px; border-bottom: 1px solid #eee; scroll-margin-top: 24px; }
  .route:last-child { border-bottom: 0; }
  .route-path { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 20px; margin: 0 0 4px; }
  .route-name { color: #666; margin: 0 0 12px; }
  .meta { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 12px; }
  .tag {
    font-size: 11px; padding: 2px 8px; border-radius: 3px;
    background: #eee; color: #444; font-family: ui-monospace, Menlo, monospace;
  }
  .tag.auth { background: #ffe9c7; color: #7a4c00; }
  .tag.public { background: #dff0d8; color: #2d5a2d; }
  .desc { color: #444; margin: 0 0 20px; max-width: 65ch; }

  dl.kv { margin: 0 0 20px; display: grid; grid-template-columns: 120px 1fr; gap: 4px 16px; font-size: 13px; }
  dl.kv dt { color: #888; }
  dl.kv dd { margin: 0; font-family: ui-monospace, Menlo, monospace; font-size: 12.5px; }

  .states { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 20px; }
  .state { border: 1px solid #e2e2e2; border-radius: 6px; overflow: hidden; }
  .state figure { margin: 0; background: #f4f4f4; aspect-ratio: 16/10; display: flex; align-items: center; justify-content: center; position: relative; }
  .state img { width: 100%; height: 100%; object-fit: cover; object-position: top; display: block; cursor: zoom-in; }
  /* Templates vary wildly in aspect ratio; fit the whole email instead of cropping it */
  .state.template-card figure { aspect-ratio: 16/10; }
  .state.template-card img { object-fit: contain; }
  .placeholder { color: #aaa; font-size: 12px; font-family: ui-monospace, Menlo, monospace; text-align: center; padding: 12px; }
  .state-body { padding: 10px 12px; }
  .state-name { font-weight: 600; font-size: 13px; margin: 0 0 4px; }
  .state-note { color: #666; font-size: 12.5px; margin: 0; }

  /* Lightbox: fullPage screenshots can be much taller than the thumbnail crop */
  .lightbox {
    position: fixed; inset: 0; z-index: 1000;
    display: none; align-items: flex-start; justify-content: center;
    background: rgba(0, 0, 0, .85); overflow: auto; padding: 40px; cursor: zoom-out;
  }
  .lightbox.open { display: flex; }
  .lightbox img { max-width: 90vw; cursor: zoom-in; box-shadow: 0 4px 24px rgba(0, 0, 0, .5); }
  .lightbox.zoomed { align-items: flex-start; justify-content: flex-start; }
  .lightbox.zoomed img { max-width: none; cursor: zoom-out; }
  .lightbox-close {
    position: fixed; top: 16px; right: 20px; color: #fff; font-size: 28px;
    line-height: 1; cursor: pointer; opacity: .8;
  }
  .lightbox-close:hover { opacity: 1; }

  .empty { color: #999; font-style: italic; }
  @media (max-width: 720px) {
    #app { flex-direction: column; }
    aside { width: auto; height: auto; position: static; border-right: 0; border-bottom: 1px solid #e2e2e2; }
    main { padding: 24px 20px 60px; }
  }
</style>
</head>
<body>

<!-- Generated by make docs - do not hand-edit -->
<script>
const ROUTES = ${routesJson};
const TEMPLATES = ${templatesJson};
</script>

<div id="app">
  <aside>
    <h1>Route Catalog</h1>
    <input id="search" type="search" placeholder="Search routes..." autocomplete="off">
    <nav id="nav"></nav>
    <div class="group-label" id="templates-nav-label" style="display:none">Templates</div>
    <nav id="templates-nav"></nav>
  </aside>
  <main id="main"></main>
</div>

<div id="lightbox" class="lightbox">
  <span class="lightbox-close">&times;</span>
  <img id="lightbox-img" src="" alt="">
</div>

<script>
const slug = p => p.replace(/[^a-z0-9]+/gi, "-").replace(/^-|-$/g, "").toLowerCase() || "root";
const esc = s => String(s).replace(/[&<>"]/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));

function render(list) {
  const groups = {};
  list.forEach(r => (groups[r.auth === "required" ? "Protected" : "Public"] ||= []).push(r));
  document.getElementById("nav").innerHTML = Object.entries(groups).map(([g, rs]) => \`
    <div class="group-label">\${esc(g)}</div>
    \${rs.map(r => \`<a href="#\${slug(r.path)}">\${esc(r.path)}</a>\`).join("")}
  \`).join("");

  document.getElementById("main").innerHTML = list.length ? list.map(r => \`
    <section class="route" id="\${slug(r.path)}">
      <h2 class="route-path">\${esc(r.path)}</h2>
      <p class="route-name">\${esc(r.title || "")}</p>
      <div class="meta">
        <span class="tag \${r.auth === "required" ? "auth" : "public"}">\${r.auth === "required" ? "auth: required" : "public"}</span>
        <span class="tag">\${r.states.length} state\${r.states.length === 1 ? "" : "s"}</span>
      </div>
      <p class="desc">\${esc(r.description || "")}</p>
      <dl class="kv">
        <dt>Source</dt><dd>\${esc(r.sourceFile || "-")}</dd>
        <dt>Fixture</dt><dd>\${esc(r.fixture || "-")}</dd>
        <dt>APIs</dt><dd>\${esc((r.apis || []).join(", ") || "-")}</dd>
      </dl>
      <div class="states">
        \${r.states.map(s => \`
          <div class="state">
            <figure>
              <img src="\${esc(s.screenshot)}" alt="\${esc(r.path)} - \${esc(s.name)}" loading="lazy"
                   onerror="this.style.display='none'; this.nextElementSibling.style.display='flex'">
              <div class="placeholder" style="display:none; position:absolute; inset:0; align-items:center; justify-content:center;">no screenshot<br>\${esc(s.name)}</div>
            </figure>
            <div class="state-body">
              <p class="state-name">\${esc(s.name)}</p>
              <p class="state-note">\${esc(s.mockSummary || "")}</p>
            </div>
          </div>\`).join("")}
      </div>
    </section>
  \`).join("") : \`<p class="empty">No routes match.</p>\`;
  document.getElementById("main").innerHTML += TEMPLATES_HTML;
}

const TEMPLATES_HTML = TEMPLATES.length ? \`
  <div class="group-label" style="font-size:13px; margin-top:32px;">Templates</div>
  \${TEMPLATES.map(t => \`
    <section class="route" id="\${slug("template-" + t.name)}">
      <h2 class="route-path">\${esc(t.name)}</h2>
      <div class="meta">
        <span class="tag">\${esc(t.path)}</span>
      </div>
      <dl class="kv">
        <dt>Variables</dt><dd>\${esc(t.variables.join(", ") || "-")}</dd>
      </dl>
      <div class="states">
        <div class="state template-card">
          <figure>
            <img src="\${esc(t.screenshot)}" alt="\${esc(t.name)} preview" loading="lazy"
                 onerror="this.style.display='none'; this.nextElementSibling.style.display='flex'">
            <div class="placeholder" style="display:none; position:absolute; inset:0; align-items:center; justify-content:center;">no screenshot</div>
          </figure>
        </div>
      </div>
    </section>\`).join("")}
\` : "";

document.getElementById("templates-nav-label").style.display = TEMPLATES.length ? "" : "none";
document.getElementById("templates-nav").innerHTML = TEMPLATES.map(t =>
  \`<a href="#\${slug("template-" + t.name)}">\${esc(t.name)}</a>\`
).join("");

render(ROUTES);

document.getElementById("search").addEventListener("input", e => {
  const q = e.target.value.toLowerCase().trim();
  render(!q ? ROUTES : ROUTES.filter(r =>
    (r.path + " " + (r.title || "") + " " + (r.description || "")).toLowerCase().includes(q)
  ));
});

const observer = new IntersectionObserver(entries => {
  entries.forEach(en => {
    if (!en.isIntersecting) return;
    document.querySelectorAll("#nav a, #templates-nav a").forEach(a =>
      a.classList.toggle("active", a.getAttribute("href") === "#" + en.target.id));
  });
}, { rootMargin: "-10% 0px -80% 0px" });
new MutationObserver(() => document.querySelectorAll(".route").forEach(el => observer.observe(el)))
  .observe(document.getElementById("main"), { childList: true });
document.querySelectorAll(".route").forEach(el => observer.observe(el));

const lightbox = document.getElementById("lightbox");
const lightboxImg = document.getElementById("lightbox-img");

function openLightbox(src, alt) {
  lightboxImg.src = src;
  lightboxImg.alt = alt;
  lightbox.classList.remove("zoomed");
  lightbox.classList.add("open");
}
function closeLightbox() {
  lightbox.classList.remove("open", "zoomed");
  lightboxImg.src = "";
}

document.getElementById("main").addEventListener("click", e => {
  const img = e.target.closest("figure img");
  if (!img) return;
  openLightbox(img.src, img.alt);
});
lightbox.addEventListener("click", e => {
  if (e.target === lightboxImg) {
    lightbox.classList.toggle("zoomed");
    return;
  }
  closeLightbox();
});
document.addEventListener("keydown", e => {
  if (e.key === "Escape") closeLightbox();
});
</script>
</body>
</html>
`;
}

function writeCatalogHtml(routes: RouteEntry[], templates: TemplateEntry[]): void {
	mkdirSync(DOCS_DIR, { recursive: true });
	writeFileSync(CATALOG_HTML_PATH, renderCatalogHtml(routes, templates));
}

async function main(): Promise<void> {
	const discoveredTemplates = discoverTemplates();

	const routes = discoverRoutes(ROUTES_DIR, []).sort((a, b) => a.urlPath.localeCompare(b.urlPath));

	const missing = routes.filter((route) => !existsSync(fixtureFileFor(route.urlPath)));
	if (missing.length > 0) {
		console.error('make docs: missing fixture file for route(s):');
		for (const route of missing) {
			console.error(
				`  ${route.urlPath} -> ${toPosix(relative(REPO_ROOT, fixtureFileFor(route.urlPath)))}`
			);
		}
		process.exit(1);
	}

	if (!existsSync(DIST_DIR)) {
		console.error(
			`make docs: ${toPosix(relative(REPO_ROOT, DIST_DIR))} not found; run "bun run build" first`
		);
		process.exit(1);
	}

	const authFixture: AuthFixture = existsSync(AUTH_FIXTURE_PATH)
		? readJson<AuthFixture>(AUTH_FIXTURE_PATH)
		: { api: {} };

	// Regenerated wholesale; avoids stale orphaned PNGs
	rmSync(SCREENSHOTS_DIR, { recursive: true, force: true });

	const server = await serveDist(DIST_DIR, PORT);
	const routeEntries: RouteEntry[] = [];
	const templates: TemplateEntry[] = [];
	try {
		const browser = await chromium.launch();
		try {
			for (const template of discoveredTemplates) {
				templates.push(await renderTemplatePreview(browser, template));
			}

			for (const route of routes) {
				const fixture = readJson<RouteFixture>(fixtureFileFor(route.urlPath));
				const stateNames = Object.keys(fixture.states ?? {}).sort();
				if (stateNames.length === 0) {
					throw new Error(
						`make docs: fixture ${toPosix(relative(REPO_ROOT, fixtureFileFor(route.urlPath)))} declares no states`
					);
				}

				const apiKeys = new Set<string>();
				const states: StateEntry[] = [];
				for (const stateName of stateNames) {
					const state = fixture.states[stateName];
					for (const key of Object.keys(state.api ?? {})) apiKeys.add(key);
					states.push(await renderState(browser, route, fixture, stateName, state, authFixture));
				}

				routeEntries.push({
					path: route.urlPath,
					title: fixture.title,
					description: fixture.description,
					auth: fixture.auth,
					sourceFile: route.sourceFile,
					fixture: toPosix(relative(REPO_ROOT, fixtureFileFor(route.urlPath))),
					apis: [...apiKeys].sort(),
					states
				});
			}
		} finally {
			await browser.close();
		}
	} finally {
		server.close();
	}

	writeTemplatesJson(templates);
	writeRoutesJson(routeEntries);
	writeCatalogHtml(routeEntries, templates);

	console.log(`==> make docs: wrote ${templates.length} template(s) to docs/templates.json`);
	console.log(`==> make docs: wrote ${routeEntries.length} route(s) to docs/routes.json`);
}

main().catch((err: unknown) => {
	console.error(err);
	process.exit(1);
});
