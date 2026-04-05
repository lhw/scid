<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from 'svelte-sonner';
  import { Flag, ShieldCheck, CheckCircle, LoaderCircle } from '@lucide/svelte';
  import { submitReport } from '$lib/utils/api';
  import { PUBLIC_TURNSTILE_SITE_KEY } from '$lib/utils/public-env';

  let reportType = $state<'user' | 'org'>('user');
  let target = $state('');
  let reason = $state('');
  let loading = $state(false);
  let submitted = $state(false);

  let turnstileToken = $state('');
  let turnstileWidgetId = $state<string | null>(null);
  let turnstileContainer = $state<HTMLDivElement | null>(null);
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

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (PUBLIC_TURNSTILE_SITE_KEY && !turnstileToken) {
      challengeError = 'Please complete the security check.';
      return;
    }

    loading = true;
    challengeError = '';
    try {
      await submitReport(reportType, target.trim(), reason.trim(), turnstileToken || undefined);
      submitted = true;
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not submit report. Please try again.');
      if (PUBLIC_TURNSTILE_SITE_KEY && turnstileWidgetId !== null) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (window as any).turnstile?.reset(turnstileWidgetId);
        turnstileToken = '';
      }
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>Report a Profile — My SCID</title>
</svelte:head>

<div class="flex min-h-[70vh] items-center justify-center px-4 py-16">
  <div class="w-full max-w-lg">

    <!-- Icon + heading -->
    <div class="mb-8 text-center">
      <div class="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-full border border-red-500/30 bg-red-500/10">
        <Flag class="h-8 w-8 text-red-400" />
      </div>
      <h1 class="mb-2 text-2xl font-bold text-[#e2e8f0]">Report a Profile</h1>
      <p class="text-sm text-[#e2e8f0]/60">
        Use this form to report a user profile or organisation logo that violates our
        community guidelines. Reports are reviewed by the SCID administrators.
      </p>
    </div>

    {#if submitted}
      <!-- Success state -->
      <div class="rounded-xl border border-green-500/20 bg-green-500/5 p-8 text-center">
        <CheckCircle class="mx-auto mb-3 h-10 w-10 text-green-400" />
        <h2 class="mb-2 text-lg font-semibold text-[#e2e8f0]">Report received</h2>
        <p class="text-sm text-[#e2e8f0]/60">
          Thank you — our team will review your report shortly. No further action is needed on your part.
        </p>
        <a href="/" class="mt-5 inline-block rounded-lg border border-[#1e3a5f] px-4 py-2 text-xs font-medium text-[#e2e8f0]/60 transition-colors hover:border-[#00d4ff]/40 hover:text-[#00d4ff]">
          Back to home
        </a>
      </div>
    {:else}
      <form onsubmit={handleSubmit} class="rounded-xl border border-[#1e3a5f] bg-[#111827]/80 p-8 backdrop-blur-sm space-y-5">

        <!-- Report type -->
        <fieldset>
          <legend class="mb-2 block text-xs font-medium text-[#e2e8f0]/60">What are you reporting?</legend>
          <div class="grid grid-cols-2 gap-2">
            {#each [['user', 'User profile', 'An RSI citizen handle'], ['org', 'Organisation', 'An org logo or listing']] as [val, label, desc]}
              <label class="flex cursor-pointer flex-col rounded-lg border p-3 transition-colors
                {reportType === val
                  ? 'border-red-500/40 bg-red-500/5 text-red-400'
                  : 'border-[#1e3a5f] text-[#e2e8f0]/60 hover:border-[#1e3a5f] hover:text-[#e2e8f0]/80'}">
                <input type="radio" name="type" value={val} bind:group={reportType} class="sr-only" />
                <span class="text-sm font-medium">{label}</span>
                <span class="mt-0.5 text-xs opacity-70">{desc}</span>
              </label>
            {/each}
          </div>
        </fieldset>

        <!-- Target -->
        <div>
          <label for="target" class="mb-1.5 block text-xs font-medium text-[#e2e8f0]/60">
            {reportType === 'user' ? 'RSI Handle' : 'Organisation SID'}
          </label>
          <input
            id="target"
            type="text"
            bind:value={target}
            required
            placeholder={reportType === 'user' ? 'e.g. CitizenName' : 'e.g. SPAWO'}
            class="w-full rounded-lg border border-[#1e3a5f] bg-[#0d1526] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30
                   outline-none transition-colors focus:border-red-500/50 focus:ring-1 focus:ring-red-500/20"
          />
        </div>

        <!-- Reason -->
        <div>
          <label for="reason" class="mb-1.5 block text-xs font-medium text-[#e2e8f0]/60">
            Reason <span class="text-[#e2e8f0]/30">(min 10 chars)</span>
          </label>
          <textarea
            id="reason"
            bind:value={reason}
            required
            minlength="10"
            maxlength="2000"
            rows="4"
            placeholder="Describe the issue..."
            class="w-full resize-none rounded-lg border border-[#1e3a5f] bg-[#0d1526] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30
                   outline-none transition-colors focus:border-red-500/50 focus:ring-1 focus:ring-red-500/20"
          ></textarea>
          <div class="mt-1 text-right text-xs text-[#e2e8f0]/30">{reason.length}/2000</div>
        </div>

        <!-- Turnstile -->
        {#if PUBLIC_TURNSTILE_SITE_KEY}
          <div>
            <div class="mb-2 flex items-center gap-1.5 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">
              <ShieldCheck class="h-3.5 w-3.5" />
              <span>Security check</span>
            </div>
            <div bind:this={turnstileContainer} class="overflow-hidden rounded-lg"></div>
            {#if challengeError}
              <p class="mt-1.5 text-xs text-red-400">{challengeError}</p>
            {/if}
          </div>
        {/if}

        <!-- Notice -->
        <p class="text-xs text-[#e2e8f0]/40">
          False reports may result in your IP being blocked. This form does not require login;
          your IP address is logged for abuse prevention.
        </p>

        <!-- Submit -->
        <button
          type="submit"
          disabled={loading || (PUBLIC_TURNSTILE_SITE_KEY ? !turnstileToken : false)}
          class="w-full rounded-lg bg-red-600/80 px-4 py-2.5 text-sm font-semibold text-white
                 transition-all hover:bg-red-600 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {#if loading}
            <span class="inline-flex items-center gap-2">
              <LoaderCircle class="h-4 w-4 animate-spin" />
              Submitting…
            </span>
          {:else}
            Submit Report
          {/if}
        </button>
      </form>
    {/if}

  </div>
</div>
