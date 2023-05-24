# README #
For go application utilities.

## Feature : ##
- Cache
  - [Redis](cache/implementations/redis)
    - Go-Redis
- HTTP
  - [Resty](http/implementations/resty)
- Logger
  - [Logrus](logger/implementations/logrus)
- Persistent (Database)
  - [SQL](persistent/sql/implementations)
    - Gorm
      - PostgreSQL
  - No SQL
    - [Mongo](persistent/nosql/mongo)
      - Go Mongo
- [Shared](shared)
- Uploader
  - [Minio](uploader/implementations/minio)
    - Minio-Go
- Monitoring (see : [learn about monitor](monitor/README.md))
  - [Sentry](monitor/implementations/sentry)
    - Sentry-Go