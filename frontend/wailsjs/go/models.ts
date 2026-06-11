export namespace main {
	
	export class DefaultConfig {
	    port: string;
	    threadCount: string;
	    intervalMs: string;
	    failureLimit: string;
	    successLimit: string;
	
	    static createFrom(source: any = {}) {
	        return new DefaultConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.threadCount = source["threadCount"];
	        this.intervalMs = source["intervalMs"];
	        this.failureLimit = source["failureLimit"];
	        this.successLimit = source["successLimit"];
	    }
	}

}

