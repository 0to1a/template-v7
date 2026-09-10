import { useEffect, useState } from 'react';
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router';
import { useMutation } from '@tanstack/react-query';
import { ArrowLeft, ArrowRight, KeyRound } from 'lucide-react';

import { AuthShell } from '@/lib/components/auth-shell';
import { api } from '@/lib/client';
import { completeLogin, getPendingEmail } from '@/lib/login';
import { Button } from '@/lib/components/ui/button';
import { Input } from '@/lib/components/ui/input';
import { Label } from '@/lib/components/ui/label';

export const Route = createFileRoute('/login/otp')({
	component: OtpComponent
});

function OtpComponent() {
	const navigate = useNavigate();
	const email = getPendingEmail();

	// No email means hard refresh; restart the flow
	useEffect(() => {
		if (!email) void navigate({ to: '/login' });
	}, [email, navigate]);

	const [code, setCode] = useState('');
	const [error, setError] = useState('');
	const mutation = useMutation({ mutationFn: api.submitLogin });

	async function submit(event: React.FormEvent<HTMLFormElement>) {
		event.preventDefault();
		setError('');
		try {
			await completeLogin(
				email,
				code,
				(path) => navigate({ to: path as '/' }),
				mutation.mutateAsync
			);
		} catch {
			// Invalid email and code deliberately look the same
			setError('Invalid email or code.');
		}
	}

	return (
		<>
			<title>Enter your code · Template</title>
			<AuthShell
				eyebrow="One more step"
				title="Check your inbox"
				description={
					<>
						We sent a 6-digit login code to{' '}
						<strong className="font-medium text-foreground">{email}</strong>. The code expires in 5
						minutes.
					</>
				}
			>
				<form className="flex flex-col gap-5" onSubmit={submit}>
					<div className="flex flex-col gap-2">
						<Label htmlFor="code">One-time code</Label>
						<div className="relative">
							<KeyRound
								className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
								aria-hidden="true"
							/>
							<Input
								id="code"
								className="h-11 pl-9 font-medium tracking-[0.3em]"
								type="text"
								name="code"
								inputMode="numeric"
								autoComplete="one-time-code"
								placeholder="000000"
								maxLength={6}
								required
								aria-invalid={error ? true : undefined}
								aria-describedby={error ? 'otp-error' : undefined}
								value={code}
								onChange={(event) => setCode(event.target.value.replace(/\D/g, ''))}
							/>
						</div>
					</div>

					{error && (
						<p
							id="otp-error"
							role="alert"
							className="rounded-lg bg-destructive/8 px-3 py-2.5 text-sm text-destructive"
						>
							{error}
						</p>
					)}

					<Button className="h-11 w-full" type="submit" disabled={mutation.isPending}>
						{mutation.isPending ? 'Signing in…' : 'Log in securely'}
						{!mutation.isPending && <ArrowRight data-icon="inline-end" />}
					</Button>
					<Button
						variant="ghost"
						className="h-10 w-full text-muted-foreground"
						render={<Link to="/login" />}
					>
						<ArrowLeft data-icon="inline-start" />
						Use a different email
					</Button>
				</form>
			</AuthShell>
		</>
	);
}
