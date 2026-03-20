<script lang="ts">
  import { Copy, Check } from '@lucide/svelte';

  let { text }: { text: string } = $props();

  let copied = $state(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      copied = true;
      setTimeout(() => {
        copied = false;
      }, 2000);
    } catch {
      // Clipboard not available in this context
    }
  }
</script>

<button
  type="button"
  onclick={copy}
  class="flex items-center gap-1.5 rounded-md border border-[#1e3a5f] px-2.5 py-1 text-xs font-medium text-[#e2e8f0]/60 transition-colors hover:border-[#00d4ff]/40 hover:text-[#00d4ff]"
  aria-label="Copy to clipboard"
>
  {#if copied}
    <Check class="h-3.5 w-3.5 text-emerald-400" />
    <span class="text-emerald-400">Copied!</span>
  {:else}
    <Copy class="h-3.5 w-3.5" />
    Copy
  {/if}
</button>
