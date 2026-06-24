<script>
  import { createEventDispatcher } from 'svelte';
  import { BookCheck } from 'lucide-svelte';
  export let item;
  const dispatch = createEventDispatcher();
</script>

<div class="nav-feed-header" role="button" tabindex="0"
  data-feed-id={item.feedId}
  on:dblclick={() => dispatch('collapse', item.feedId)}
  on:keydown={e => e.key === 'Enter' && dispatch('collapse', item.feedId)}>
  <span class="feed-header-title">{item.title}</span>
  {#if item.count != null}<span class="feed-header-count">{item.count}</span>{/if}
  <button class="feed-header-markread" title="Mark all read"
    on:click|stopPropagation={() => dispatch('markread', item.feedId)}>
    <BookCheck size={14} />
  </button>
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
  .feed-header-markread {
    margin-left: 8px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    color: inherit;
    opacity: 0.6;
    transition: opacity 0.1s, transform 0.1s;
  }
  .feed-header-markread:hover {
    opacity: 1;
  }
  .feed-header-markread:active {
    transform: scale(0.85);
    opacity: 1;
  }
  .feed-header-count {
    font-size: 10px;
    color: #a9b1d6;
    margin-left: 6px;
    flex-shrink: 0;
  }
</style>
