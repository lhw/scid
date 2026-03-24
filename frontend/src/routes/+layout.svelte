<script lang="ts">
  import '../app.css';
  import { ModeWatcher } from 'mode-watcher';
  import { Toaster } from 'svelte-sonner';
  import { onMount } from 'svelte';
  import { logout, login } from '$lib/utils/auth';
  import { getVerifyStatus } from '$lib/utils/api';
  import { PUBLIC_POCKET_ID_URL } from '$lib/utils/public-env';

  let { children } = $props();
  let isAuthenticated = $state(false);
  let isAdmin = $state(false);

  onMount(async () => {
    try {
      const status = await getVerifyStatus(fetch);
      isAuthenticated = status.authenticated === true;
      isAdmin = status.admin === true;
    } catch {
      isAuthenticated = false;
      isAdmin = false;
    }
  });

  async function handleSignOut() {
    try {
      await logout();
    } finally {
      window.location.href = '/';
    }
  }
</script>

<svelte:head>
  <title>My SCID — Unofficial Star Citizen Identity Provider</title>
</svelte:head>

<ModeWatcher defaultMode="dark" />
<Toaster richColors position="top-right" />

<div class="flex min-h-screen flex-col bg-[#0a0e1a] text-[#e2e8f0]">
  <header
    class="sticky top-0 z-50 border-b border-[#1e3a5f] bg-[#0a0e1a]/80 backdrop-blur-md"
  >
    <div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
      <a href="/" class="group flex items-center gap-2.5">
        <img
          src="/favicon.svg"
          alt=""
          aria-hidden="true"
          class="h-8 w-8 shrink-0 transition-opacity group-hover:opacity-80"
        />
        <div class="flex flex-col leading-tight">
          <span
            class="text-2xl font-bold tracking-widest text-[#00d4ff] transition-opacity group-hover:opacity-80"
          >
            My SCID
          </span>
          <span class="text-xs text-[#e2e8f0]/50">Unofficial Star Citizen Identity Provider</span>
        </div>
      </a>
      {#if isAuthenticated}
        <div class="flex items-center gap-2">
          <a
            href="/apps"
            class="rounded-lg border border-[#1e3a5f] px-3 py-1.5 text-xs font-medium text-[#e2e8f0]/60 transition-colors hover:border-[#00d4ff]/40 hover:text-[#00d4ff]"
          >
            My Apps
          </a>
          {#if isAdmin}
            <a
              href="/admin/apps"
              class="rounded-lg border border-[#1e3a5f] px-3 py-1.5 text-xs font-medium text-[#e2e8f0]/60 transition-colors hover:border-[#ffd700]/40 hover:text-[#ffd700]"
            >
              Admin
            </a>
          {/if}
          <a
            href="{PUBLIC_POCKET_ID_URL}"
            class="rounded-lg border border-[#1e3a5f] px-3 py-1.5 text-xs font-medium text-[#e2e8f0]/60 transition-colors hover:border-[#00d4ff]/40 hover:text-[#00d4ff]"
          >
            Manage Account
          </a>
          <button
            type="button"
            onclick={handleSignOut}
            class="rounded-lg border border-[#1e3a5f] px-3 py-1.5 text-xs font-medium text-[#e2e8f0]/40 transition-colors hover:border-[#e2e8f0]/20 hover:text-[#e2e8f0]/70"
          >
            Sign Out
          </button>
        </div>
      {:else}
        <button
          type="button"
          onclick={() => login('/')}
          class="rounded-lg border border-[#00d4ff]/40 bg-[#00d4ff]/5 px-3.5 py-1.5 text-xs font-semibold text-[#00d4ff] transition-colors hover:bg-[#00d4ff]/10"
        >
          Sign In
        </button>
      {/if}
    </div>
  </header>

  <main class="flex flex-1 flex-col">
    {@render children()}
  </main>

  <footer class="border-t border-[#1e3a5f] py-6 text-center text-xs text-[#e2e8f0]/40">
    This is an unofficial fansite — The site is not affiliated with Cloud Imperium Games.
    <span class="mx-2 opacity-40">·</span>
    <a href="/impressum" class="hover:text-[#e2e8f0]/70 transition-colors">Impressum</a>
  </footer>
</div>
