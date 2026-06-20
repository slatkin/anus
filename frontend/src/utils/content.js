export function processContent(html) {
  if (!html) return html;
  html = html.replace(/<a\b[^>]*>\s*View Image in Fullscreen\s*<\/a>/gi, '');
  // Strip width/height attrs so hotlink-protected images (200 OK, blank body)
  // don't reserve space proportional to their declared dimensions.
  html = html.replace(/<img(\s[^>]*?)>/gi, (_, attrs) => {
    const cleaned = attrs.replace(/\s+(width|height)=["'][^"']*["']/gi, '');
    return `<img${cleaned}>`;
  });
  html = html.replace(
    /<iframe[^>]*src=["']https?:\/\/(?:www\.)?youtube(?:-nocookie)?\.com\/embed\/([a-zA-Z0-9_-]+)[^"']*["'][^>]*>(?:<\/iframe>)?/gi,
    (_, id) =>
      `<div class="yt-thumb" data-yt-url="https://www.youtube.com/watch?v=${id}">` +
      `<img src="https://img.youtube.com/vi/${id}/hqdefault.jpg" ` +
      `style="width:100%;aspect-ratio:16/9;object-fit:cover;display:block" alt="Watch on YouTube">` +
      `<span class="yt-play">▶ Watch on YouTube</span>` +
      `</div>`
  );
  // Clean up figure/figcaption: remove non-image nodes before <figcaption>
  // (some feeds duplicate caption text as siblings before the figcaption).
  const doc = new DOMParser().parseFromString('<body>' + html + '</body>', 'text/html');
  doc.querySelectorAll('figure').forEach(fig => {
    const cap = fig.querySelector(':scope > figcaption');
    if (!cap) return;
    for (const node of [...fig.childNodes]) {
      if (node === cap) break;
      const isImg = node.nodeType === 1 &&
        (node.tagName === 'IMG' || node.tagName === 'PICTURE' || node.tagName === 'VIDEO' ||
         (node.tagName === 'A' && node.querySelector('img, picture, video')));
      if (!isImg) node.remove();
    }
  });

  // Ars Technica (and similar feeds) emit images as bare <a><img></a> followed
  // by a plain-text caption node — no <figure> wrapper. The caption text also
  // repeats itself (sometimes multiple times) in the same text node.
  // Wrap each image link + its caption text in <figure><figcaption> so CSS
  // can style them, and strip the duplicated text.
  function deduplicateCaption(rawText) {
    const text = rawText.trim();
    const len = text.length;
    if (!len) return '';
    // Minimum prefix length to test — long enough to avoid coincidental matches.
    const minLen = Math.max(10, Math.floor(len * 0.08));
    for (let i = minLen; i <= Math.floor(len * 0.7); i++) {
      // If the text starting at position i begins with the same characters as
      // the text from position 0, the caption repeats — keep only the first copy.
      if (text.slice(i).trimStart().startsWith(text.slice(0, minLen))) {
        return text.slice(0, i).trimEnd();
      }
    }
    return text;
  }

  doc.querySelectorAll('a').forEach(link => {
    if (link.closest('figure')) return;              // already inside a figure
    if (!link.querySelector('img')) return;          // not an image link
    if (link.textContent.trim()) return;             // link has visible text (alt text etc.)
    const next = link.nextSibling;
    if (!next || next.nodeType !== 3) return;        // no following text node
    const caption = deduplicateCaption(next.textContent);
    if (!caption) return;

    const figure = doc.createElement('figure');
    link.parentNode.insertBefore(figure, link);
    figure.appendChild(link);
    const figcap = doc.createElement('figcaption');
    figcap.textContent = caption;
    figure.appendChild(figcap);
    next.remove();  // removes entire text node (incl. any orphan gallery captions within)
  });

  // Some feeds (e.g. plain-HTML sites) use <br> instead of <p> tags, which
  // produces very tight line spacing because our CSS only styles <p>. Walk
  // block-level containers and wrap inline segments (separated by <br>) into
  // <p> tags, while leaving existing block children in place.
  const IS_BLOCK = new Set(['P','DIV','H1','H2','H3','H4','H5','H6',
    'UL','OL','LI','TABLE','BLOCKQUOTE','FIGURE','FIGCAPTION','PRE',
    'SECTION','ARTICLE','HEADER','FOOTER','ASIDE','NAV']);
  doc.querySelectorAll('body, div, section, article').forEach(block => {
    // Only process blocks that have at least one direct <br> child.
    if (![...block.children].some(c => c.tagName === 'BR')) return;
    // Walk children: collect inline runs between <br> or block elements.
    const children = [...block.childNodes];
    const segments = []; // [{type:'inline'|'block', nodes:[]}]
    let run = [];
    for (const n of children) {
      const isBlock = n.nodeType === 1 && IS_BLOCK.has(n.tagName);
      const isBr    = n.nodeType === 1 && n.tagName === 'BR';
      if (isBr) {
        segments.push({type: 'inline', nodes: run}); run = [];
      } else if (isBlock) {
        if (run.length) { segments.push({type: 'inline', nodes: run}); run = []; }
        segments.push({type: 'block', nodes: [n]});
      } else {
        run.push(n);
      }
    }
    if (run.length) segments.push({type: 'inline', nodes: run});
    // Only rewrite if at least one inline segment has visible content.
    const hasInline = segments.some(s => s.type === 'inline' && s.nodes.some(n =>
      n.nodeType === 1 || (n.nodeType === 3 && n.textContent.trim())
    ));
    if (!hasInline) return;
    block.innerHTML = '';
    for (const seg of segments) {
      if (seg.type === 'block') {
        block.appendChild(seg.nodes[0]);
      } else {
        const hasContent = seg.nodes.some(n =>
          n.nodeType === 1 || (n.nodeType === 3 && n.textContent.trim())
        );
        if (hasContent) {
          const p = doc.createElement('p');
          seg.nodes.forEach(n => p.appendChild(n));
          block.appendChild(p);
        }
      }
    }
  });

  return doc.body.innerHTML;
}

