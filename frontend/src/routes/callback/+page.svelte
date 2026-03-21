<script lang="ts">
  import { onMount } from 'svelte';
  import { handleCallback } from '$lib/utils/auth.js';

  let error = $state('');
  let working = $state(true);

  onMount(async () => {
    const params = new URLSearchParams(window.location.search);
    const code = params.get('code');
    const state = params.get('state');
    const errParam = params.get('error');

    if (errParam) {
      error = params.get('error_description') ?? errParam;
      working = false;
      return;
    }

    if (!code || !state) {
      error = 'Missing authorization parameters.';
      working = false;
      return;
    }

    try {
      const returnPath = await handleCallback(code, state);
      window.location.replace(returnPath);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Authentication failed.';
      working = false;
    }
  });
</script>

<div class="flex min-h-[60vh] items-center justify-center px-4">
  {#if working}
    <p class="animate-pulse text-[#94a3b8]">Completing login…</p>
  {:else}
    <div class="w-full max-w-sm rounded-xl border border-red-500/40 bg-red-500/10 p-8 text-center">
      <p class="mb-1 font-semibold text-red-400">Authentication failed</p>
      <p class="mb-6 text-sm text-[#94a3b8]">{error}</p>
      <a
        href="/verify"
        class="text-sm text-[#00d4ff] hover:underline"
      >← Back to verification</a>
    </div>
  {/if}
</div>
