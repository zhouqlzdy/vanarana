CREATE TABLE IF NOT EXISTS vanarana_repositories (
    id          INT AUTO_INCREMENT PRIMARY KEY,
    repo_url    VARCHAR(512) NOT NULL,
    name        VARCHAR(255) NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_repo_url (repo_url)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS vanarana_pipeline_runs (
    id                  INT AUTO_INCREMENT PRIMARY KEY,
    repo_id             INT NOT NULL,
    pipeline_job_name   VARCHAR(255) NOT NULL,
    branch              VARCHAR(255) NOT NULL DEFAULT '',
    commit_hash         VARCHAR(64) NOT NULL DEFAULT '',
    build_id            VARCHAR(128) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'processing',
    triggered_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repo_id) REFERENCES vanarana_repositories(id),
    UNIQUE KEY uk_run (repo_id, pipeline_job_name, build_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS vanarana_module_reports (
    id              INT AUTO_INCREMENT PRIMARY KEY,
    pipeline_run_id INT NOT NULL,
    module_name     VARCHAR(255) NOT NULL,
    junit_archive   VARCHAR(512) NOT NULL DEFAULT '',
    jacoco_archive  VARCHAR(512) NOT NULL DEFAULT '',
    status          VARCHAR(20) NOT NULL DEFAULT 'processing',
    upload_time     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (pipeline_run_id) REFERENCES vanarana_pipeline_runs(id) ON DELETE CASCADE,
    UNIQUE KEY uk_module (pipeline_run_id, module_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS vanarana_junit_metrics (
    id           INT AUTO_INCREMENT PRIMARY KEY,
    report_id    INT NOT NULL UNIQUE,
    total_tests  INT NOT NULL DEFAULT 0,
    failures     INT NOT NULL DEFAULT 0,
    ignored      INT NOT NULL DEFAULT 0,
    duration_ms  BIGINT NOT NULL DEFAULT 0,
    success_rate DOUBLE NOT NULL DEFAULT 0,
    packages     JSON,
    FOREIGN KEY (report_id) REFERENCES vanarana_module_reports(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS vanarana_jacoco_metrics (
    id                    INT AUTO_INCREMENT PRIMARY KEY,
    report_id             INT NOT NULL UNIQUE,
    instruction_coverage  DOUBLE NOT NULL DEFAULT 0,
    branch_coverage       DOUBLE NOT NULL DEFAULT 0,
    line_coverage         DOUBLE NOT NULL DEFAULT 0,
    method_coverage       DOUBLE NOT NULL DEFAULT 0,
    lines_total           INT NOT NULL DEFAULT 0,
    lines_missed          INT NOT NULL DEFAULT 0,
    packages              JSON,
    FOREIGN KEY (report_id) REFERENCES vanarana_module_reports(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