const _HIGHLIGHT_SKIP = new Set(['SCRIPT', 'STYLE', 'PRE', 'CODE']);

export function highlightTerms(html, query) {
  const STOP = new Set(['the','in','of','a','an','is','it','to','and','or','for','on','at','by','as','be','was','are','has','had','have','but','not','this','that','with','from','its','than','then','into','over','also','after','before','about','so','if','do','no','up','out','can','all','any','my','we','you','he','she','they','his','her','our','your','their','what','who','which','when','where','how','will','just','been','one','would','could','should','may','might','more','most','some','such']);
  const esc = s => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const phrase = query.trim();
  const tokens = phrase.split(/\s+/).filter(w => w.length >= 4 && !STOP.has(w.toLowerCase()));
  if (!tokens.length && !phrase) return html;
  const parts = [esc(phrase), ...tokens.map(esc)].filter(Boolean);
  const pat = new RegExp(`(${parts.join('|')})`, 'gi');
  const doc = new DOMParser().parseFromString('<body>' + html + '</body>', 'text/html');
  const walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      let el = node.parentElement;
      while (el) {
        if (_HIGHLIGHT_SKIP.has(el.tagName)) return NodeFilter.FILTER_REJECT;
        el = el.parentElement;
      }
      return NodeFilter.FILTER_ACCEPT;
    }
  });
  const nodes = [];
  let n;
  while ((n = walker.nextNode())) nodes.push(n);
  for (const textNode of nodes) {
    const text = textNode.textContent;
    if (!pat.test(text)) { pat.lastIndex = 0; continue; }
    pat.lastIndex = 0;
    const frag = doc.createDocumentFragment();
    let last = 0, m;
    while ((m = pat.exec(text)) !== null) {
      if (m.index > last) frag.appendChild(doc.createTextNode(text.slice(last, m.index)));
      const mark = doc.createElement('mark');
      mark.textContent = m[0];
      frag.appendChild(mark);
      last = pat.lastIndex;
    }
    if (last < text.length) frag.appendChild(doc.createTextNode(text.slice(last)));
    textNode.parentNode.replaceChild(frag, textNode);
  }
  return doc.body.innerHTML;
}
