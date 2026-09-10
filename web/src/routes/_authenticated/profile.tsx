import { useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { api } from '@/lib/client';
import { Button } from '@/lib/components/ui/button';
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle
} from '@/lib/components/ui/card';
import { Input } from '@/lib/components/ui/input';
import { Label } from '@/lib/components/ui/label';

export const Route = createFileRoute('/_authenticated/profile')({
	component: ProfileComponent
});

function ProfileComponent() {
	const queryClient = useQueryClient();
	const profileQuery = useQuery({ queryKey: ['profile'], queryFn: () => api.getProfile({}) });
	const mutation = useMutation({
		mutationFn: (display_name: string) => api.updateProfile({ display_name }),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: ['profile'] })
	});

	// Sync-on-first-load only; refetch wont clobber typing
	const [displayName, setDisplayName] = useState('');
	const [syncedEmail, setSyncedEmail] = useState<string | null>(null);
	const [saved, setSaved] = useState(false);
	if (profileQuery.data && syncedEmail === null) {
		setSyncedEmail(profileQuery.data.email);
		setDisplayName(profileQuery.data.display_name);
	}

	async function submit(event: React.FormEvent<HTMLFormElement>) {
		event.preventDefault();
		setSaved(false);
		try {
			await mutation.mutateAsync(displayName);
			setSaved(true);
		} catch {
			// Surfaced below via mutation.isError.
		}
	}

	return (
		<>
			<title>Your profile · Template v7</title>
			<main className="mx-auto flex min-h-screen max-w-sm flex-col justify-center p-8">
				<Card>
					<CardHeader>
						<CardTitle>Your profile</CardTitle>
						<CardDescription>
							{profileQuery.isPending
								? 'Loading…'
								: profileQuery.isError
									? 'Could not load your profile.'
									: profileQuery.data.email}
						</CardDescription>
					</CardHeader>
					<CardContent>
						<form className="flex flex-col gap-4" onSubmit={submit}>
							<div className="flex flex-col gap-1.5">
								<Label htmlFor="display_name">Display name</Label>
								<Input
									id="display_name"
									name="display_name"
									disabled={profileQuery.isPending}
									value={displayName}
									onChange={(event) => setDisplayName(event.target.value)}
								/>
							</div>

							{mutation.isError && (
								<p className="text-sm text-destructive">Something went wrong. Please try again.</p>
							)}
							{saved && !mutation.isError && (
								<p className="text-sm text-muted-foreground">Saved.</p>
							)}

							<Button type="submit" disabled={profileQuery.isPending || mutation.isPending}>
								{mutation.isPending ? 'Saving…' : 'Save'}
							</Button>
						</form>
					</CardContent>
				</Card>
			</main>
		</>
	);
}
