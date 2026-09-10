import { useState } from 'react';
import { createFileRoute, Link } from '@tanstack/react-router';
import { ArrowRight, Check, Layers3, ShieldCheck, Sparkles, Zap } from 'lucide-react';

import { isAuthenticated, logout } from '@/lib/auth';
import { Button } from '@/lib/components/ui/button';

export const Route = createFileRoute('/')({
	component: HomeComponent
});

function HomeComponent() {
	const [authed, setAuthed] = useState(isAuthenticated);

	function handleLogout() {
		logout();
		setAuthed(false);
	}

	return (
		<>
			<title>Template · Build your next product</title>
			<div className="min-h-screen overflow-hidden bg-background text-foreground">
				<header className="relative z-20 border-b bg-background/85 backdrop-blur-xl">
					<nav
						className="mx-auto flex h-16 max-w-7xl items-center justify-between px-5 sm:px-8"
						aria-label="Main navigation"
					>
						<Link to="/" className="flex items-center gap-2.5 font-semibold tracking-tight">
							<span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
								<Layers3 className="size-4" aria-hidden="true" />
							</span>
							Template
						</Link>

						<div className="hidden items-center gap-7 text-sm text-muted-foreground md:flex">
							<a href="#features" className="transition-colors hover:text-foreground">
								Features
							</a>
							<a href="#workflow" className="transition-colors hover:text-foreground">
								How it works
							</a>
							<a href="#about" className="transition-colors hover:text-foreground">
								About
							</a>
						</div>

						{authed ? (
							<div className="flex items-center gap-2">
								<Button variant="ghost" render={<Link to="/profile" />}>
									Profile
								</Button>
								<Button type="button" variant="outline" onClick={handleLogout}>
									Log out
								</Button>
							</div>
						) : (
							<Button variant="outline" className="px-4" render={<Link to="/login" />}>
								Log in
							</Button>
						)}
					</nav>
				</header>

				<main>
					<section className="relative px-5 pt-20 pb-16 sm:px-8 sm:pt-28 lg:pt-32 lg:pb-24">
						<div className="absolute inset-x-0 top-0 -z-10 h-[38rem] bg-[radial-gradient(ellipse_55%_50%_at_50%_0%,rgba(120,119,198,0.14),transparent)]" />
						<div className="mx-auto max-w-7xl">
							<div className="mx-auto max-w-3xl text-center">
								<div className="mb-6 inline-flex items-center gap-2 rounded-full border bg-background px-3 py-1.5 text-xs font-medium shadow-sm">
									<Sparkles className="size-3.5" aria-hidden="true" />A better foundation for your
									next idea
								</div>
								<h1 className="text-5xl leading-[0.98] font-semibold tracking-[-0.06em] text-balance sm:text-6xl lg:text-7xl">
									Ship ambitious products,{' '}
									<span className="text-muted-foreground">without the busywork.</span>
								</h1>
								<p className="mx-auto mt-6 max-w-2xl text-lg leading-8 text-balance text-muted-foreground">
									A production-ready starting point that brings your product, team, and workflow
									together in one focused place.
								</p>
								<div className="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
									<Button
										size="lg"
										className="h-11 px-5"
										render={<Link to={authed ? '/profile' : '/login'} />}
									>
										{authed ? 'Open your profile' : 'Start building'}
										<ArrowRight data-icon="inline-end" />
									</Button>
									<Button
										size="lg"
										variant="outline"
										className="h-11 px-5"
										render={<a href="#features" />}
									>
										Explore features
									</Button>
								</div>
							</div>

							<div
								id="workflow"
								className="relative mx-auto mt-16 max-w-5xl rounded-[1.75rem] border bg-zinc-950 p-2 shadow-2xl shadow-zinc-950/15 sm:mt-20 sm:p-3"
							>
								<div className="overflow-hidden rounded-[1.25rem] border border-white/10 bg-zinc-900">
									<div className="flex h-11 items-center gap-1.5 border-b border-white/10 px-4">
										<span className="size-2.5 rounded-full bg-white/20" />
										<span className="size-2.5 rounded-full bg-white/20" />
										<span className="size-2.5 rounded-full bg-white/20" />
										<span className="ml-4 text-[11px] text-zinc-500">workspace / overview</span>
									</div>
									<div className="grid min-h-80 grid-cols-[4.5rem_1fr] sm:grid-cols-[12rem_1fr]">
										<div className="border-r border-white/10 p-3 sm:p-5">
											<div className="mb-6 h-6 w-6 rounded-md bg-white sm:w-20" />
											<div className="space-y-3">
												<div className="h-7 rounded-md bg-white/10" />
												<div className="h-7 rounded-md bg-white/4" />
												<div className="h-7 rounded-md bg-white/4" />
											</div>
										</div>
										<div className="p-5 sm:p-8">
											<div className="flex items-start justify-between gap-4">
												<div>
													<p className="text-xs text-zinc-500">Good morning</p>
													<p className="mt-1 text-lg font-medium text-white sm:text-xl">
														Your workspace
													</p>
												</div>
												<div className="h-8 w-20 rounded-lg bg-white text-center text-xs leading-8 text-zinc-900">
													New project
												</div>
											</div>
											<div className="mt-7 grid gap-3 sm:grid-cols-3">
												{['12 active', '84 shipped', '99.9% uptime'].map((value) => (
													<div
														key={value}
														className="rounded-xl border border-white/10 bg-white/5 p-4 text-sm text-zinc-300"
													>
														{value}
													</div>
												))}
											</div>
											<div className="mt-4 h-28 rounded-xl border border-white/10 bg-[linear-gradient(120deg,rgba(255,255,255,0.03),rgba(120,119,198,0.16))] p-4">
												<div className="h-2 w-1/3 rounded-full bg-white/15" />
												<div className="mt-3 h-2 w-2/3 rounded-full bg-white/8" />
											</div>
										</div>
									</div>
								</div>
							</div>
						</div>
					</section>

					<section id="features" className="border-y bg-muted/30 px-5 py-20 sm:px-8 lg:py-28">
						<div className="mx-auto max-w-7xl">
							<div className="max-w-xl">
								<p className="text-xs font-semibold tracking-[0.18em] text-muted-foreground uppercase">
									Made to move fast
								</p>
								<h2 className="mt-3 text-3xl font-semibold tracking-[-0.04em] sm:text-4xl">
									A solid foundation, from day one.
								</h2>
							</div>
							<div className="mt-12 grid gap-px overflow-hidden rounded-2xl border bg-border md:grid-cols-3">
								{[
									{
										icon: Zap,
										title: 'Move with speed',
										text: 'Skip repetitive setup and focus on the details that make your product unique.'
									},
									{
										icon: ShieldCheck,
										title: 'Secure by default',
										text: 'Authentication and sensible production patterns are built into the foundation.'
									},
									{
										icon: Sparkles,
										title: 'Designed to adapt',
										text: 'A clean shadcn system that is easy to extend as your product and team grow.'
									}
								].map(({ icon: Icon, title, text }) => (
									<article key={title} className="bg-background p-7 sm:p-9">
										<span className="flex size-10 items-center justify-center rounded-xl border bg-muted/50">
											<Icon className="size-4.5" aria-hidden="true" />
										</span>
										<h3 className="mt-8 text-lg font-semibold">{title}</h3>
										<p className="mt-2 leading-6 text-muted-foreground">{text}</p>
									</article>
								))}
							</div>
						</div>
					</section>

					<section id="about" className="px-5 py-20 sm:px-8 lg:py-28">
						<div className="mx-auto flex max-w-5xl flex-col items-center text-center">
							<span className="flex size-11 items-center justify-center rounded-2xl bg-primary text-primary-foreground">
								<Check className="size-5" />
							</span>
							<h2 className="mt-6 text-3xl font-semibold tracking-[-0.045em] sm:text-5xl">
								Ready to turn your idea into something real?
							</h2>
							<p className="mt-4 max-w-xl text-lg text-muted-foreground">
								Start with a foundation that is clear, reliable, and ready for what comes next.
							</p>
							<Button
								size="lg"
								className="mt-8 h-11 px-5"
								render={<Link to={authed ? '/profile' : '/login'} />}
							>
								{authed ? 'Go to profile' : 'Get started'}
								<ArrowRight data-icon="inline-end" />
							</Button>
						</div>
					</section>
				</main>

				<footer className="border-t px-5 py-7 sm:px-8">
					<div className="mx-auto flex max-w-7xl flex-col gap-3 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
						<p>© 2026 Template. Built with care.</p>
						<p>Simple. Secure. Ready to ship.</p>
					</div>
				</footer>
			</div>
		</>
	);
}
