<script lang="ts">
  import type { Snippet } from 'svelte';
  import { twMerge } from 'tailwind-merge';

  let {
    forId,
    label,
    required = false,
    error = '',
    hint = '',
    class: className = '',
    children,
  }: {
    forId?: string;
    label: string;
    required?: boolean;
    error?: string;
    hint?: string;
    class?: string;
    children?: Snippet;
  } = $props();
</script>

<div class={twMerge('space-y-1.5', className)}>
  <svelte:element
    this={forId ? 'label' : 'div'}
    {...(forId ? { for: forId } : {})}
    class="block text-sm font-medium text-[#e2e8f0]/70"
  >
    {label}{#if required} <span class="text-red-400">*</span>{/if}
  </svelte:element>
  {@render children?.()}
  {#if hint}
    <p class="text-xs text-[#e2e8f0]/30">{hint}</p>
  {/if}
  {#if error}
    <p class="text-xs text-red-400">{error}</p>
  {/if}
</div>