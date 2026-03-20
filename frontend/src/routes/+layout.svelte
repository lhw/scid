<script lang="ts">
  import { ModeWatcher } from 'mode-watcher';
  import { Toaster } from 'svelte-sonner';

  let { children } = $props();
</script>

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

      <nav>
        <a
          href="/verify"
          class="rounded border border-[#00d4ff]/40 px-4 py-1.5 text-sm font-medium text-[#00d4ff] transition-colors hover:bg-[#00d4ff]/10"
        >
          Verify Identity
        </a>
      </nav>
    </div>
  </header>

  <main class="flex flex-1 flex-col">
    {@render children()}
  </main>

  <footer class="border-t border-[#1e3a5f] py-6 text-center text-xs text-[#e2e8f0]/40">
    This is an unofficial fansite — not affiliated with Cloud Imperium Games.
  </footer>
</div>
