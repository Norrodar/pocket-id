<script lang="ts">
	import SignInWrapper from '$lib/components/login-wrapper.svelte';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import QrLoginService from '$lib/services/qr-login-service';
	import WebAuthnService from '$lib/services/webauthn-service';
	import userStore from '$lib/stores/user-store';
	import { getAxiosErrorMessage } from '$lib/utils/error-util';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { onMount } from 'svelte';
	import LoginLogoErrorSuccessIndicator from '../../login/components/login-logo-error-success-indicator.svelte';

	let { data } = $props();

	const qrLoginService = new QrLoginService();
	const webauthnService = new WebAuthnService();

	let isLoading = $state(false);
	let success = $state(false);
	let error: string | undefined = $state();

	onMount(async () => {
		if ($userStore) {
			// Already logged in, show confirmation directly
			return;
		}
	});

	async function confirm() {
		isLoading = true;
		try {
			// Authenticate with passkey if not signed in
			if (!$userStore) {
				const loginOptions = await webauthnService.getLoginOptions();
				const authResponse = await startAuthentication({ optionsJSON: loginOptions });
				const user = await webauthnService.finishLogin(authResponse);
				await userStore.setUser(user);
			}

			// Confirm the QR login session
			await qrLoginService.confirmSession(data.token);
			success = true;
		} catch (e) {
			error = getAxiosErrorMessage(e);
		} finally {
			isLoading = false;
		}
	}
</script>

<svelte:head>
	<title>{m.confirm_login()}</title>
</svelte:head>

<SignInWrapper>
	<div class="flex justify-center">
		<LoginLogoErrorSuccessIndicator {success} error={!!error} />
	</div>
	<h1 class="font-playfair mt-5 text-4xl font-bold">{m.confirm_login()}</h1>
	{#if error}
		<p class="text-muted-foreground mt-2">
			{error}. {m.please_try_again()}
		</p>
	{:else if success}
		<p class="text-muted-foreground mt-2">{m.login_has_been_confirmed()}</p>
	{:else}
		<p class="text-muted-foreground mt-2">{m.do_you_want_to_authorize_this_login()}</p>
	{/if}
	{#if !success}
		<div class="mt-8 flex w-full max-w-[450px] gap-2">
			<Button variant="secondary" class="flex-1" href="/">{m.cancel()}</Button>
			{#if !error}
				<Button class="flex-1" onclick={confirm} {isLoading}>{m.confirm_login()}</Button>
			{:else}
				<Button class="flex-1" onclick={() => (error = undefined)}>{m.try_again()}</Button>
			{/if}
		</div>
	{/if}
</SignInWrapper>
