export namespace app {
	
	export class FeedSummary {
	    feed_id: number;
	    feed_title: string;
	    unread: number;
	
	    static createFrom(source: any = {}) {
	        return new FeedSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.feed_id = source["feed_id"];
	        this.feed_title = source["feed_title"];
	        this.unread = source["unread"];
	    }
	}
	export class FetchResult {
	    entries: miniflux.FeedEntry[];
	    feeds: FeedSummary[];
	    remember_read_position: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FetchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entries = this.convertValues(source["entries"], miniflux.FeedEntry);
	        this.feeds = this.convertValues(source["feeds"], FeedSummary);
	        this.remember_read_position = source["remember_read_position"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace miniflux {
	
	export class Feed {
	    id: number;
	    title: string;
	    site_url: string;
	    feed_url: string;
	
	    static createFrom(source: any = {}) {
	        return new Feed(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.site_url = source["site_url"];
	        this.feed_url = source["feed_url"];
	    }
	}
	export class FeedEntry {
	    id: number;
	    feed_id: number;
	    title: string;
	    url: string;
	    content: string;
	    feed: Feed;
	    status: string;
	    starred: boolean;
	    // Go type: time
	    published_at: any;
	    // Go type: time
	    fetched_at?: any;
	    original_content?: string;
	
	    static createFrom(source: any = {}) {
	        return new FeedEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.feed_id = source["feed_id"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.content = source["content"];
	        this.feed = this.convertValues(source["feed"], Feed);
	        this.status = source["status"];
	        this.starred = source["starred"];
	        this.published_at = this.convertValues(source["published_at"], null);
	        this.fetched_at = this.convertValues(source["fetched_at"], null);
	        this.original_content = source["original_content"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

