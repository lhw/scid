<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from 'svelte-sonner';
  import { Users, ShieldX, ShieldCheck, Trash2, AlertCircle, LoaderCircle } from '@lucide/svelte';
  import { goto } from '$app/navigation';
  import type { PageData } from './$types';
  import type { AdminUserEntry, AdminBlockedHandle } from '$lib/utils/api';
  import {
    adminListUsers, adminDeleteUser,
    adminListBlockedHandles, adminBlockHandle, adminUnblockHandle,
  } from '$lib/utils/api';

  let { data }: { data: PageData } = $props();

  let users = $state<AdminUserEntry[]>([]);
  let blockedHandles = $state<AdminBlockedHandle[]>([]);
  let loading = $state(true);
  let actionLoading = $state(false);

  // Delete user modal
  let deleteTarget = $state<AdminUserEntry | null>(null);
  let deleteBlockHandle = $state(false);
  let deleteReason = $state('');

  // Block handle modal
  let blockTarget = $state('');
  let blockReason = $state('');
  let showBlockModal = $state(false);

  onMount(async () => {
    if (!data.status?.admin) { goto('/'); return; }
    await loadAll();
  });

  async function loadAll() {
    loading = true;
    try {
      [users, blockedHandles] = await Promise.all([adminListUsers(), adminListBlockedHandles()]);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      loading = false;
    }
  }

  async function confirmDelete() {
    if (!deleteTarget) return;
    actionLoading = true;
    try {
      await adminDeleteUser(deleteTarget.user_id, deleteBlockHandle, deleteReason.trim() || undefined);
      toast.success(`User ${deleteTarget.handle} removed`);
      deleteTarget = null;
      await loadAll();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Delete failed');
    } finally {
      actionLoading = false;
    }
  }

  async function confirmBlock() {
    actionLoading = true;
    try {
      await adminBlockHandle(blockTarget.trim(), blockReason.trim() || undefined);
      toast.success(`Handle ${blockTarget.trim()} blocked`);
      showBlockModal = false;
      blockTarget = '';
      blockReason = '';
      blockedHandles = await adminListBlockedHandles();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Block failed');
    } finally {
      actionLoading = false;
    }
  }

  async function unblock(handle: string) {
    actionLoading = true;
    try {
      await adminUnblockHandle(handle);
      toast.success(`Handle ${handle} unblocked`);
      blockedHandles = blockedHandles.filter(b => b.handle !== handle);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Unblock failed');
    } finally {
      actionLoading = false;
    }
  }
</script>

