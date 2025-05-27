# Resolyte
service-governance

## Environment Variables

Create a `.env` file in the root directory of the project and add the following fields:

```ini
# Environment type (e.g., development, production)
ENV=development
USE_HTTPS=false

GOVERNANCE_HTTP_PORT=12001
GOVERNANCE_GRPC_PORT=9090
SECRET_KEY=my-secret-key

# Redis configuration
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=myredispassword
REDIS_DATABASE_GOVERNANCE=0

# Mongo configuration
MONGO_HOST=mongo
MONGO_PORT=27017
MONGODB_INITDB_ROOT_USERNAME=admin
MONGODB_INITDB_ROOT_PASSWORD=mymongopassword

CORS_ORIGIN=http://127.0.0.1:5500
```

## HTTPS Setup

If `HTTPS=true` is set in your `.env` file, you must generate TLS certificates (`cert.pem` and `key.pem`) to enable secure communication.

To generate self-signed certificates:

```bash
# run from project root
make generate_cert
```

NOTE: self-signed certs won't be accepted by browsers unless you explicitly tell them to