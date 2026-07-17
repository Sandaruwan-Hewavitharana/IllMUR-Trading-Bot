export namespace config {
	
	export class Config {
	    gemini_keys: string[];
	    mistral_keys: string[];
	    chrome_user_data_dir: string;
	    chrome_profile_name: string;
	    tradingview_url: string;
	    chart_render_wait_secs: number;
	    loop_interval_minutes: number;
	    mt5_endpoint: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gemini_keys = source["gemini_keys"];
	        this.mistral_keys = source["mistral_keys"];
	        this.chrome_user_data_dir = source["chrome_user_data_dir"];
	        this.chrome_profile_name = source["chrome_profile_name"];
	        this.tradingview_url = source["tradingview_url"];
	        this.chart_render_wait_secs = source["chart_render_wait_secs"];
	        this.loop_interval_minutes = source["loop_interval_minutes"];
	        this.mt5_endpoint = source["mt5_endpoint"];
	    }
	}

}

