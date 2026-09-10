import type { ReactNode } from 'react';
import { Link } from '@tanstack/react-router';
import { Check, Layers3 } from 'lucide-react';

type AuthShellProps = {
	eyebrow: string;
	title: string;
	description: ReactNode;
	children: ReactNode;
};

export function AuthShell({ eyebrow, title, description, children }: AuthShellProps) {
	return (
		<main className="grid min-h-screen bg-background lg:grid-cols-[minmax(0,1fr)_minmax(28rem,0.86fr)]">
			<section className="flex min-h-screen flex-col px-5 py-6 sm:px-10 lg:px-14 lg:py-10">
				<Link to="/" className="flex w-fit items-center gap-2.5 font-semibold tracking-tight">
					<span className="flex size-9 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-sm">
						<Layers3 className="size-4.5" aria-hidden="true" />
					</span>
					Template
				</Link>

				<div className="mx-auto flex w-full max-w-[25rem] flex-1 flex-col justify-center py-16">
					<p className="mb-3 text-xs font-semibold tracking-[0.18em] text-muted-foreground uppercase">
						{eyebrow}
					</p>
					<h1 className="text-3xl font-semibold tracking-[-0.035em] sm:text-4xl">{title}</h1>
					<div className="mt-3 text-[0.95rem] leading-6 text-muted-foreground">{description}</div>
					<div className="mt-8">{children}</div>
				</div>

				<p className="text-center text-xs text-muted-foreground lg:text-left">
					Protected by secure, passwordless authentication.
				</p>
			</section>

			<aside className="relative m-3 hidden overflow-hidden rounded-[2rem] bg-zinc-950 p-12 text-white lg:flex lg:flex-col lg:justify-between">
				<div className="absolute inset-0 bg-[radial-gradient(circle_at_70%_10%,rgba(255,255,255,0.13),transparent_32%),radial-gradient(circle_at_10%_85%,rgba(120,119,198,0.25),transparent_38%)]" />
				<div className="relative flex items-center gap-2 text-sm text-zinc-300">
					<span className="size-2 rounded-full bg-emerald-400 shadow-[0_0_18px_rgba(52,211,153,0.8)]" />
					All systems operational
				</div>

				<div className="relative max-w-lg">
					<p className="text-3xl leading-[1.18] font-medium tracking-[-0.035em] text-balance">
						Everything you need to move from an idea to a product people love.
					</p>
					<div className="mt-8 grid gap-3 text-sm text-zinc-300 sm:grid-cols-2">
						{[
							'Production-ready foundation',
							'Secure by default',
							'Built for fast teams',
							'Simple to customize'
						].map((item) => (
							<div key={item} className="flex items-center gap-2.5">
								<span className="flex size-5 items-center justify-center rounded-full bg-white/10">
									<Check className="size-3" aria-hidden="true" />
								</span>
								{item}
							</div>
						))}
					</div>
				</div>

				<p className="relative text-xs text-zinc-500">Template v6 · Built for modern products</p>
			</aside>
		</main>
	);
}
