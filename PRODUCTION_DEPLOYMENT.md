# Flowa Production Deployment Guide

## Pre-Deployment Checklist

### 1. Environment Configuration ✅

**Create production .env file:**
```bash
cp .env.production.example .env
```

**Required environment variables:**
- `SMTP_HOST` - Your SMTP server (e.g., smtp.gmail.com)
- `SMTP_PORT` - SMTP port (usually 587 for TLS)
- `SMTP_USER` - SMTP username/email
- `SMTP_PASS` - SMTP password (use app-specific password for Gmail)
- `JWT_SECRET` - Strong random secret (min 32 characters)
- `PORT` - Server port (default: 8080)
- `ENV` - Environment (production/staging/development)

**Generate strong JWT secret:**
```bash
openssl rand -base64 32
```

---

### 2. Security Hardening ✅

**SMTP Credentials:**
- ✅ Stored in .env file (not in code)
- ✅ .env file added to .gitignore
- ⚠️ Use app-specific passwords for Gmail
- ⚠️ Rotate credentials regularly

**JWT Secrets:**
- ✅ Minimum 32 characters
- ⚠️ Different secrets for prod/staging/dev
- ⚠️ Never commit secrets to git
-⚠️ Rotate secrets periodically

**CORS Configuration:**
- ⚠️ Update WebSocket upgrader in vm.go to check origin
- ⚠️ Configure allowed origins for production
- ⚠️ Don't use `CheckOrigin: return true` in production

**bcrypt Configuration:**
- ✅ Using cost factor 10 (current)
- ℹ️ Can increase to 12 for higher security (slower)

---

### 3. Performance Optimization ✅

**Already Implemented:**
- ✅ VM pooling for zero-allocation execution
- ✅ Compiler pooling
- ✅ Integer caching (-4096 to 4096)
- ✅ Stack size optimized (256 instead of 2048)
- ✅ Request VM cloning for concurrent HTTP handling

**Recommended:**
- ⏳ Add nginx reverse proxy for load balancing
- ⏳ Enable gzip compression in nginx
- ⏳ Set up CDN for static assets
- ⏳ Connection pooling for database (when implemented)

---

### 4. Testing Checklist ⏳

**Unit Tests:**
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Benchmark tests
go test -bench=. ./benchmarks/
```

**Integration Tests:**
```bash
# Test email sending
./flowa examples/test_email.flowa

# Test authentication
./flowa examples/test_auth.flowa

# Test JWT
./flowa examples/test_jwt.flowa
```

**Load Testing:**
```bash
# Install wrk or hey
brew install wrk

# Load test endpoint (adjust URL and duration)
wrk -t4 -c100 -d30s http://localhost:8080/api/endpoint
```

**Manual Tests:**
- [ ] Test all HTTP endpoints
- [ ] Test path parameter extraction
- [ ] Test query parameter parsing
- [ ] Test request body handling
- [ ] Test all response types (json/text/html/redirect)
- [ ] Test error handling
- [ ] Test concurrent requests
- [ ] Test WebSocket connections (if used)

---

### 5. Monitoring & Logging 📋

**Add Logging:**
```flowa
# Example logging in handlers
func api_handler(req){
    print("[" + time.now() + "] " + req["method"] + " " + req["path"] + " - IP: " + req["ip"])
    
    # Your handler logic here
    result = process_request(req)
    
    print("[" + time.now() + "] Response: 200")
    return response.json(result, 200)
}
```

**Recommended Monitoring:**
- Application logs (stdout/stderr)
- HTTP request/response logs
- Error rates and types
- Response times (p50, p95, p99)
- Memory usage
- CPU usage
- Active WebSocket connections (if used)

**Tools:**
- Prometheus for metrics
- Grafana for dashboards
- ELK stack for logging
- Sentry for error tracking

---

### 6. Deployment Options

#### Option A: Standalone Server

```bash
# Build binary
go build -o flowa cmd/flowa/main.go

