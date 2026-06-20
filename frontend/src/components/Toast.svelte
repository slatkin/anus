<script>
  import { fade } from 'svelte/transition';
  export let msg = '';
  export let visible = false;
</script>

{#if visible}
  <div class="toast" transition:fade={{ duration: 150 }}>{msg}</div>
{/if}

<style>
  .toast {
    position: absolute;
    top: 16px;
    left: 50%;
    transform: translateX(-50%);
    background: rgba(36, 40, 59, 0.92);
    backdrop-filter: blur(4px);
    color: #c0caf5;
    font-size: 13px;
    padding: 8px 18px;
    border-radius: 20px;
    border: 1px solid #414868;
    white-space: nowrap;
    pointer-events: none;
    z-index: 20;
  }
</style>
