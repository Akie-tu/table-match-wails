export namespace backend {
	
	export class EmailConfig {
	    sender_email: string;
	    auth_code: string;
	    smtp_host: string;
	    smtp_port: string;
	    sender_name: string;
	
	    static createFrom(source: any = {}) {
	        return new EmailConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sender_email = source["sender_email"];
	        this.auth_code = source["auth_code"];
	        this.smtp_host = source["smtp_host"];
	        this.smtp_port = source["smtp_port"];
	        this.sender_name = source["sender_name"];
	    }
	}
	export class EmailResult {
	    ok: boolean;
	    msg: string;
	
	    static createFrom(source: any = {}) {
	        return new EmailResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.msg = source["msg"];
	    }
	}
	export class FixedContent {
	    invoice_type: string;
	    tax_included: string;
	    item_name: string;
	    tax_code: string;
	    unit: string;
	    tax_rate: string;
	
	    static createFrom(source: any = {}) {
	        return new FixedContent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.invoice_type = source["invoice_type"];
	        this.tax_included = source["tax_included"];
	        this.item_name = source["item_name"];
	        this.tax_code = source["tax_code"];
	        this.unit = source["unit"];
	        this.tax_rate = source["tax_rate"];
	    }
	}
	export class ImgConvertResult {
	    total: number;
	    converted: number;
	    copied: number;
	    failed: number;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImgConvertResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.converted = source["converted"];
	        this.copied = source["copied"];
	        this.failed = source["failed"];
	        this.errors = source["errors"];
	    }
	}
	export class Invoice {
	    invoice_type: string;
	    tax_included: string;
	    is_natural: string;
	    buyer: string;
	    tax_id: string;
	    remark: string;
	    item_name: string;
	    tax_code: string;
	    unit: string;
	    qty: string;
	    amount: string;
	    tax_rate: string;
	
	    static createFrom(source: any = {}) {
	        return new Invoice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.invoice_type = source["invoice_type"];
	        this.tax_included = source["tax_included"];
	        this.is_natural = source["is_natural"];
	        this.buyer = source["buyer"];
	        this.tax_id = source["tax_id"];
	        this.remark = source["remark"];
	        this.item_name = source["item_name"];
	        this.tax_code = source["tax_code"];
	        this.unit = source["unit"];
	        this.qty = source["qty"];
	        this.amount = source["amount"];
	        this.tax_rate = source["tax_rate"];
	    }
	}
	export class ImportResult {
	    rows: Invoice[];
	    imported: number;
	    missing: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rows = this.convertValues(source["rows"], Invoice);
	        this.imported = source["imported"];
	        this.missing = source["missing"];
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
	
	export class InvoiceResult {
	    path: string;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new InvoiceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.errors = source["errors"];
	    }
	}
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