<div class="mx-auto w-full max-w-4xl px-6 py-16">
  <!-- Admin sub-nav -->
  <nav class="mb-8 flex gap-2 overflow-x-auto">
    {#each [
      ['/admin/apps', 'App Registrations'],
      ['/admin/users', 'Users & Handles'],
      ['/admin/orgs', 'Org Logos'],
      ['/admin/reports', 'Reports'],
    ] as [href, label]}
      <a
        {href}
        class="shrink-0 rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors
          {label === 'Users & Handles'
            ? 'border-[#ffd700]/40 bg-[#ffd700]/5 text-[#ffd700]'
            : 'border-[#1e3a5f] text-[#e2e8f0]/50 hover:text-[#e2e8f0]'}"
      >{label}</a>
    {/each}
  </nav>

  <div class="mb-8 flex items-center gap-3">
    <Users class="h-7 w-7 text-[#ffd700]" />
    <div>
      <h1 class="text-2xl font-bold text-[#ffd700]">Users &amp; Handles</h1>
      <p class="text-xs text-[#e2e8f0]/50">Manage verified users and RSI handle blocks</p>
    </div>
  </div>

  {#if loading}
    <div class="py-16 text-center text-sm text-[#e2e8f0]/40">
      <LoaderCircle class="mx-auto mb-3 h-6 w-6 animate-spin opacity-40" />
      Loading…
    </div>
  {:else if !data.status?.admin}
    <div class="rounded-2xl border border-red-900/40 bg-red-950/20 p-8 text-center">
      <AlertCircle class="mx-auto mb-3 h-10 w-10 text-red-400/60" />
      <p class="text-[#e2e8f0]/60">You do not have admin access.</p>
    </div>
  {:else}
    <!-- Verified users -->
    <section class="mb-10">
      <h2 class="mb-3 text-sm font-semibold uppercase tracking-wider text-[#e2e8f0]/50">
        Verified users ({users.length})
      </h2>
      {#if users.length === 0}
        <div class="rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-8 text-center text-sm text-[#e2e8f0]/40">
          No verified users yet.
        </div>
      {:else}
        <div class="overflow-hidden rounded-xl border border-[#1e3a5f]">
          <table class="w-full text-sm">
            <thead class="bg-[#0d1526] text-xs text-[#e2e8f0]/50">
              <tr>
                <th class="px-4 py-2.5 text-left font-medium">RSI Handle</th>
                <th class="hidden px-4 py-2.5 text-left font-medium md:table-cell">Verified</th>
                <th class="hidden px-4 py-2.5 text-left font-medium md:table-cell">Status</th>
                <th class="px-4 py-2.5 text-right font-medium">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[#1e3a5f] bg-[#0a0e1a]">
              {#each users as user (user.user_id)}
                <tr class="transition-colors hover:bg-[#0d1526]/60">
                  <td class="px-4 py-3 font-medium text-[#e2e8f0]">{user.handle}</td>
                  <td class="hidden px-4 py-3 text-xs text-[#e2e8f0]/50 md:table-cell">
                    {new Date(user.verified_at).toLocaleDateString()}
                  </td>
                  <td class="hidden px-4 py-3 md:table-cell">
                    {#if user.handle_blocked}
                      <span class="rounded border border-red-500/20 bg-red-500/10 px-1.5 py-0.5 text-[10px] font-medium text-red-400">Handle blocked</span>
                    {:else}
                      <span class="rounded border border-green-500/20 bg-green-500/10 px-1.5 py-0.5 text-[10px] font-medium text-green-400">Active</span>
                    {/if}
                  </td>
                  <td class="px-4 py-3">
                    <div class="flex justify-end gap-1.5">
                      {#if !user.handle_blocked}
                        <button
                          onclick={() => { blockTarget = user.handle; blockReason = ''; showBlockModal = true; }}
                          class="flex items-center gap-1 rounded border border-yellow-500/20 bg-yellow-500/10 px-2 py-1 text-xs text-yellow-400 transition-colors hover:bg-yellow-500/20"
                          title="Block handle only"
                        >
                          <ShieldX class="h-3 w-3" />
                          Block
                        </button>
                      {/if}
                      <button
                        onclick={() => { deleteTarget = user; deleteBlockHandle = false; deleteReason = ''; }}
                        class="flex items-center gap-1 rounded border border-red-500/20 bg-red-500/10 px-2 py-1 text-xs text-red-400 transition-colors hover:bg-red-500/20"
                        title="Remove user data"
                      >
                        <Trash2 class="h-3 w-3" />
                        Remove
                      </button>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </section>

    <!-- Blocked handles -->
    <section>
      <div class="mb-3 flex items-center justify-between">
        <h2 class="text-sm font-semibold uppercase tracking-wider text-[#e2e8f0]/50">
          Blocked handles ({blockedHandles.length})
        </h2>
        <button
          onclick={() => { blockTarget = ''; blockReason = ''; showBlockModal = true; }}
          class="flex items-center gap-1.5 rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-1.5 text-xs font-medium text-yellow-400 transition-colors hover:bg-yellow-500/20"
        >
          <ShieldX class="h-3.5 w-3.5" />
          Block a Handle
        </button>
      </div>

      {#if blockedHandles.length === 0}
        <div class="rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-8 text-center text-sm text-[#e2e8f0]/40">
          No handles are currently blocked.
        </div>
      {:else}
        <div class="overflow-hidden rounded-xl border border-[#1e3a5f]">
          <table class="w-full text-sm">
            <thead class="bg-[#0d1526] text-xs text-[#e2e8f0]/50">
              <tr>
                <th class="px-4 py-2.5 text-left font-medium">Handle</th>
                <th class="hidden px-4 py-2.5 text-left font-medium md:table-cell">Blocked</th>
                <th class="hidden px-4 py-2.5 text-left font-medium md:table-cell">Reason</th>
                <th class="px-4 py-2.5 text-right font-medium">Action</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[#1e3a5f] bg-[#0a0e1a]">
              {#each blockedHandles as bh (bh.handle)}
                <tr class="transition-colors hover:bg-[#0d1526]/60">
                  <td class="px-4 py-3 font-medium text-[#e2e8f0]">{bh.handle}</td>
                  <td class="hidden px-4 py-3 text-xs text-[#e2e8f0]/50 md:table-cell">
                    {new Date(bh.blocked_at).toLocaleDateString()}
                    {#if bh.blocked_by}<span class="ml-1 text-[#e2e8f0]/30">by {bh.blocked_by}</span>{/if}
                  </td>
                  <td class="hidden max-w-[200px] truncate px-4 py-3 text-xs text-[#e2e8f0]/50 md:table-cell">
                    {bh.reason || '—'}
                  </td>
                  <td class="px-4 py-3 text-right">
                    <button
                      onclick={() => unblock(bh.handle)}
                      disabled={actionLoading}
                      class="flex items-center gap-1 rounded border border-green-500/20 bg-green-500/10 px-2 py-1 text-xs text-green-400 transition-colors hover:bg-green-500/20 disabled:opacity-50"
                    >
                      <ShieldCheck class="h-3 w-3" />
                      Unblock
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </section>
  {/if}
</div>

<!-- Delete user modal -->
{#if deleteTarget}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
    onclick={(e) => { if (e.target === e.currentTarget && !actionLoading) deleteTarget = null; }}
    onkeydown={(e) => { if (e.key === 'Escape' && !actionLoading) deleteTarget = null; }}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div class="w-full max-w-md rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-6 shadow-xl">
      <h2 class="mb-1 text-base font-semibold text-[#e2e8f0]">Remove user <span class="text-red-400">{deleteTarget.handle}</span>?</h2>
      <p class="mb-4 text-xs text-[#e2e8f0]/50">
        This will delete all local verification data for this user. Their Pocket ID account remains untouched.
      </p>

      <label class="mb-4 flex cursor-pointer items-start gap-2.5">
        <input type="checkbox" bind:checked={deleteBlockHandle} class="mt-0.5 accent-red-500" />
        <span class="text-sm text-[#e2e8f0]/80">Also block this RSI handle from re-verifying</span>
      </label>

      {#if deleteBlockHandle}
        <label for="del-reason" class="mb-1.5 block text-xs text-[#e2e8f0]/60">Block reason (optional)</label>
        <input
          id="del-reason"
          type="text"
          bind:value={deleteReason}
          placeholder="Reason for blocking…"
          class="mb-4 w-full rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 outline-none focus:border-red-500/50"
        />
      {/if}

      <div class="flex justify-end gap-3">
        <button
          onclick={() => deleteTarget = null}
          disabled={actionLoading}
          class="rounded-lg border border-[#1e3a5f] px-4 py-2 text-sm text-[#e2e8f0]/60 transition-colors hover:text-[#e2e8f0] disabled:opacity-50"
        >Cancel</button>
        <button
          onclick={confirmDelete}
          disabled={actionLoading}
          class="flex items-center gap-1.5 rounded-lg border border-red-500/30 bg-red-600/80 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-red-600 disabled:opacity-50"
        >
          {#if actionLoading}<LoaderCircle class="h-4 w-4 animate-spin" />{/if}
          Remove User
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Block handle modal -->
{#if showBlockModal}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
    onclick={(e) => { if (e.target === e.currentTarget && !actionLoading) showBlockModal = false; }}
    onkeydown={(e) => { if (e.key === 'Escape' && !actionLoading) showBlockModal = false; }}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div class="w-full max-w-md rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-6 shadow-xl">
      <h2 class="mb-4 text-base font-semibold text-[#e2e8f0]">Block RSI Handle</h2>
      <label for="block-handle" class="mb-1.5 block text-xs text-[#e2e8f0]/60">RSI Handle</label>
      <input
        id="block-handle"
        type="text"
        bind:value={blockTarget}
        placeholder="exact RSI handle"
        class="mb-3 w-full rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 outline-none focus:border-yellow-500/50"
      />
      <label for="block-reason" class="mb-1.5 block text-xs text-[#e2e8f0]/60">Reason (optional)</label>
      <input
        id="block-reason"
        type="text"
        bind:value={blockReason}
        placeholder="Reason…"
        class="mb-4 w-full rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 outline-none focus:border-yellow-500/50"
      />
      <div class="flex justify-end gap-3">
        <button
          onclick={() => showBlockModal = false}
          disabled={actionLoading}
          class="rounded-lg border border-[#1e3a5f] px-4 py-2 text-sm text-[#e2e8f0]/60 transition-colors hover:text-[#e2e8f0] disabled:opacity-50"
        >Cancel</button>
        <button
          onclick={confirmBlock}
          disabled={actionLoading || !blockTarget.trim()}
          class="flex items-center gap-1.5 rounded-lg border border-yellow-500/30 bg-yellow-600/80 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-yellow-600 disabled:opacity-50"
        >
          {#if actionLoading}<LoaderCircle class="h-4 w-4 animate-spin" />{/if}
          Block Handle
        </button>
      </div>
    </div>
  </div>
{/if}