# Run directly
./flowa your_app.flowa

# Or with systemd service
sudo systemctl start flowa
```

#### Option B: Docker Container

**Create Dockerfile:**
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o flowa cmd/flowa/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/flowa .
COPY --from=builder /app/your_app.flowa .
COPY --from=builder /app/.env .
CMD ["./flowa", "your_app.flowa"]
```

```bash
# Build image
docker build -t flowa-app .

# Run container
docker run -p 8080:8080 --env-file .env flowa-app
```

#### Option C: Docker Compose

**docker-compose.yml:**
```yaml
version: '3.8'
services:
  flowa-app:
    build: .
    ports:
      - "8080:8080"
    env_file:
      - .env
    restart: unless-stopped
    volumes:
      - ./logs:/var/log/flowa
```

```bash
docker-compose up -d
```

---

### 7. Production Server Setup

**System Requirements:**
- Linux server (Ubuntu 20.04+ recommended)
- 1GB RAM minimum (2GB+ recommended)
- 1 CPU core minimum (2+ recommended)
- 10GB disk space

**Install Dependencies:**
```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Go (if building on server)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install supervisord or systemd for process management
sudo apt install supervisor -y
```

**Systemd Service:**
```ini
[Unit]
Description=Flowa Application
After=network.target

[Service]
Type=simple
User=flowa
WorkingDirectory=/opt/flowa
EnvironmentFile=/opt/flowa/.env
ExecStart=/opt/flowa/flowa /opt/flowa/app.flowa
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start service
sudo systemctl enable flowa
sudo systemctl start flowa
sudo systemctl status flowa
```

---

### 8. Nginx Reverse Proxy

**Install Nginx:**
```bash
sudo apt install nginx -y
```

**Configure /etc/nginx/sites-available/flowa:**
```nginx
server {
    listen 80;
    server_name yourdomain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
    
    # WebSocket support
    location /ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
    }
}
```

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/flowa /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

**SSL with Let's Encrypt:**
```bash
sudo apt install certbot python3-certbot-nginx -y
sudo certbot --nginx -d yourdomain.com
sudo systemctl reload nginx
```

---

### 9. Database Setup (When Implemented)

**PostgreSQL:**
```bash
sudo apt install postgresql postgresql-contrib -y
sudo -u postgres createdb flowa_prod
sudo -u postgres createuser flowa_user
sudo -u postgres psql -c "ALTER USER flowa_user WITH PASSWORD 'strong_password';"
```

**Add to .env:**
```
DATABASE_URL=postgres://flowa_user:strong_password@localhost/flowa_prod
```

---

### 10. Backup Strategy

**Application Code:**
```bash
# Git repository backup
git push origin production

# Binary backup
tar -czf flowa-backup-$(date +%Y%m%d).tar.gz flowa app.flowa .env
```

**Database Backup (when implemented):**
```bash
# Daily PostgreSQL backup
pg_dump flowa_prod > backup-$(date +%Y%m%d).sql
gzip backup-*.sql

# Automated backup script
#!/bin/bash
BACKUP_DIR="/var/backups/flowa"
mkdir -p $BACKUP_DIR
pg_dump flowa_prod | gzip > $BACKUP_DIR/db-$(date +%Y%m%d-%H%M).sql.gz
find $BACKUP_DIR -name "*.sql.gz" -mtime +7 -delete  # Keep 7 days
```

**Add to crontab:**
```bash
0 2 * * * /opt/flowa/backup.sh
```

---

### 11. Health Checks

**Create health endpoint in your Flowa app:**
```flowa
func health_check(req){
    return response.json({"status": "healthy", "timestamp": time.now()}, 200)
}

http.route("GET", "/health", health_check)
```

**Monitor with script:**
```bash
#!/bin/bash
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo "Application is healthy"
else
    echo "Application is down! Restarting..."
    sudo systemctl restart flowa
fi
```

