import { useState } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useMutation } from '@tanstack/react-query';
import { ArrowRight, Mail } from 'lucide-react';

import { AuthShell } from '@/lib/components/auth-shell';
import { api } from '@/lib/client';
import { setPendingEmail } from '@/lib/login';
import { Button } from '@/lib/components/ui/button';
import { Input } from '@/lib/components/ui/input';
import { Label } from '@/lib/components/ui/label';

export const Route = createFileRoute('/login/')({
	component: LoginComponent
});

function LoginComponent() {
	const navigate = useNavigate();
	const [email, setEmail] = useState('');
	const [error, setError] = useState('');
	const mutation = useMutation({ mutationFn: (email: string) => api.requestLogin({ email }) });

	async function submit(event: React.FormEvent<HTMLFormElement>) {
		event.preventDefault();
		setError('');
		try {
			// Response is generic: never reveals account existence
			await mutation.mutateAsync(email);
			setPendingEmail(email);
			await navigate({ to: '/login/otp' });
		} catch {
			setError('Something went wrong. Please try again.');
		}
	}

	return (
		<>
			<title>Log in · Template</title>
			<AuthShell
				eyebrow="Welcome back"
				title="Log in to your account"
				description="Enter your email address and we'll send you a secure, one-time code. No password needed."
			>
				<form className="flex flex-col gap-5" onSubmit={submit}>
					<div className="flex flex-col gap-2">
						<Label htmlFor="email">Email address</Label>
						<div className="relative">
							<Mail
								className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
								aria-hidden="true"
							/>
							<Input
								id="email"
								className="h-11 pl-9"
								type="email"
								name="email"
								autoComplete="email"
								placeholder="you@company.com"
								required
								aria-invalid={error ? true : undefined}
								aria-describedby={error ? 'login-error' : undefined}
								value={email}
								onChange={(event) => setEmail(event.target.value)}
							/>
						</div>
					</div>

					{error && (
						<p
							id="login-error"
							role="alert"
							className="rounded-lg bg-destructive/8 px-3 py-2.5 text-sm text-destructive"
						>
							{error}
						</p>
					)}

					<Button className="h-11 w-full" type="submit" disabled={mutation.isPending}>
						{mutation.isPending ? 'Sending code…' : 'Continue with email'}
						{!mutation.isPending && <ArrowRight data-icon="inline-end" />}
					</Button>
					<p className="text-center text-xs leading-5 text-muted-foreground">
						By continuing, you agree to our Terms of Service and Privacy Policy.
					</p>
				</form>
			</AuthShell>
		</>
	);
}
