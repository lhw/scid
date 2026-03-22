// See https://svelte.dev/docs/kit/types#app.d.ts
declare global {
  interface Window {
    __SCID_PUBLIC_ENV__?: {
      PUBLIC_POCKET_ID_URL?: string;
      PUBLIC_OIDC_CLIENT_ID?: string;
      PUBLIC_TURNSTILE_SITE_KEY?: string;
    };
  }

  namespace App {
    // interface Error {}
    // interface Locals {}
    // interface PageData {}
    interface PageState {
      secret?: string;
    }
    // interface Platform {}
  }
}

export {};
