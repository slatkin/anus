export function processContent(html) {
  if (!html) return html;
  html = stripFullscreenLinks(html);
  html = stripImageDimensions(html);
  html = replaceYouTubeEmbeds(html);
  const doc = new DOMParser().parseFromString('<body>' + html + '</body>', 'text/html');
  cleanFigureCaptions(doc);
  wrapBareImageFigures(doc);
  wrapBrParagraphs(doc);
  return doc.body.innerHTML;
}

function stripFullscreenLinks(html) {
  return html.replace(/<a\b[^>]*>\s*View Image in Fullscreen\s*<\/a>/gi, '');
}

// Strip width/height attrs so hotlink-protected images (200 OK, blank body)
// don't reserve space proportional to their declared dimensions.
function stripImageDimensions(html) {
  return html.replace(/<img(\s[^>]*?)>/gi, (_, attrs) => {
    const cleaned = attrs.replace(/\s+(width|height)=["'][^"']*["']/gi, '');
    return `<img${cleaned}>`;
  });
}

function replaceYouTubeEmbeds(html) {
  return html.replace(
    /<iframe[^>]*src=["']https?:\/\/(?:www\.)?youtube(?:-nocookie)?\.com\/embed\/([a-zA-Z0-9_-]+)[^"']*["'][^>]*>(?:<\/iframe>)?/gi,
    (_, id) =>
      `<div class="yt-thumb" data-yt-url="https://www.youtube.com/watch?v=${id}">` +
      `<img src="https://img.youtube.com/vi/${id}/hqdefault.jpg" ` +
      `style="width:100%;aspect-ratio:16/9;object-fit:cover;display:block" alt="Watch on YouTube">` +
      `<span class="yt-play">▶ Watch on YouTube</span>` +
      `</div>`
  );
}

// Removes non-image nodes that appear before <figcaption> inside <figure>.
// Some feeds duplicate caption text as siblings before the figcaption.
function cleanFigureCaptions(doc) {
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
}

// Returns the first non-repeating prefix of rawText, stripping duplicated captions.
// Minimum prefix length avoids coincidental matches on short strings.
function deduplicateCaption(rawText) {
  const text = rawText.trim();
  const len = text.length;
  if (!len) return '';
  const minLen = Math.max(10, Math.floor(len * 0.08));
  for (let i = minLen; i <= Math.floor(len * 0.7); i++) {
    if (text.slice(i).trimStart().startsWith(text.slice(0, minLen))) {
      return text.slice(0, i).trimEnd();
    }
  }
  return text;
}

// Wraps bare <a><img></a> links followed by a text node into <figure><figcaption>.
// Handles feeds (e.g. Ars Technica) that emit image links without <figure> markup,
// and strips repeated caption text within the same text node.
function wrapBareImageFigures(doc) {
  doc.querySelectorAll('a').forEach(link => {
    if (link.closest('figure')) return;
    if (!link.querySelector('img')) return;
    if (link.textContent.trim()) return;
    const next = link.nextSibling;
    if (!next || next.nodeType !== 3) return;
    const caption = deduplicateCaption(next.textContent);
    if (!caption) return;

    const figure = doc.createElement('figure');
    link.parentNode.insertBefore(figure, link);
    figure.appendChild(link);
    const figcap = doc.createElement('figcaption');
    figcap.textContent = caption;
    figure.appendChild(figcap);
    next.remove();
  });
}

const IS_BLOCK = new Set(['P','DIV','H1','H2','H3','H4','H5','H6',
  'UL','OL','LI','TABLE','BLOCKQUOTE','FIGURE','FIGCAPTION','PRE',
  'SECTION','ARTICLE','HEADER','FOOTER','ASIDE','NAV']);

// Converts <br>-separated inline runs into <p> tags within block containers.
// Some feeds use <br> instead of <p>, producing tight spacing with our CSS.
function wrapBrParagraphs(doc) {
  doc.querySelectorAll('body, div, section, article').forEach(block => {
    if (![...block.children].some(c => c.tagName === 'BR')) return;
    const children = [...block.childNodes];
    const segments = [];
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
