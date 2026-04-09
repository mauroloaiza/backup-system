export namespace main {
	
	export class BackupStats {
	    last_run: string;
	    next_run: string;
	    last_job_id: string;
	    last_files: number;
	    last_bytes: number;
	    last_duration: string;
	    last_errors: number;
	    total_jobs: number;
	
	    static createFrom(source: any = {}) {
	        return new BackupStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.last_run = source["last_run"];
	        this.next_run = source["next_run"];
	        this.last_job_id = source["last_job_id"];
	        this.last_files = source["last_files"];
	        this.last_bytes = source["last_bytes"];
	        this.last_duration = source["last_duration"];
	        this.last_errors = source["last_errors"];
	        this.total_jobs = source["total_jobs"];
	    }
	}
	export class ServiceStatus {
	    installed: boolean;
	    running: boolean;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.running = source["running"];
	        this.status = source["status"];
	    }
	}
	export class UIConfig {
	    server_url: string;
	    api_token: string;
	    source_paths: string[];
	    exclude_patterns: string[];
	    encryption_passphrase: string;
	    use_vss: boolean;
	    incremental: boolean;
	    schedule_interval: string;
	    retention_days: number;
	    dest_type: string;
	    local_path: string;
	    s3_bucket: string;
	    s3_region: string;
	    s3_prefix: string;
	    sftp_host: string;
	    sftp_path: string;
	    sftp_user: string;
	    log_level: string;
	    log_path: string;
	    config_path: string;
	
	    static createFrom(source: any = {}) {
	        return new UIConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server_url = source["server_url"];
	        this.api_token = source["api_token"];
	        this.source_paths = source["source_paths"];
	        this.exclude_patterns = source["exclude_patterns"];
	        this.encryption_passphrase = source["encryption_passphrase"];
	        this.use_vss = source["use_vss"];
	        this.incremental = source["incremental"];
	        this.schedule_interval = source["schedule_interval"];
	        this.retention_days = source["retention_days"];
	        this.dest_type = source["dest_type"];
	        this.local_path = source["local_path"];
	        this.s3_bucket = source["s3_bucket"];
	        this.s3_region = source["s3_region"];
	        this.s3_prefix = source["s3_prefix"];
	        this.sftp_host = source["sftp_host"];
	        this.sftp_path = source["sftp_path"];
	        this.sftp_user = source["sftp_user"];
	        this.log_level = source["log_level"];
	        this.log_path = source["log_path"];
	        this.config_path = source["config_path"];
	    }
	}

}

