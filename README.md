# version0
resolyte

## Environment Variables

Create a `.env` file in the root directory of the project and add the following fields:

```ini
# Environment type (e.g., development, production)
ENV=development


GOVERNANCE_HTTP_PORT=12001
SECRET_KEY=my-secret-key

# Redis configuration
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=myredispassword
REDIS_DATABASE_GOVERNANCE=0
