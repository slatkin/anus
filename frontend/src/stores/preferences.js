import { writable } from 'svelte/store';

function persisted(key, defaultValue, parse = v => v) {
  const raw = localStorage.getItem(key);
  const initial = raw !== null ? parse(raw) : defaultValue;
  const store = writable(initial);
  store.subscribe(val => localStorage.setItem(key, typeof val === 'object' ? JSON.stringify([...val]) : String(val)));
  return store;
}

export const showRead      = persisted('showRead',      true,  v => v !== 'false');
export const sortOldest    = persisted('sortOldest',    false, v => v === 'true');
export const grouped       = persisted('grouped',       true,  v => v !== 'false');
export const groupedCats   = persisted('groupedCats',   false, v => v === 'true');
export const showScrollbar = persisted('showScrollbar', false, v => v === 'true');
export const navWidth      = persisted('navWidth',      300,   v => parseInt(v, 10));
export const navCollapsed  = persisted('navCollapsed',  false, v => v === 'true');
export const fontSize      = persisted('readerFontSize',16,    v => parseInt(v, 10));

export const collapsedFeeds = persisted(
  'collapsedFeeds',
  new Set(),
  v => new Set(JSON.parse(v))
);

export const keptUnread = persisted(
  'keptUnread',
  new Set(),
  v => new Set(JSON.parse(v))
);
