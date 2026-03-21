<script lang="ts">
  import { toast } from 'svelte-sonner';
  import { untrack } from 'svelte';
  import { startVerify, confirmVerify } from '$lib/utils/api.js';
  import { login } from '$lib/utils/auth.js';
  import { CircleCheckBig, CircleX, Copy, Check, LoaderCircle, LogIn, UserPlus } from '@lucide/svelte';
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  type Step = 'handle' | 'token' | 'confirming' | 'success' | 'failed';

  // Allow re-verification even when already verified (e.g. handle change).
  let allowReVerify = $state(false);

  // Snapshot page data once at init time (data prop may be a proxy — read eagerly)
  const initStatus = untrack(() => data.status);
  let authenticated = $state(initStatus?.authenticated === true);
  const pendingStatus = initStatus && !initStatus.verified && initStatus.pending_handle
    ? initStatus
    : null;

  let step = $state<Step>(pendingStatus ? 'token' : 'handle');
  let handle = $state(pendingStatus?.pending_handle ?? '');
  let token = $state('');
  let expiresAt = $state<Date | null>(
    pendingStatus?.pending_expires_at ? new Date(pendingStatus.pending_expires_at) : null
  );
  let errorMessage = $state('');
  let loading = $state(false);
  let handleError = $state('');
  let timeLeft = $state('');
  let copied = $state(false);

  // Countdown timer — updates every minute while expiresAt is set
  $effect(() => {
    if (!expiresAt) return;
    const target = expiresAt;

    function update() {
      const ms = target.getTime() - Date.now();
      if (ms <= 0) {
        timeLeft = 'Expired';
        return;
      }
      const hours = Math.floor(ms / 3_600_000);
      const mins = Math.floor((ms % 3_600_000) / 60_000);
      timeLeft = hours > 0 ? `Expires in ${hours}h ${mins}m` : `Expires in ${mins}m`;
    }

    update();
    const id = setInterval(update, 60_000);
    return () => clearInterval(id);
  });

  // Auto-confirm when entering the confirming step
  $effect(() => {
    if (step !== 'confirming') return;
    confirmVerify()
      .then((res) => {
        if (res.verified) {
          step = 'success';
        } else {
          errorMessage =
            res.message ?? 'Token not found in your RSI profile bio. Make sure it is saved and try again.';
          step = 'failed';
        }
      })
      .catch((err: unknown) => {
        errorMessage = err instanceof Error ? err.message : 'Verification failed. Please try again.';
        step = 'failed';
      });
  });

  function validateHandle(v: string): string {
    if (v.length < 3) return 'Handle must be at least 3 characters.';
    if (v.length > 60) return 'Handle must be at most 60 characters.';
    if (!/^[a-zA-Z0-9_-]+$/.test(v)) return 'Only letters, numbers, hyphens, and underscores allowed.';
    return '';
  }

  async function submitHandle(e: SubmitEvent) {
    e.preventDefault();
    handleError = validateHandle(handle);
    if (handleError) return;
    loading = true;
    try {
      const res = await startVerify(handle);
      token = res.token;
      expiresAt = new Date(res.expires_at);
      step = 'token';
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to start verification.');
    } finally {
      loading = false;
    }
  }

  async function copyToken() {
    try {
      await navigator.clipboard.writeText(token);
      copied = true;
      setTimeout(() => {
        copied = false;
      }, 2000);
    } catch {
      toast.error('Failed to copy — please select and copy the token manually.');
    }
  }

  function startOver() {
    step = 'handle';
    handle = '';
    token = '';
    expiresAt = null;
    errorMessage = '';
    handleError = '';
  }

  const stepNum = $derived(
    step === 'handle' ? 1 :
    step === 'token' || step === 'failed' ? 2 :
    step === 'confirming' ? 3 : 4
  );

  const wizardSteps = [
    { n: 1, label: 'Handle' },
    { n: 2, label: 'Token' },
    { n: 3, label: 'Verify' },
    { n: 4, label: 'Done' },
  ] as const;
</script>

