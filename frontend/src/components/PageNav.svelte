<script>
  export let page = 0;
  export let totalPages = 1;
</script>

{#if totalPages > 1}
  <div class="page-nav">
    <button class="page-btn" disabled={page === 0}
            on:click={() => page--}>‹</button>
    <span class="page-indicator">{page + 1} / {totalPages}</span>
    <button class="page-btn" disabled={page === totalPages - 1}
            on:click={() => page++}>›</button>
  </div>
{/if}

<style>
  .page-nav {
    position: absolute;
    bottom: 16px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 8px;
    background: rgba(253, 246, 227, 0.85);
    backdrop-filter: blur(4px);
    border: 1px solid #d5c9a8;
    border-radius: 20px;
    padding: 4px 10px;
    user-select: none;
  }

  .page-btn {
    background: none;
    border: none;
    cursor: pointer;
    font-size: 20px;
    line-height: 1;
    color: #8a7355;
    padding: 0 4px;
    transition: color 80ms, transform 80ms;
  }
  .page-btn:disabled { color: #c9b89a; cursor: default; }
  .page-btn:not(:disabled):hover  { color: #3a2c1a; }
  .page-btn:not(:disabled):active { color: #3a2c1a; transform: scale(0.85); }

  .page-indicator {
    font-size: 12px;
    color: #7a6345;
    min-width: 40px;
    text-align: center;
  }
</style>
