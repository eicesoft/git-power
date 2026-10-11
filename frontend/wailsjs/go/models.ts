export namespace main {
	
	export class Branch {
	    name: string;
	    ref: string;
	    hash: string;
	    upstream: string;
	    remote: boolean;
	    current: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Branch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ref = source["ref"];
	        this.hash = source["hash"];
	        this.upstream = source["upstream"];
	        this.remote = source["remote"];
	        this.current = source["current"];
	    }
	}
	export class Change {
	    path: string;
	    originalPath: string;
	    status: string;
	    staged: boolean;
	    unstaged: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Change(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.originalPath = source["originalPath"];
	        this.status = source["status"];
	        this.staged = source["staged"];
	        this.unstaged = source["unstaged"];
	    }
	}
	export class Commit {
	    hash: string;
	    shortHash: string;
	    author: string;
	    date: string;
	    subject: string;
	    decorations: string;
	    parents: string[];
	
	    static createFrom(source: any = {}) {
	        return new Commit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.shortHash = source["shortHash"];
	        this.author = source["author"];
	        this.date = source["date"];
	        this.subject = source["subject"];
	        this.decorations = source["decorations"];
	        this.parents = source["parents"];
	    }
	}
	export class CommitFile {
	    path: string;
	    originalPath: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new CommitFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.originalPath = source["originalPath"];
	        this.status = source["status"];
	    }
	}
	export class CommitDetail {
	    hash: string;
	    shortHash: string;
	    author: string;
	    date: string;
	    subject: string;
	    body: string;
	    files: CommitFile[];
	
	    static createFrom(source: any = {}) {
	        return new CommitDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.shortHash = source["shortHash"];
	        this.author = source["author"];
	        this.date = source["date"];
	        this.subject = source["subject"];
	        this.body = source["body"];
	        this.files = this.convertValues(source["files"], CommitFile);
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
	
	export class GitCommandLog {
	    id: number;
	    command: string;
	    output: string;
	    error: string;
	    startedAt: string;
	    duration: number;
	    success: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GitCommandLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.command = source["command"];
	        this.output = source["output"];
	        this.error = source["error"];
	        this.startedAt = source["startedAt"];
	        this.duration = source["duration"];
	        this.success = source["success"];
	    }
	}
	export class Repository {
	    path: string;
	    name: string;
	    currentBranch: string;
	    aheadCommits: number;
	    detached: boolean;
	    remotes: string[];
	    branches: Branch[];
	    commits: Commit[];
	    changes: Change[];
	
	    static createFrom(source: any = {}) {
	        return new Repository(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.currentBranch = source["currentBranch"];
	        this.aheadCommits = source["aheadCommits"];
	        this.detached = source["detached"];
	        this.remotes = source["remotes"];
	        this.branches = this.convertValues(source["branches"], Branch);
	        this.commits = this.convertValues(source["commits"], Commit);
	        this.changes = this.convertValues(source["changes"], Change);
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

