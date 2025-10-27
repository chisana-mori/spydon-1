#!/bin/bash

echo "=== Traefik Debug Script ==="
echo

# Check if Traefik is running
echo "1. Checking Traefik container status:"
docker-compose ps traefik
echo

# Check Traefik logs
echo "2. Recent Traefik logs:"
docker-compose logs --tail=20 traefik
echo

# Check if dynamic configuration is loaded
echo "3. Checking Traefik API routers:"
curl -s http://localhost:8080/api/http/routers | jq '.' 2>/dev/null || curl -s http://localhost:8080/api/http/routers
echo
echo

echo "4. Checking Traefik API services:"
curl -s http://localhost:8080/api/http/services | jq '.' 2>/dev/null || curl -s http://localhost:8080/api/http/services
echo
echo

echo "5. Checking Traefik API middlewares:"
curl -s http://localhost:8080/api/http/middlewares | jq '.' 2>/dev/null || curl -s http://localhost:8080/api/http/middlewares
echo
echo

# Check if Spydon service is accessible directly
echo "6. Testing direct access to Spydon service (port 8086):"
curl -v http://localhost:8086/ 2>&1 | head -10
echo

# Check if Traefik is routing to Spydon
echo "7. Testing Traefik routing to /spydon:"
curl -v http://localhost/spydon/ 2>&1 | head -10
echo

# Check access logs
echo "8. Recent access logs (if available):"
if [ -f "./logs/access.log" ]; then
    tail -10 ./logs/access.log
else
    echo "Access log file not found at ./logs/access.log"
    echo "Checking container logs:"
    docker-compose exec traefik cat /var/log/traefik/access.log 2>/dev/null | tail -10 || echo "No access logs found in container"
fi
echo

# Check if port 8086 is listening
echo "9. Checking if port 8086 is listening:"
netstat -tlnp | grep :8086 || ss -tlnp | grep :8086
echo

echo "=== Debug Complete ==="