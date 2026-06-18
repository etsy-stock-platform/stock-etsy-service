# stock-etsy-service

Go microservice for Etsy shop connection and product inventory synchronization.

Local service URL:

```text
http://localhost:8082
```

Health endpoint:

```text
GET /health
```

Auth-protected Etsy check endpoint:

```text
GET /etsy/auth-check
```

Local auth service URL is configured with:

```text
AUTH_SERVICE_URL=http://localhost:8081
AUTH_SERVICE_REQUEST_TIMEOUT=5s
```