---

### 12. Firewall Configuration

```bash
# Allow SSH, HTTP, HTTPS
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# Internal port (only accessible via nginx)
# Don't expose port 8080 externally
```

---

## Production Deployment Checklist

### Pre-Deployment ✅
- [x] All features implemented and tested
- [x] Build successful with no errors
- [ ] .env file configured with production values
- [ ] JWT secret generated and configured
- [ ] SMTP credentials configured
- [ ] CORS settings reviewed and secured

### Security ⚠️
- [ ] Change default JWT secret
- [ ] Use app-specific SMTP password
- [ ] Update WebSocket CORS check for production
- [ ] SSL/TLS certificate installed
- [ ] Firewall configured
- [ ] Secrets not in git repository

### Performance ✅
- [x] VM pooling enabled
- [x] Compiler pooling enabled
- [x] Integer caching enabled
- [ ] Nginx reverse proxy configured
- [ ] Load testing completed

### Monitoring 📋
- [ ] Logging implemented in handlers
- [ ] Health check endpoint created
- [ ] Monitoring tools configured
- [ ] Error tracking setup
- [ ] Alerts configured

### Backup 📋
- [ ] Code backed up in git
- [ ] Database backup script created (if applicable)
- [ ] Backup tested and verified
- [ ] Automated backup scheduled

### Testing ⏳
- [ ] All unit tests passing
- [ ] Integration tests passing
- [ ] Load tests completed
- [ ] Manual testing completed
- [ ] Error scenarios tested

### Deployment 📋
- [ ] Server provisioned
- [ ] Dependencies installed
- [ ] Application deployed
- [ ] Systemd/supervisor configured
- [ ] Nginx configured
- [ ] DNS configured
- [ ] SSL certificate installed

---

## Post-Deployment

### Verify Deployment
```bash
# Check application is running
curl https://yourdomain.com/health

# Check SSL
curl -I https://yourdomain.com

# Test endpoints
curl https://yourdomain.com/api/test

# Monitor logs
sudo journalctl -u flowa -f
```

### Monitor Performance
```bash
# Check resource usage
top
htop

# Check connections
netstat -an | grep :8080

# Check logs for errors
sudo journalctl -u flowa | grep ERROR
```

---

## Troubleshooting

### Application Won't Start
```bash
# Check logs
sudo journalctl -u flowa -n 50

# Check .env file exists
ls -la /opt/flowa/.env

# Check file permissions
sudo chown -R flowa:flowa /opt/flowa

# Test binary directly
./flowa app.flowa
```

### High Memory Usage
- Review VM pooling configuration
- Check for memory leaks in handlers
- Monitor with `top` or `htop`
- Consider increasing server resources

### Database Connection Errors (when implemented)
- Verify DATABASE_URL in .env
- Check database is running
- Verify credentials
- Check connection limits

---

## Scaling Considerations

### Horizontal Scaling
- Multiple Flowa instances behind load balancer
- Session storage in Redis (when implemented)
- Shared database
- Distributed caching

### Vertical Scaling
- Increase server resources (CPU, RAM)
- Optimize database queries
- Add caching layer
- CDN for static assets

---

## Support & Maintenance

### Regular Tasks
- Weekly: Review logs for errors
- Monthly: Security updates
- Quarterly: Performance review
- Annually: Credential rotation

### Updates
```bash
# Pull latest code
git pull origin production

# Rebuild
go build -o flowa cmd/flowa/main.go

# Restart service
sudo systemctl restart flowa
```

---

## Conclusion

Your Flowa application is ready for production deployment! Follow this guide step by step to ensure a smooth and secure deployment.

**Remember:**
- Always test in staging first
- Keep backups
- Monitor your application
- Rotate secrets regularly
- Keep dependencies updated

For additional help, refer to DOCUMENTATION.md for feature-specific details.
