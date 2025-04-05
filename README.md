# Resolyte
service-governance

## Environment Variables

Create a `.env` file in the root directory of the project and add the following fields:

```ini
# Environment type (e.g., development, production)
ENV=development

HTTPS=false

GOVERNANCE_HTTP_PORT=12001
SECRET_KEY=my-secret-key

# Redis configuration
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=myredispassword
REDIS_DATABASE_GOVERNANCE=0
```

## HTTPS Setup

If `HTTPS=true` is set in your `.env` file, you must generate TLS certificates (`cert.pem` and `key.pem`) to enable secure communication.

To generate self-signed certificates:

```bash
# run from project root
make generate_cert_and_key