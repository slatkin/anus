<script>
  import { fly } from 'svelte/transition';
  import { Search } from 'lucide-svelte';

  export let query = '';
  export let inputColor = '#e0af68';
  export let inputEl = null;

  import { createEventDispatcher } from 'svelte';
  const dispatch = createEventDispatcher();
</script>

<div class="search-bar" transition:fly={{ y: -20, duration: 150 }}>
  <input
    bind:this={inputEl}
    bind:value={query}
    on:input={() => dispatch('input')}
    on:keydown={e => {
      if (e.key === 'Enter') { e.preventDefault(); dispatch('search'); }
      else if (e.key === 'Escape') dispatch('close');
    }}
    placeholder="Search…"
    class="search-input"
    style="color: {inputColor}"
    type="search"
  />
  <button class="search-go-btn" on:click={() => dispatch('search')} title="Search">
    <Search size={14}/>
  </button>
</div>

<style>
  .search-bar {
    display: flex;
    align-items: stretch;
    gap: 0;
    padding: 6px 10px;
    background: #1a1b26;
    border-bottom: 1px solid #2a2b3d;
  }
  .search-input {
    flex: 1;
    min-width: 0;
    background: #24283b;
    border: 1px solid #414868;
    border-right: none;
    border-radius: 4px 0 0 4px;
    color: #c0caf5;
    font-family: inherit;
    font-size: 13px;
    font-weight: 300;
    padding: 4px 8px;
    outline: none;
  }
  .search-input::-webkit-search-cancel-button { display: none; }
  .search-go-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    aspect-ratio: 1;
    background: #24283b;
    border: 1px solid #414868;
    border-radius: 0 4px 4px 0;
    color: #7aa2f7;
    cursor: pointer;
    padding: 0 4px;
  }
  .search-go-btn:hover { background: #2a2b3d; color: #c0caf5; }
</style>
