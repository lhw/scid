<script lang="ts">
  import type { Snippet } from 'svelte';
  import { twMerge } from 'tailwind-merge';

  let { class: className = '', children }: { class?: string; children?: Snippet } = $props();
</script>

<div
  class={twMerge(
    'rounded-2xl border border-[#1e3a5f] bg-[#0d1526] shadow-[0_0_40px_rgba(0,212,255,0.04)]',
    className
  )}
>
  {@render children?.()}
</div>