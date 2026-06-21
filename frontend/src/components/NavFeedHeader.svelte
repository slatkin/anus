<script>
  import { createEventDispatcher } from 'svelte';
  export let item;
  const dispatch = createEventDispatcher();
</script>

<div class="nav-feed-header" role="button" tabindex="0"
  data-feed-id={item.feedId}
  on:dblclick={() => dispatch('collapse', item.feedId)}
  on:keydown={e => e.key === 'Enter' && dispatch('collapse', item.feedId)}>
  <span class="feed-header-title">{item.title}</span>
  {#if item.count != null}<span class="feed-header-count">{item.count}</span>{/if}
</div>

<style>
  .nav-feed-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    padding: 8px 14px 5px;
    font-size: 12px;
    font-weight: 400;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: #7aa2f7;
    background: #24283b;
    border-bottom: 1px solid #414868;
    position: sticky;
    top: 0;
    z-index: 1;
    cursor: default;
    user-select: none;
  }
  .feed-header-title {
    min-width: 0;
    flex: 1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .feed-header-count {
    font-size: 10px;
    color: #414868;
    margin-left: 6px;
    flex-shrink: 0;
  }
</style>
