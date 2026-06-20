<script>
  import { createEventDispatcher } from 'svelte';
  export let item;
  export let selected = false;
  export let open = false;
  export let windowFocused = true;
  export let el = null;

  const dispatch = createEventDispatcher();
</script>

<div
  class="nav-item"
  class:window-focused={windowFocused}
  role="button"
  tabindex="0"
  class:selected
  class:open
  bind:this={el}
  on:click={() => dispatch('select', item.cursorIdx)}
  on:keydown={e => e.key === 'Enter' && dispatch('select', item.cursorIdx)}
>
  <div class="nav-title" class:unread={item.unread}>{item.title}</div>
  <div class="nav-sub">{item.sub}</div>
</div>

<style>
  .nav-item {
    padding: 9px 14px 8px;
    border-bottom: 1px solid #24283b;
    cursor: pointer;
    user-select: none;
    transition: background 0.08s;
  }
  .nav-item.window-focused:hover { background: #24283b; }
  .nav-item.selected .nav-title { color: #c0caf5; }
  .nav-item.open { box-shadow: inset 5px 0 0 #73daca; }
  .nav-item.open .nav-title { color: #73daca; }

  .nav-title {
    font-size: 13px;
    font-weight: 400;
    line-height: 1.35;
    color: #737aa2;
    word-break: break-word;
  }
  .nav-title.unread {
    font-weight: 400;
    color: #c0caf5;
  }

  .nav-sub {
    font-size: 11px;
    color: #414868;
    margin-top: 3px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .nav-item.selected .nav-sub { color: #7a89b8; }
  .nav-item.open     .nav-sub { color: #6b7499; }
</style>
