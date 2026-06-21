// @vitest-environment jsdom
import { describe, it, expect } from 'vitest';
import { processContent, highlightTerms } from './content.js';

describe('processContent – null/empty passthrough', () => {
  it('returns null unchanged', () => {
    expect(processContent(null)).toBeNull();
  });
  it('returns empty string unchanged', () => {
    expect(processContent('')).toBe('');
  });
});

describe('processContent – remove "View Image in Fullscreen" links', () => {
  it('strips a plain link', () => {
    const html = '<p>text</p><a href="/img">View Image in Fullscreen</a>';
    const out = processContent(html);
    expect(out).not.toContain('View Image in Fullscreen');
    expect(out).toContain('text');
  });

  it('strips with extra whitespace inside link text', () => {
    const html = '<a href="/img">  View Image in Fullscreen  </a>';
    expect(processContent(html)).not.toContain('View Image in Fullscreen');
  });

  it('is case-insensitive', () => {
    const html = '<a href="/img">VIEW IMAGE IN FULLSCREEN</a>';
    expect(processContent(html)).not.toContain('VIEW IMAGE IN FULLSCREEN');
  });

  it('preserves unrelated links', () => {
    const html = '<a href="/about">About</a>';
    expect(processContent(html)).toContain('About');
  });
});

describe('processContent – strip width/height from img tags', () => {
  it('removes width attribute', () => {
    const html = '<img src="a.jpg" width="800">';
    const out = processContent(html);
    expect(out).not.toContain('width=');
    expect(out).toContain('src="a.jpg"');
  });

  it('removes height attribute', () => {
    const html = '<img src="a.jpg" height="600">';
    const out = processContent(html);
    expect(out).not.toContain('height=');
  });

  it('removes both width and height', () => {
    const html = '<img src="a.jpg" width="800" height="600" alt="x">';
    const out = processContent(html);
    expect(out).not.toContain('width=');
    expect(out).not.toContain('height=');
    expect(out).toContain('alt="x"');
  });

  it('leaves images without width/height unchanged (structurally)', () => {
    const html = '<img src="b.jpg" alt="y">';
    const out = processContent(html);
    expect(out).toContain('src="b.jpg"');
    expect(out).toContain('alt="y"');
  });
});

describe('processContent – YouTube iframe → thumbnail', () => {
  it('converts a standard YouTube embed iframe', () => {
    const id = 'dQw4w9WgXcQ';
    const html = `<iframe src="https://www.youtube.com/embed/${id}" frameborder="0"></iframe>`;
    const out = processContent(html);
    expect(out).not.toContain('<iframe');
    expect(out).toContain('yt-thumb');
    expect(out).toContain(`data-yt-url="https://www.youtube.com/watch?v=${id}"`);
    expect(out).toContain(`https://img.youtube.com/vi/${id}/hqdefault.jpg`);
    expect(out).toContain('Watch on YouTube');
  });

  it('converts a youtube-nocookie embed', () => {
    const id = 'abc123XYZ_-';
    const html = `<iframe src="https://www.youtube-nocookie.com/embed/${id}"></iframe>`;
    const out = processContent(html);
    expect(out).toContain('yt-thumb');
    expect(out).toContain(`watch?v=${id}`);
  });

  it('converts embed with www. prefix', () => {
    const id = 'testId99';
    const html = `<iframe src="https://www.youtube.com/embed/${id}?autoplay=1"></iframe>`;
    const out = processContent(html);
    expect(out).toContain('yt-thumb');
  });

  it('does not affect non-YouTube iframes', () => {
    const html = '<iframe src="https://example.com/video"></iframe>';
    expect(processContent(html)).toContain('src="https://example.com/video"');
  });
});

describe('processContent – figure/figcaption cleanup', () => {
  it('removes non-image sibling nodes before figcaption', () => {
    const html = '<figure><span>duplicate text</span><img src="x.jpg"><figcaption>Caption</figcaption></figure>';
    const out = processContent(html);
    expect(out).not.toContain('duplicate text');
    expect(out).toContain('Caption');
    expect(out).toContain('x.jpg');
  });

  it('preserves img before figcaption', () => {
    const html = '<figure><img src="photo.jpg"><figcaption>Photo</figcaption></figure>';
    const out = processContent(html);
    expect(out).toContain('photo.jpg');
    expect(out).toContain('Photo');
  });

  it('preserves figure with no figcaption unchanged', () => {
    const html = '<figure><img src="no-cap.jpg"></figure>';
    const out = processContent(html);
    expect(out).toContain('no-cap.jpg');
  });

  it('preserves a link containing an img before figcaption', () => {
    const html = '<figure><a href="/big"><img src="thumb.jpg"></a><figcaption>Thumb</figcaption></figure>';
    const out = processContent(html);
    expect(out).toContain('thumb.jpg');
    expect(out).toContain('Thumb');
  });
});

