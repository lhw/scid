<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from 'svelte-sonner';
  import { getSignupToken } from '$lib/utils/api.js';
  import Panel from '$lib/components/Panel.svelte';
  import { PUBLIC_POCKET_ID_URL, PUBLIC_TURNSTILE_SITE_KEY } from '$lib/utils/public-env';
  import { UserPlus, LoaderCircle, ShieldCheck, LogIn } from '@lucide/svelte';

  let turnstileToken = $state('');
  let turnstileWidgetId = $state<string | null>(null);
  let turnstileContainer = $state<HTMLDivElement | null>(null);
  let loading = $state(false);
  let challengeError = $state('');

  onMount(() => {
    if (!PUBLIC_TURNSTILE_SITE_KEY) return;

    const script = document.createElement('script');
    script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';
    script.async = true;
    script.onload = () => {
      if (!turnstileContainer) return;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const id = (window as any).turnstile.render(turnstileContainer, {
        sitekey: PUBLIC_TURNSTILE_SITE_KEY,
        theme: 'dark',
        size: 'flexible',
        callback: (t: string) => { turnstileToken = t; challengeError = ''; },
        'expired-callback': () => { turnstileToken = ''; },
        'error-callback': () => {
          turnstileToken = '';
          challengeError = 'Challenge failed — please try again.';
        },
      });
      turnstileWidgetId = id;
    };
    document.head.appendChild(script);
  });

  async function proceed() {
    loading = true;
    challengeError = '';
    try {
      const signupToken = await getSignupToken(turnstileToken || undefined);
      window.location.href = `${PUBLIC_POCKET_ID_URL}/signup?token=${encodeURIComponent(signupToken)}`;
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Could not start registration. Try again.';
      toast.error(msg);
      if (PUBLIC_TURNSTILE_SITE_KEY && turnstileWidgetId !== null) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (window as any).turnstile?.reset(turnstileWidgetId);
      }
      turnstileToken = '';
      loading = false;
    }
  }
</script>

<div class="flex min-h-[70vh] items-center justify-center px-4 py-16">
  <div class="w-full max-w-md">
    <!-- Icon + heading -->
    <div class="mb-8 text-center">
      <div
        class="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-full border border-[#00d4ff]/30 bg-[#00d4ff]/10"
      >
        <UserPlus class="h-8 w-8 text-[#00d4ff]" />
      </div>
      <h1 class="mb-2 text-2xl font-bold text-[#e2e8f0]">Create a SCID Account</h1>
      <p class="text-sm text-[#e2e8f0]/60">
        SCID uses <strong class="text-[#e2e8f0]/80">Pocket ID</strong> for secure passkey authentication.
        You'll be taken there to create your account, then returned here automatically to link your RSI handle.
      </p>
    </div>

    <!-- Step flow -->
    <ol class="mb-7 flex items-center gap-0">
      {#each [
        { label: 'Start here', active: true },
        { label: 'Create account in Pocket ID', active: false },
        { label: 'Return & verify RSI', active: false },
      ] as step, i}
        <li class="flex flex-1 flex-col items-center gap-1.5 text-center">
          <span class="flex h-7 w-7 items-center justify-center rounded-full border text-xs font-bold
            {step.active ? 'border-[#00d4ff] bg-[#00d4ff]/10 text-[#00d4ff]' : 'border-[#1e3a5f] bg-[#0d1526] text-[#e2e8f0]/30'}">
            {i + 1}
          </span>
          <span class="text-[10px] leading-tight {step.active ? 'text-[#e2e8f0]/70' : 'text-[#e2e8f0]/30'}">{step.label}</span>
        </li>
        {#if i < 2}
          <div class="mb-4 h-px w-6 flex-shrink-0 bg-[#1e3a5f]"></div>
        {/if}
      {/each}
    </ol>

    <Panel class="rounded-xl bg-[#111827]/80 p-8 backdrop-blur-sm">
      {#if PUBLIC_TURNSTILE_SITE_KEY}
        <!-- Security check -->
        <div class="mb-5">
          <div class="mb-2 flex items-center gap-1.5 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">
            <ShieldCheck class="h-3.5 w-3.5" />
            <span>Security check</span>
          </div>
          <!--
            Wrap the Turnstile widget in a styled container so the dark widget
            background blends with the SCID dark palette.
            overflow-hidden ensures the widget's rounded corners are clipped.
          -->
          <div class="overflow-hidden rounded-lg border border-[#1e3a5f] bg-[#0d1526] shadow-inner">
            <div bind:this={turnstileContainer}></div>
          </div>
          {#if challengeError}
            <p class="mt-2 text-xs text-red-400">{challengeError}</p>
          {/if}
        </div>
      {/if}

      <button
        onclick={proceed}
        disabled={loading || (!!PUBLIC_TURNSTILE_SITE_KEY && !turnstileToken)}
        class="flex w-full items-center justify-center gap-2 rounded-lg bg-[#00d4ff] px-5 py-3 text-sm font-semibold text-[#0a0e1a] transition-colors hover:bg-[#00b8dc] disabled:cursor-not-allowed disabled:opacity-50"
      >
        {#if loading}
          <LoaderCircle class="h-4 w-4 animate-spin" />
          Opening Pocket ID…
        {:else}
          <UserPlus class="h-4 w-4" />
          Continue to Pocket ID →
        {/if}
      </button>

      <p class="mt-5 text-center text-xs text-[#e2e8f0]/40">
        After completing your account in Pocket ID you'll be redirected back here automatically
        to complete the RSI verification step.
      </p>

      <p class="mt-4 text-center text-sm text-[#e2e8f0]/50">
        Already have an account?
        <a href="/verify" class="text-[#00d4ff] hover:underline">
          <LogIn class="inline h-3.5 w-3.5 align-[-1px]" />
          Sign in
        </a>
      </p>
    </Panel>
  </div>
</div>