<div class="flex flex-1 items-center justify-center px-4 py-16">
  {#if authenticated === null}
    <!-- Resolving auth state — brief flicker guard -->
    <div></div>
  {:else if !authenticated}
    <!-- Login gate — new users create an account; returning users sign in -->
    <div class="w-full max-w-lg space-y-4">
      <!-- Create account card -->
      <div class="rounded-xl border border-[#1e3a5f] bg-[#111827]/80 p-8 backdrop-blur-sm">
        <div class="mb-4 flex items-center gap-3">
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-[#00d4ff]/30 bg-[#00d4ff]/10">
            <UserPlus class="h-5 w-5 text-[#00d4ff]" />
          </div>
          <div>
            <h2 class="font-bold text-[#e2e8f0]">New here?</h2>
            <p class="text-xs text-[#e2e8f0]/50">Create a SCID account, then link your RSI identity.</p>
          </div>
        </div>
        <a
          href="/register"
          class="flex w-full items-center justify-center gap-2 rounded-lg border border-[#00d4ff]/40 px-5 py-2.5 text-sm font-semibold text-[#00d4ff] transition-colors hover:bg-[#00d4ff]/10"
        >
          <UserPlus class="h-4 w-4" />
          Create a SCID account →
        </a>
        <p class="mt-3 text-center text-xs text-[#e2e8f0]/40">
          You'll set up a passkey on the next page.<br />After that, come back here to link your RSI handle.
        </p>
      </div>

      <!-- Sign in card -->
      <div class="rounded-xl border border-[#1e3a5f] bg-[#111827]/80 p-8 backdrop-blur-sm">
        <div class="mb-4 flex items-center gap-3">
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-[#1e3a5f] bg-[#0d1526]">
            <LogIn class="h-5 w-5 text-[#94a3b8]" />
          </div>
          <div>
            <h2 class="font-bold text-[#e2e8f0]">Already have an account?</h2>
            <p class="text-xs text-[#e2e8f0]/50">Sign in to start or continue RSI identity verification.</p>
          </div>
        </div>
        <button
          onclick={() => login('/verify')}
          class="flex w-full items-center justify-center gap-2 rounded-lg bg-[#00d4ff] px-5 py-2.5 text-sm font-semibold text-[#0a0e1a] transition-colors hover:bg-[#00b8dc]"
        >
          <LogIn class="h-4 w-4" />
          Sign in with SCID
        </button>
      </div>
    </div>
  {:else if data.status?.verified && !allowReVerify}
    <!-- Already-verified card -->
    <div
      class="w-full max-w-lg rounded-xl border border-emerald-500/30 bg-[#111827]/80 p-8 text-center backdrop-blur-sm"
    >
      <div class="mb-4 flex justify-center">
        <CircleCheckBig class="h-16 w-16 text-emerald-400" />
      </div>
      <h1 class="mb-2 text-2xl font-bold text-[#e2e8f0]">Already Verified</h1>
      <p class="mb-1 text-[#e2e8f0]/70">
        You are verified as
        <span class="font-semibold text-[#00d4ff]">{data.status.handle}</span>
      </p>
      {#if data.status.verified_at}
        <p class="mb-4 text-sm text-[#e2e8f0]/40">
          Verified on {new Date(data.status.verified_at).toLocaleDateString(undefined, {
            year: 'numeric',
            month: 'long',
            day: 'numeric'
          })}
        </p>
      {/if}
      <p class="mb-8 text-sm text-[#e2e8f0]/60">
        You can safely remove your verification token from your RSI bio.
      </p>
      <div class="flex flex-col gap-3 sm:flex-row sm:justify-center">
        <a
          href="https://robertsspaceindustries.com/en/account/profile"
          target="_blank"
          rel="noopener noreferrer"
          class="rounded-lg border border-[#1e3a5f] px-5 py-2.5 text-sm font-medium text-[#e2e8f0]/70 transition-colors hover:border-[#00d4ff]/40 hover:text-[#00d4ff]"
        >
          Edit RSI Profile →
        </a>
        <a
          href="/"
          class="rounded-lg bg-[#00d4ff] px-5 py-2.5 text-sm font-semibold text-[#0a0e1a] transition-colors hover:bg-[#00b8dc]"
        >
          Go to Homepage →
        </a>
      </div>
      <div class="mt-6 border-t border-[#1e3a5f] pt-6">
        <p class="mb-3 text-xs text-[#e2e8f0]/40">Changed your RSI handle?</p>
        <button
          onclick={() => { allowReVerify = true; }}
          class="text-sm text-[#e2e8f0]/50 underline underline-offset-2 transition-colors hover:text-[#00d4ff]"
        >
          Re-verify with a different RSI handle
        </button>
      </div>
    </div>
  {:else}
    <!-- Wizard card -->
    <div
      class="w-full max-w-lg rounded-xl border border-[#1e3a5f] bg-[#111827]/80 p-8 backdrop-blur-sm"
    >
      {#if allowReVerify}
        <div class="mb-6 rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-xs text-amber-400">
          <strong>Changing your RSI handle</strong> — completing this flow will replace your existing verification.
          Your account stays; only the linked RSI handle changes.
        </div>
      {/if}
      <!-- Step indicator -->
      <div class="mb-8 flex items-center">
        {#each wizardSteps as { n, label } (n)}
          <div class="flex flex-col items-center gap-1.5">
            <div
              class={[
                'flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold transition-colors',
                stepNum > n
                  ? 'bg-[#00d4ff] text-[#0a0e1a]'
                  : stepNum === n
                    ? 'border-2 border-[#00d4ff] text-[#00d4ff]'
                    : 'border-2 border-[#1e3a5f] text-[#e2e8f0]/30'
              ].join(' ')}
            >
              {#if stepNum > n}
                <Check class="h-4 w-4" />
              {:else}
                {n}
              {/if}
            </div>
            <span
              class={[
                'text-xs font-medium',
                stepNum >= n ? 'text-[#00d4ff]' : 'text-[#e2e8f0]/30'
              ].join(' ')}>{label}</span
            >
          </div>
          {#if n < 4}
            <div
              class={[
                'mx-2 mb-5 h-px flex-1 transition-colors',
                stepNum > n ? 'bg-[#00d4ff]/40' : 'bg-[#1e3a5f]'
              ].join(' ')}
            ></div>
          {/if}
        {/each}
      </div>

      <!-- ── Step 1: Enter handle ── -->
      {#if step === 'handle'}
        <form onsubmit={submitHandle} class="space-y-6">
          <div>
            <h2 class="mb-1 text-xl font-bold text-[#e2e8f0]">Enter Your RSI Handle</h2>
            <p class="text-sm text-[#e2e8f0]/60">
              We'll generate a unique token for you to paste into your RSI profile bio.
            </p>
          </div>

          <div class="space-y-1.5">
            <label for="rsi-handle" class="block text-sm font-medium text-[#e2e8f0]/80">
              Your RSI Handle
            </label>
            <input
              id="rsi-handle"
              type="text"
              bind:value={handle}
              placeholder="Example"
              autocomplete="off"
              spellcheck="false"
              oninput={() => {
                if (handleError) handleError = validateHandle(handle);
              }}
              class={[
                'w-full rounded-lg border bg-[#0d1526] px-4 py-2.5 font-mono text-[#e2e8f0] placeholder-[#e2e8f0]/30 outline-none transition-colors focus:ring-1',
                handleError
                  ? 'border-red-500/60 focus:border-red-500 focus:ring-red-500/30'
                  : 'border-[#1e3a5f] focus:border-[#00d4ff] focus:ring-[#00d4ff]/20'
              ].join(' ')}
            />
            {#if handleError}
              <p class="text-xs text-red-400">{handleError}</p>
            {/if}
          </div>

          <button
            type="submit"
            disabled={loading}
            class="flex w-full items-center justify-center gap-2 rounded-lg bg-[#00d4ff] px-5 py-2.5 font-semibold text-[#0a0e1a] transition-colors hover:bg-[#00b8dc] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {#if loading}
              <LoaderCircle class="h-4 w-4 animate-spin" />
              Starting…
            {:else}
              Begin Verification →
            {/if}
          </button>
        </form>

      <!-- ── Step 2: Place token ── -->
      {:else if step === 'token'}
        <div class="space-y-6">
          <div>
            <h2 class="mb-1 text-xl font-bold text-[#e2e8f0]">Place Token in Your Bio</h2>
            <p class="text-sm text-[#e2e8f0]/60">
              Add the following token anywhere in your RSI profile bio, then click Verify.
            </p>
          </div>

          {#if token}
            <!-- Token display block -->
            <div class="rounded-lg border border-[#1e3a5f] bg-[#0d1526] p-4">
              <div class="mb-3 flex items-center justify-between">
                <span class="text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">
                  Verification Token
                </span>
                <button
                  type="button"
                  onclick={copyToken}
                  class="flex items-center gap-1.5 rounded-md border border-[#1e3a5f] px-2.5 py-1 text-xs font-medium text-[#e2e8f0]/60 transition-colors hover:border-[#00d4ff]/40 hover:text-[#00d4ff]"
                  aria-label="Copy token"
                >
                  {#if copied}
                    <Check class="h-3.5 w-3.5 text-emerald-400" />
                    <span class="text-emerald-400">Copied!</span>
                  {:else}
                    <Copy class="h-3.5 w-3.5" />
                    Copy
                  {/if}
                </button>
              </div>
              <code class="block break-all font-mono text-lg text-[#00d4ff]">{token}</code>
              {#if timeLeft}
                <p class="mt-2 text-xs text-[#e2e8f0]/40">{timeLeft}</p>
              {/if}
            </div>
          {:else}
            <!-- Resumed pending state — no token available client-side -->
            <div
              class="rounded-lg border border-yellow-500/20 bg-yellow-500/5 p-4 text-sm text-yellow-400/80"
            >
              You have a pending verification in progress. If you already placed your token in your
              RSI bio, click <strong class="text-yellow-400">Verify Now</strong>. Otherwise, start
              over to receive a fresh token.
            </div>
          {/if}

          <p class="text-sm text-[#e2e8f0]/70">
            Open your RSI profile to add the token:
            <a
              href="https://robertsspaceindustries.com/en/account/profile"
              target="_blank"
              rel="noopener noreferrer"
              class="text-[#00d4ff] hover:underline"
            >
              Edit your RSI profile →
            </a>
          </p>

          <div class="flex flex-col gap-3">
            <button
              type="button"
              onclick={() => {
                step = 'confirming';
              }}
              class="w-full rounded-lg bg-[#00d4ff] px-5 py-2.5 font-semibold text-[#0a0e1a] transition-colors hover:bg-[#00b8dc]"
            >
              I've added it — Verify Now
            </button>
            <button
              type="button"
              onclick={startOver}
              class="text-sm text-[#e2e8f0]/50 transition-colors hover:text-[#e2e8f0]"
            >
              ← Start over
            </button>
          </div>
        </div>

      <!-- ── Step 3: Confirming ── -->
      {:else if step === 'confirming'}
        <div class="flex flex-col items-center justify-center space-y-4 py-10 text-center">
          <LoaderCircle class="h-12 w-12 animate-spin text-[#00d4ff]" />
          <h2 class="text-xl font-bold text-[#e2e8f0]">Checking Your RSI Profile…</h2>
          <p class="text-sm text-[#e2e8f0]/60">This usually takes just a moment.</p>
        </div>

      <!-- ── Step 4: Success ── -->
      {:else if step === 'success'}
        <div class="flex flex-col items-center justify-center space-y-4 py-6 text-center">
          <CircleCheckBig class="h-16 w-16 text-emerald-400" />
          <h2 class="text-2xl font-bold text-[#e2e8f0]">Verified!</h2>
          <p class="text-[#e2e8f0]/70">
            You are now verified as
            <span class="font-semibold text-[#00d4ff]">{handle}</span>.
          </p>
          <p class="text-sm text-[#e2e8f0]/50">
            You can safely remove the token from your RSI bio now.
          </p>
          <div class="flex flex-col gap-3 pt-2 sm:flex-row sm:justify-center">
            <a
              href="https://robertsspaceindustries.com/en/account/profile"
              target="_blank"
              rel="noopener noreferrer"
              class="rounded-lg border border-[#1e3a5f] px-5 py-2.5 text-sm font-medium text-[#e2e8f0]/70 transition-colors hover:border-[#00d4ff]/40 hover:text-[#00d4ff]"
            >
              Edit RSI Profile →
            </a>
            <a
              href="/"
              class="rounded-lg bg-[#00d4ff] px-5 py-2.5 text-sm font-semibold text-[#0a0e1a] transition-colors hover:bg-[#00b8dc]"
            >
              Go to Homepage →
            </a>
          </div>
        </div>

      <!-- ── Step 5: Failed ── -->
      {:else if step === 'failed'}
        <div class="flex flex-col items-center justify-center space-y-4 py-6 text-center">
          <CircleX class="h-16 w-16 text-red-400" />
          <h2 class="text-2xl font-bold text-[#e2e8f0]">Verification Failed</h2>
          <p class="max-w-sm text-sm text-[#e2e8f0]/60">{errorMessage}</p>
          <div class="flex flex-col gap-3 pt-2 sm:flex-row sm:justify-center">
            <button
              type="button"
              onclick={() => {
                step = 'token';
              }}
              class="rounded-lg bg-[#00d4ff] px-5 py-2.5 text-sm font-semibold text-[#0a0e1a] transition-colors hover:bg-[#00b8dc]"
            >
              Try Again
            </button>
            <button
              type="button"
              onclick={startOver}
              class="rounded-lg border border-[#1e3a5f] px-5 py-2.5 text-sm font-medium text-[#e2e8f0]/70 transition-colors hover:border-[#00d4ff]/40 hover:text-[#00d4ff]"
            >
              Start Over
            </button>
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>
