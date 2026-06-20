<script>
  import { fade, fly } from 'svelte/transition';
  import { showScrollbar } from '../stores/preferences.js';
  import { createEventDispatcher } from 'svelte';

  export let cfg = null;
  export let saving = false;
  export let clearing = false;

  const dispatch = createEventDispatcher();
</script>

<div class="settings-backdrop" role="presentation"
  on:click|self={() => dispatch('close')}
  on:keydown={e => e.key === 'Escape' && dispatch('close')}
  transition:fade={{ duration: 150 }}>
  <div class="settings-modal" transition:fly={{ y: 20, duration: 180 }}>
    <div class="settings-header">
      <span class="settings-title">Settings <span class="settings-version">v{__APP_VERSION__}</span></span>
      <button class="settings-close" on:click={() => dispatch('close')}>✕</button>
    </div>
    <div class="settings-body">
      <label class="settings-label settings-row">
        <span>Display scrollbar in feed list</span>
        <button class="settings-toggle" class:on={$showScrollbar} on:click={() => $showScrollbar = !$showScrollbar} role="switch" aria-checked={$showScrollbar} aria-label="Display scrollbar in feed list"></button>
      </label>
      <label class="settings-label">
        <span>Cache expiry (days)</span>
        <input class="settings-input settings-input-sm" type="number" min="1" bind:value={cfg.cache_expiry_days}/>
      </label>
      <label class="settings-label">
        <span>Polling interval (minutes, 0 = off)</span>
        <input class="settings-input settings-input-sm" type="number" min="0" bind:value={cfg.polling_interval_minutes}/>
      </label>
    </div>
    <div class="settings-footer">
      <button class="settings-clear-cache" on:click={() => dispatch('clearCache')} disabled={clearing}>
        {clearing ? 'Clearing…' : 'Clear cache'}
      </button>
      <button class="settings-save" on:click={() => dispatch('save')} disabled={saving}>
        {saving ? 'Saving…' : 'Save'}
      </button>
    </div>
  </div>
</div>

<style>
  .settings-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.55);
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .settings-modal {
    background: #1a1b26;
    border: 1px solid #414868;
    border-radius: 8px;
    width: 420px;
    max-width: 95vw;
    max-height: 85vh;
    display: flex;
    flex-direction: column;
    box-shadow: 0 8px 32px rgba(0,0,0,0.5);
  }

  .settings-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    border-bottom: 1px solid #414868;
    flex-shrink: 0;
  }

  .settings-title {
    font-size: 15px;
    font-weight: 500;
    color: #fdfdfd;
    letter-spacing: 0.03em;
  }

  .settings-close {
    background: none;
    border: none;
    cursor: pointer;
    color: #737aa2;
    font-size: 14px;
    padding: 2px 6px;
    border-radius: 4px;
    transition: color 80ms, background 80ms;
  }
  .settings-close:hover { background: #24283b; color: #c0caf5; }

  .settings-body {
    overflow-y: auto;
    padding: 14px 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    flex: 1;
  }

  .settings-label {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 13px;
    color: #fdfdfd;
    letter-spacing: 0.02em;
  }

  .settings-row {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }

  .settings-input {
    background: #24283b;
    border: 1px solid #414868;
    border-radius: 4px;
    color: #c0caf5;
    font-family: inherit;
    font-size: 14px;
    padding: 6px 10px;
    outline: none;
    transition: border-color 120ms;
  }
  .settings-input:focus { border-color: #7aa2f7; }
  .settings-input-sm { width: 80px; }

  .settings-toggle {
    position: relative;
    width: 36px;
    height: 20px;
    border-radius: 10px;
    border: none;
    background: #414868;
    cursor: pointer;
    flex-shrink: 0;
    transition: background 150ms;
    padding: 0;
  }
  .settings-toggle.on { background: #7aa2f7; }
  .settings-toggle::after {
    content: '';
    position: absolute;
    top: 3px;
    left: 3px;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: #fdfdfd;
    transition: transform 150ms;
  }
  .settings-toggle.on::after { transform: translateX(16px); }

  .settings-footer {
    padding: 10px 16px;
    border-top: 1px solid #414868;
    display: flex;
    justify-content: space-between;
    flex-shrink: 0;
  }

  .settings-version {
    font-size: 11px;
    font-weight: 400;
    color: #414868;
    margin-left: 6px;
  }

  .settings-save {
    background: #2d3f76;
    border: none;
    border-radius: 4px;
    color: #7aa2f7;
    cursor: pointer;
    font-family: inherit;
    font-size: 14px;
    font-weight: 500;
    padding: 6px 18px;
    transition: background 80ms, color 80ms;
  }
  .settings-save:hover:not(:disabled)  { background: #3d59a1; color: #c0caf5; }
  .settings-save:disabled { opacity: 0.5; cursor: default; }

  .settings-clear-cache {
    background: transparent;
    border: 1px solid #414868;
    border-radius: 4px;
    color: #565f89;
    cursor: pointer;
    font-family: inherit;
    font-size: 14px;
    font-weight: 500;
    padding: 6px 18px;
    transition: border-color 80ms, color 80ms;
  }
  .settings-clear-cache:hover:not(:disabled) { border-color: #7aa2f7; color: #7aa2f7; }
  .settings-clear-cache:disabled { opacity: 0.5; cursor: default; }
</style>
