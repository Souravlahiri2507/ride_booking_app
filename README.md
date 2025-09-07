Mini Ride Booking — README

1) Requirements & goals-
Implement the two services with this behaviour:

- POST /bookings — create a booking, generate booking_id, persist with ride_status="Requested", driver_id=null, produce booking.created (Topic A), return 201 with full booking JSON.

- GET /bookings — list bookings (newest first).

- Booking service consumes booking.accepted (Topic B) and updates ride_status="Accepted" and driver_id idempotently.

- driver_svc seeds drivers at startup, consumes booking.created to expose open jobs, and supports POST /jobs/{booking_id}/accept with body { "driver_id": "d-1" }. The accept must be atomic / first-wins (others get 409 Conflict), and it must produce booking.accepted.

3) Repo Layout -
/
├── booking_svc/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── api/
│   │   ├── domain/
│   │   ├── infrastructure/
│   │   └── utils/
│   └── go.mod
├── driver_svc/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── api/
│   │   ├── domain/
│   │   ├── infrastructure/
│   │   └── utils/
│   └── go.mod
├── deploy/
│   └── docker-compose.yml   # brings up Kafka, Zookeeper, MongoDB
└── README.md

4) Setup Steps - 
Assumes you have: Go 1.21+, Docker & Docker Compose, mongosh or MongoDB Compass.
# from your project root
# 1) Start infra (Zookeeper, Kafka, Mongo)
docker-compose -f deploy/docker-compose.yml up -d

# 2) Ensure Kafka advertises localhost (if you run Go services on host).
# If you see consumer dial errors like "lookup kafka: no such host",
# edit deploy/docker-compose.yml kafka service and set:
#   KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9092
#   KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092
# then restart:
docker-compose -f deploy/docker-compose.yml down
docker-compose -f deploy/docker-compose.yml up -d

# 3) Install deps & run booking service (terminal 1)
cd booking_svc
go mod tidy
MONGO_URI="mongodb://localhost:27017" MONGO_DB="ride_booking_app" \
KAFKA_BROKERS="localhost:9092" PORT=8080 \
go run ./cmd/server

# 4) Install deps & run driver service (terminal 2)
cd driver_svc
go mod tidy
MONGO_URI="mongodb://localhost:27017" MONGO_DB="ride_booking_app" \
KAFKA_BROKERS="localhost:9092" PORT=8081 \
go run ./cmd/server


5) ENV Variables -
Set these for both services (change PORT per service):

MONGO_URI — Mongo connection string
Example: mongodb://localhost:27017

MONGO_DB — Database name (must match Compass DB)
Example: ride_booking_app

KAFKA_BROKERS — Kafka bootstrap address (when running services on host)
Example: localhost:9092

PORT — Service port

booking_svc: 8080

driver_svc: 8081

6) All sample cURLs

Create booking (produces booking.created):
curl -X POST http://localhost:8080/bookings \
 -H "Content-Type: application/json" \
 -d '{"pickuploc":{"lat":12.9,"lng":77.6},"dropoff":{"lat":12.95,"lng":77.64},"price":220}'

List bookings:
curl http://localhost:8080/bookings

List drivers:
curl http://localhost:8081/drivers

List open jobs (driver svc — jobs created after booking.created consumed):
curl http://localhost:8081/jobs

Accept a job (first-wins; returns 200 on success, 409 if already taken):
curl -X POST http://localhost:8081/jobs/<booking_id>/accept \
 -H "Content-Type: application/json" \
 -d '{"driver_id":"d-1"}'


7) How to see messages (Kafka & Mongo)
A) Kafka — list topics & view messages (run on host; uses docker)

List topics:

# get kafka container id/name from compose
docker-compose -f deploy/docker-compose.yml ps

# inside kafka container: list topics
docker exec -it $(docker-compose -f deploy/docker-compose.yml ps -q kafka) \
  /opt/bitnami/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list


Consume messages (show from beginning; open new terminal):

# booking.created
docker exec -it $(docker-compose -f deploy/docker-compose.yml ps -q kafka) \
  /opt/bitnami/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 --topic booking.created --from-beginning --timeout-ms 10000

# booking.accepted
docker exec -it $(docker-compose -f deploy/docker-compose.yml ps -q kafka) \
  /opt/bitnami/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 --topic booking.accepted --from-beginning --timeout-ms 10000


If the --bootstrap-server localhost:9092 fails inside container, use the address the broker advertises. (If you followed the recommended compose change it should be localhost:9092.)

You can also tail Kafka logs:

docker-compose -f deploy/docker-compose.yml logs -f kafka

8) Assumptions (what I assumed in the code / setup):
- DB name: I used ride_app in examples. If you used a different DB name in Compass, set MONGO_DB to that name.

- Collections used by code: bookings, drivers, jobs (not booking_svc/driver_svc). Insert these names or let inserts auto-create them.

- Kafka bootstrap address: When running services on your host with go run, KAFKA_BROKERS must be reachable from host (use localhost:9092). If Kafka advertises kafka:9092 (container hostname), host processes can't resolve it — change KAFKA_ADVERTISED_LISTENERS to localhost:9092 in docker-compose or run the services inside Docker.

- Driver seeding: driver_svc auto-seeds a couple of drivers at startup if drivers collection is empty. You can also insert drivers manually.

- Atomic accept guarantee: POST /jobs/:booking_id/accept uses atomic DB update on the jobs collection (claim only if taken: false) — first writer wins; others get 409.

- No TLS/auth: Kafka and Mongo run without auth/TLS (dev only). Don’t use this configuration in production.

- Single-broker dev Kafka: This is a simple single-broker setup adequate for local testing only.