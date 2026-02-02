<script lang="ts">
	import { afterNavigate, goto } from '$app/navigation';
	import Qrcode from '$lib/components/qrcode/qrcode.svelte';
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import QrLoginService from '$lib/services/qr-login-service';
	import userStore from '$lib/stores/user-store';
	import { getAxiosErrorMessage } from '$lib/utils/error-util';
	import { onDestroy, onMount } from 'svelte';
	import LoginLogoErrorSuccessIndicator from '../../components/login-logo-error-success-indicator.svelte';

	let { data } = $props();

	const qrLoginService = new QrLoginService();

	let token: string | null = $state(null);
	let qrUrl: string | null = $state(null);
	let isLoading = $state(true);
	let success = $state(false);
	let expired = $state(false);
	let error: string | undefined = $state();
	let pollInterval: ReturnType<typeof setInterval> | null = $state(null);
	let backHref = $state('/login/alternative');

	afterNavigate((e) => {
		if (e.from?.url.pathname) {
			backHref = e.from.url.pathname + e.from.url.search;
		}
	});

	onMount(async () => {
		try {
			const session = await qrLoginService.initSession();
			token = session.token;
			qrUrl = `${window.location.origin}/qr/${session.token}`;
			isLoading = false;

			// Start polling for authorization
			pollInterval = setInterval(pollStatus, 3000);

			// Auto-expire after session duration
			setTimeout(() => {
				if (!success) {
					expired = true;
					stopPolling();
				}
			}, session.expiresIn * 1000);
		} catch (e) {
			error = getAxiosErrorMessage(e);
			isLoading = false;
		}
	});

	onDestroy(() => {
		stopPolling();
	});

	function stopPolling() {
		if (pollInterval) {
			clearInterval(pollInterval);
			pollInterval = null;
		}
	}

	async function pollStatus() {
		if (!token) return;

		try {
			const status = await qrLoginService.getStatus(token);
			if (status.authorized) {
				stopPolling();
				await exchangeSession();
			}
		} catch {
			// Token expired or invalid, stop polling
			expired = true;
			stopPolling();
		}
	}

	async function exchangeSession() {
		if (!token) return;

		try {
			const user = await qrLoginService.exchangeSession(token);
			await userStore.setUser(user);
			success = true;

			setTimeout(() => {
				try {
					goto(data.redirect);
				} catch {
					goto('/settings');
				}
			}, 1000);
		} catch (e) {
			error = getAxiosErrorMessage(e);
		}
	}

	async function retry() {
		error = undefined;
		expired = false;
		isLoading = true;
		token = null;
		qrUrl = null;

		try {
			const session = await qrLoginService.initSession();
			token = session.token;
			qrUrl = `${window.location.origin}/qr/${session.token}`;
			isLoading = false;

			pollInterval = setInterval(pollStatus, 3000);

			setTimeout(() => {
				if (!success) {
					expired = true;
					stopPolling();
				}
			}, session.expiresIn * 1000);
		} catch (e) {
			error = getAxiosErrorMessage(e);
			isLoading = false;
		}
	}
</script>

<svelte:head>
	<title>{m.qr_code()}</title>
</svelte:head>

<SignInWrapper>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator {success} error={!!error || expired} />
	</div>
	<h1 class="font-playfair mt-5 text-4xl font-bold">{m.qr_code()}</h1>
	{#if error}
		<p class="text-muted-foreground mt-2">
			{error}. {m.please_try_again()}
		</p>
	{:else if expired}
		<p class="text-muted-foreground mt-2">{m.qr_code_expired()}</p>
	{:else if success}
		<p class="text-muted-foreground mt-2">{m.sign_in_successful()}</p>
	{:else if isLoading}
		<p class="text-muted-foreground mt-2">{m.loading()}</p>
	{:else}
		<p class="text-muted-foreground mt-2">{m.scan_qr_code_with_phone_to_sign_in()}</p>
		<div class="mt-6">
			<Qrcode value={qrUrl} size={200} />
		</div>
		<p class="text-muted-foreground mt-4 text-sm">{m.waiting_for_confirmation()}</p>
	{/if}
	<div class="mt-8 flex w-full max-w-[450px] gap-2">
		<Button variant="secondary" class="flex-1" href={backHref}>{m.go_back()}</Button>
		{#if error || expired}
			<Button class="flex-1" onclick={retry}>{m.try_again()}</Button>
		{/if}
	</div>
</SignInWrapper>
