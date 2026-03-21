<script lang="ts">
  import '../app.css';
  import { ModeWatcher } from 'mode-watcher';
  import { Toaster } from 'svelte-sonner';
  import { onMount } from 'svelte';
  import { getAccessToken } from '$lib/utils/auth';
  import { getVerifyStatus } from '$lib/utils/api';
  import { PUBLIC_POCKET_ID_URL } from '$env/static/public';

  let { children } = $props();
  let isAuthenticated = $state(false);
  let isAdmin = $state(false);

  onMount(async () => {
    isAuthenticated = getAccessToken() !== null;
    if (isAuthenticated) {
      try {
        const status = await getVerifyStatus(fetch);
        isAdmin = status.admin === true;
      } catch {
        // Non-fatal; admin link will simply not appear.
      }
    }
  });
</script>

<svelte:head>
  <title>SCID — Star Citizen Identity Provider</title>
</svelte:head>

<ModeWatcher defaultMode="dark" />
<Toaster richColors position="top-right" />

<div class="flex min-h-screen flex-col bg-[#0a0e1a] text-[#e2e8f0]">
  <header
    class="sticky top-0 z-50 border-b border-[#1e3a5f] bg-[#0a0e1a]/80 backdrop-blur-md"
  >
    <div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
      <a href="/" class="group flex flex-col leading-tight">
        <span
          class="text-2xl font-bold tracking-widest text-[#00d4ff] transition-opacity group-hover:opacity-80"
        >
          SCID
        </span>
        <span class="text-xs text-[#e2e8f0]/50">Star Citizen Identity Provider</span>
      </a>
      {#if isAuthenticated}
        <div class="flex items-center gap-3">
          <a
            href="/apps"
            class="text-sm text-[#e2e8f0]/60 transition-colors hover:text-[#00d4ff]"
          >
            My Apps
          </a>
          {#if isAdmin}
            <a
              href="/admin/apps"
              class="text-sm text-[#e2e8f0]/60 transition-colors hover:text-[#ffd700]"
            >
              Admin
            </a>
          {/if}
          <a
            href="{PUBLIC_POCKET_ID_URL}"
            class="flex items-center gap-1.5 rounded-lg border border-[#1e3a5f] bg-[#0d1526] px-3.5 py-1.5 text-xs font-medium text-[#e2e8f0]/70 transition-colors hover:border-[#00d4ff]/40 hover:text-[#00d4ff]"
          >
            Manage Account
          </a>
        </div>
      {/if}
    </div>
  </header>

  <main class="flex flex-1 flex-col">
    {@render children()}
  </main>

  <footer class="border-t border-[#1e3a5f] py-6 text-center text-xs text-[#e2e8f0]/40">
    This is an unofficial fansite — not affiliated with Cloud Imperium Games.
  </footer>
</div>
