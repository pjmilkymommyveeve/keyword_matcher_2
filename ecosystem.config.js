module.exports = {
  apps: [{
    name: 'keyword-matcher_2',
    script: './main',
    cwd: '/root/keyword_matcher_2',
    instances: 1,
    autorestart: true,
    watch: false,
    max_memory_restart: '500M',
    env: {
      PORT: 8050,
      NODE_ENV: 'production'
    },
    error_file: '/var/log/pm2/keyword-matcher-2-error.log',
    out_file: '/var/log/pm2/keyword-matcher-2-out.log',
    log_file: '/var/log/pm2/keyword-matcher-2-combined.log',
    time: true,
    merge_logs: true,
    // Restart delay
    restart_delay: 4000,
    // Max restart attempts
    max_restarts: 10,
    min_uptime: '10s'
  }]
};