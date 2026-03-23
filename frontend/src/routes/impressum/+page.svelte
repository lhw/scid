<script lang="ts">
  // Contact details are split across variables to reduce indexing by bots.
  // Displayed inline via CSS so it reads naturally for humans.
  const nameParts = ['L.', '\u00a0', 'W\u200beller'];
  const emailUser = 'admin';
  const emailDomain = 'scid' + '\u2024' + 'my';
</script>

<div class="mx-auto w-full max-w-2xl px-6 py-16">
  <div class="rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-10 shadow-[0_0_40px_rgba(0,212,255,0.04)]">
    <h1 class="mb-8 text-2xl font-bold text-[#e2e8f0]">Impressum</h1>

    <section class="mb-8">
      <h2 class="mb-3 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">
        Angaben gemäß § 5 TMG
      </h2>
      <p class="text-sm leading-relaxed text-[#e2e8f0]/80">
        {nameParts.join('')}<br />
        Privatperson / private individual
      </p>
    </section>

    <section class="mb-8">
      <h2 class="mb-3 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">Kontakt</h2>
      <p class="text-sm text-[#e2e8f0]/80">
        E-Mail:
        <!-- Address split to deter scrapers; reads as emailUser @ emailDomain -->
        <span class="font-mono text-[#00d4ff]"
          >{emailUser}<!-- noscript --><span aria-hidden="true" class="select-none opacity-0 w-0 overflow-hidden inline-block">REMOVEME</span>@<!-- -->{emailDomain.replace('\u2024', '.')}</span
        >
      </p>
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
