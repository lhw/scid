<script lang="ts">
  import { CircleCheckBig } from '@lucide/svelte';

  let { verified, handle }: { verified: boolean; handle: string } = $props();
</script>

{#if verified}
  <span
    class="inline-flex items-center gap-1.5 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-400"
  >
    <CircleCheckBig class="h-3.5 w-3.5" />
    {handle}
  </span>
{:else}
  <span
    class="inline-flex items-center rounded-full border border-[#1e3a5f] bg-[#0d1526] px-3 py-1 text-xs font-medium text-[#e2e8f0]/40"
  >
    Unverified
  </span>
{/if}
