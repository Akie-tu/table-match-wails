export namespace backend {
	
	export class MatchResult {
	    matched: number;
	    multi: number;
	    notfound: string[];
	    out_path: string;
	
	    static createFrom(source: any = {}) {
	        return new MatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.matched = source["matched"];
	        this.multi = source["multi"];
	        this.notfound = source["notfound"];
	        this.out_path = source["out_path"];
	    }
	}

}

