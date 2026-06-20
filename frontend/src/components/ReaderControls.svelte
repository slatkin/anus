<script>
  import { BookOpen, Bookmark, ExternalLink, EyeOff, Minus, Plus } from 'lucide-svelte';
  import { createEventDispatcher } from 'svelte';

  export let entry = null;
  export let originalActive = false;
  export let keptUnread = new Set();

  const dispatch = createEventDispatcher();
</script>

<div class="reader-controls">
  <button class="ctrl-btn" class:active={originalActive}
          on:click={() => dispatch('fetchOriginal')} title="Readability mode">
    <BookOpen size={14}/>
  </button>
  <div class="ctrl-sep"></div>
  <button class="ctrl-btn" on:click={() => dispatch('decreaseFontSize')} title="Decrease font size"><Minus size={13}/></button>
  <span class="ctrl-label">A</span>
  <button class="ctrl-btn" on:click={() => dispatch('increaseFontSize')} title="Increase font size"><Plus size={13}/></button>
  <div class="ctrl-sep"></div>
  <button class="ctrl-btn"
          class:active={entry && keptUnread.has(entry.id)}
          on:click={() => dispatch('mailClick')}
          title={entry?.status === 'unread' ? 'Keep unread' : 'Mark as unread'}>
    <EyeOff size={14}/>
  </button>
  <button class="ctrl-btn" on:click={() => dispatch('save')} title="Save to Miniflux"><Bookmark size={14}/></button>
  <button class="ctrl-btn" on:click={() => dispatch('openBrowser')} title="Open in browser"><ExternalLink size={14}/></button>
</div>

<style>
  .reader-controls {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 0;
    margin-bottom: 20px;
    border-top: 1px dotted #c4a882;
    margin-top: 10px;
  }
  .ctrl-sep { width: 1px; height: 14px; background: #c4a882; margin: 0 2px; }
  .ctrl-label { font-size: 13px; color: #9a7a58; font-weight: 600; line-height: 1; }
  .ctrl-btn {
    background: none;
    border: none;
    padding: 3px 4px;
    color: #9a7a58;
    cursor: pointer;
    border-radius: 3px;
    display: flex;
    align-items: center;
  }
  .ctrl-btn:hover { background: #2a1f14; color: #c4a882; }
  .ctrl-btn.active { background: #9a7a58; color: #fdf6e3; border-radius: 4px; }
</style>
