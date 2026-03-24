<script lang="ts">
  import { onMount } from 'svelte';

  function decodeFrag(arr: number[], key: number): string {
    return (arr || []).slice().reverse().map((c) => String.fromCharCode(c ^ key)).join('');
  }

  const nameFrag = [88, 79, 70, 70, 79, 125, 10, 4, 102];
  const emailFrag = [64, 84, 23, 93, 80, 90, 74, 121, 87, 80, 84, 93, 88];

  let contactMount: HTMLDivElement | null = null;

  function renderContact(el: HTMLDivElement) {
    const lines = [decodeFrag(nameFrag, 42), decodeFrag(emailFrag, 57)];

    while (el.firstChild) el.removeChild(el.firstChild);

    const canvas = document.createElement('canvas');
    canvas.setAttribute('role', 'img');
    canvas.setAttribute('aria-label', 'Impressum name and email');
    canvas.style.width = '100%';
    canvas.style.height = 'auto';
    el.appendChild(canvas);

    const rect = el.getBoundingClientRect();
    const cssWidth = rect.width || parseFloat(getComputedStyle(el).width) || 700;
    const padding = 16;
    const baseFontSize = Math.max(13, parseFloat(getComputedStyle(document.documentElement).fontSize) || 16);
    const fontSize = Math.round(baseFontSize);
    const lineHeight = Math.round(fontSize * 1.45);
    const dpr = window.devicePixelRatio || 1;

    canvas.width = Math.ceil(cssWidth * dpr);
    canvas.height = Math.ceil((lines.length * lineHeight + padding * 2) * dpr);

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    ctx.scale(dpr, dpr);
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.textBaseline = 'top';
    ctx.fillStyle = '#1a2933';
    ctx.font = `${fontSize}px system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial`;

    for (let i = 0; i < lines.length; i++) {
      const y = padding + i * lineHeight;
      const jitter = Math.round((Math.random() - 0.5) * 6);

      if (i === 0) {
        ctx.font = `bold ${Math.round(fontSize * 1)}px system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial`;
        ctx.fillStyle = '#0f1419';
      } else {
        ctx.font = `${fontSize}px system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial`;
        ctx.fillStyle = '#1a2933';
      }

      ctx.fillText(lines[i], padding + jitter, y);
    }
  }

  onMount(() => {
    if (!contactMount) return;

    const render = () => renderContact(contactMount!);
    render();

    let t: ReturnType<typeof setTimeout> | undefined;
    const handleResize = () => {
      if (t) clearTimeout(t);
      t = setTimeout(render, 120);
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  });
</script>

<div class="mx-auto w-full max-w-2xl px-6 py-16">
  <div class="rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-10 shadow-[0_0_40px_rgba(0,212,255,0.04)]">
    <h1 class="mb-8 text-2xl font-bold text-[#e2e8f0]">Impressum</h1>

    <section class="mb-8">
      <h2 class="mb-3 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">
        Angaben gemäß § 5 TMG
      </h2>
      <div bind:this={contactMount} class="min-h-[3.5rem]" aria-label="impressum-contact"></div>
      <noscript>
        <p class="text-sm leading-relaxed text-[#e2e8f0]/80">
          Name and email are shown with JavaScript enabled.
        </p>
      </noscript>
    </section>

    <section class="mb-8">
      <h2 class="mb-3 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">Kontakt</h2>
      <p class="text-sm text-[#e2e8f0]/80">The email address is shown above for human readers.</p>
    </section>

    <section class="mb-8">
      <h2 class="mb-3 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">
        Disclaimer / Haftungsausschluss
      </h2>
      <p class="text-sm leading-relaxed text-[#e2e8f0]/60">
        My SCID is an <strong class="text-[#e2e8f0]/80">unofficial</strong> fan project and is not
        affiliated with, authorized by, or endorsed by Cloud Imperium Games Corporation or Roberts
        Space Industries Corp. Star Citizen® is a registered trademark of Cloud Imperium Rights LLC.
      </p>
      <p class="mt-3 text-sm leading-relaxed text-[#e2e8f0]/60">
        This site uses the Star Citizen visual aesthetic (colors, typography) without embedding RSI
        copyrighted assets. No subscription fees are charged. No advertising revenue is generated.
      </p>
    </section>

    <section>
      <h2 class="mb-3 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">
        Datenschutz / Privacy
      </h2>
      <p class="text-sm leading-relaxed text-[#e2e8f0]/60">
        My SCID stores only what is necessary for identity verification: your SCID username,
        passkey credential, and publicly-visible RSI profile data (handle, citizen record, enlistment
        date, bio). No personal data is sold or shared with third parties. Data is stored on
        EU-based infrastructure. You can delete your account at any time from your profile page.
      </p>
    </section>
  </div>
</div>