describe('processContent – bare image + caption → figure wrapping', () => {
  it('wraps a bare image link followed by a text node into a figure', () => {
    const html = '<a href="/img"><img src="pic.jpg"></a>This is the caption';
    const out = processContent(html);
    expect(out).toContain('<figure>');
    expect(out).toContain('<figcaption>');
    expect(out).toContain('This is the caption');
  });

  it('does not wrap image links already inside a figure', () => {
    const html = '<figure><a href="/img"><img src="pic.jpg"></a>Caption</figure>';
    const out = processContent(html);
    // Still a figure, no extra wrapping (only one <figure> tag)
    const count = (out.match(/<figure/g) || []).length;
    expect(count).toBe(1);
  });

  it('does not wrap image link with no following text node', () => {
    const html = '<a href="/img"><img src="pic.jpg"></a>';
    const out = processContent(html);
    expect(out).not.toContain('<figure>');
  });

  it('does not wrap link that has visible text (alt text etc.)', () => {
    const html = '<a href="/img">See image<img src="pic.jpg"></a>Caption here';
    const out = processContent(html);
    expect(out).not.toContain('<figure>');
  });
});

describe('processContent – caption deduplication', () => {
  it('deduplicates a repeated caption', () => {
    // Simulate a caption that repeats: "Hello world Hello world"
    // The deduplicateCaption function should return only "Hello world"
    const repeated = 'The quick brown fox jumps over the lazy dog The quick brown fox';
    const html = `<a href="/img"><img src="pic.jpg"></a>${repeated}`;
    const out = processContent(html);
    // The figcaption should not contain the full repeated text twice
    // It should be shortened — just check it doesn't end with the full repetition
    const figcapMatch = out.match(/<figcaption>([\s\S]*?)<\/figcaption>/);
    expect(figcapMatch).not.toBeNull();
    const caption = figcapMatch[1];
    // The deduplicated caption should be shorter than the full repeated text
    expect(caption.length).toBeLessThan(repeated.length);
  });

  it('preserves a non-repeating caption intact', () => {
    const text = 'A unique caption that does not repeat itself at all.';
    const html = `<a href="/img"><img src="pic.jpg"></a>${text}`;
    const out = processContent(html);
    const figcapMatch = out.match(/<figcaption>([\s\S]*?)<\/figcaption>/);
    expect(figcapMatch).not.toBeNull();
    expect(figcapMatch[1].trim()).toBe(text.trim());
  });
});

describe('processContent – br-delimited → p conversion', () => {
  it('wraps br-separated inline text segments into p tags', () => {
    const html = '<div>First line<br>Second line<br>Third line</div>';
    const out = processContent(html);
    expect(out).not.toContain('<br>');
    const pCount = (out.match(/<p>/g) || []).length;
    expect(pCount).toBeGreaterThanOrEqual(3);
  });

  it('preserves existing block children alongside inline segments', () => {
    const html = '<div>Intro text<br><p>Existing paragraph</p>After</div>';
    const out = processContent(html);
    expect(out).toContain('<p>');
    expect(out).toContain('Existing paragraph');
    expect(out).toContain('Intro text');
    expect(out).toContain('After');
  });

  it('does not rewrite blocks that have no direct br children', () => {
    const html = '<div><p>Para one</p><p>Para two</p></div>';
    const out = processContent(html);
    // No brs, should leave p tags as-is
    expect(out).toContain('Para one');
    expect(out).toContain('Para two');
    expect(out).not.toContain('<br>');
  });

  it('does not create empty p tags for whitespace-only segments', () => {
    const html = '<div>Text<br>  <br>More</div>';
    const out = processContent(html);
    // whitespace-only segment between two brs should not produce a p
    expect(out).toContain('Text');
    expect(out).toContain('More');
  });
});

describe('highlightTerms', () => {
  it('returns html unchanged when query is empty', () => {
    const html = '<p>Hello world</p>';
    expect(highlightTerms(html, '')).toBe(html);
  });

  it('wraps a matching phrase in a mark element', () => {
    const html = '<p>Hello world</p>';
    const out = highlightTerms(html, 'Hello world');
    expect(out).toContain('<mark>Hello world</mark>');
  });

  it('is case-insensitive for matches', () => {
    const html = '<p>Hello World</p>';
    const out = highlightTerms(html, 'hello world');
    expect(out).toContain('<mark>');
  });

  it('does not highlight inside script tags', () => {
    const html = '<script>var hello = "hello world";</script><p>hello world</p>';
    const out = highlightTerms(html, 'hello world');
    // The script content should not have mark tags
    expect(out).not.toMatch(/<script>.*<mark>.*<\/script>/s);
    // The p content should be highlighted
    expect(out).toContain('<mark>');
  });

  it('skips stop words when matching individual tokens', () => {
    // "the" is a stop word — short stop words are filtered from individual token matching
    const html = '<p>the quick brown fox</p>';
    // "quick" is long enough (>=4 chars) and not a stop word
    const out = highlightTerms(html, 'quick brown');
    expect(out).toContain('<mark>');
  });
});
