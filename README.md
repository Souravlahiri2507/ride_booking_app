🚖 Mini Ride Booking

This is a mini ride booking system built with Go (Gin), Kafka, and MongoDB.
It consists of two services:

Booking Service – Handles ride bookings and produces events.

Driver Service – Manages drivers, open jobs, and consumes/produces booking events.

📌 Requirements & Goals

Implement the two services with this behaviour:

POST /bookings — create a booking, generate booking_id, persist with ride_status="Requested", driver_id=null, produce booking.created (Topic A), return 201 with full booking JSON.

GET /bookings — list bookings (newest first).

Booking service consumes booking.accepted (Topic B) and updates ride_status="Accepted" and driver_id idempotently.

Driver service seeds drivers at startup, consumes booking.created to expose open jobs, and supports
POST /jobs/{booking_id}/accept with body { "driver_id": "d-1" }.
The accept must be atomic / first-wins (others get 409 Conflict), and it must produce booking.accepted.

📂 Repo Layout
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

⚡ Setup Steps

Assumes you have Go 1.21+, Docker & Docker Compose, and mongosh or MongoDB Compass.

From your project root:

Start infra (Zookeeper, Kafka, Mongo)

docker-compose -f deploy/docker-compose.yml up -d


Ensure Kafka advertises localhost (if you run Go services on host)
If you see consumer dial errors like lookup kafka: no such host,
edit deploy/docker-compose.yml kafka service and set:

KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9092
KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092


Then restart:

docker-compose -f deploy/docker-compose.yml down
docker-compose -f deploy/docker-compose.yml up -d


Run booking service (terminal 1)

cd booking_svc
go mod tidy
MONGO_URI="mongodb://localhost:27017" MONGO_DB="ride_booking_app" \
KAFKA_BROKERS="localhost:9092" PORT=8080 \
go run ./cmd/server


Run driver service (terminal 2)

cd driver_svc
go mod tidy
MONGO_URI="mongodb://localhost:27017" MONGO_DB="ride_booking_app" \
KAFKA_BROKERS="localhost:9092" PORT=8081 \
go run ./cmd/server

🌍 ENV Variables

Set these for both services (change PORT per service):

Variable	Description	Example
MONGO_URI	Mongo connection string	mongodb://localhost:27017
MONGO_DB	Database name (must match Compass DB)	ride_booking_app
KAFKA_BROKERS	Kafka bootstrap address	localhost:9092
PORT	Service port	booking_svc: 8080
driver_svc: 8081
📬 Sample cURLs

Create booking (produces booking.created):

curl -X POST http://localhost:8080/bookings \
 -H "Content-Type: application/json" \
 -d '{"pickuploc":{"lat":12.9,"lng":77.6},"dropoff":{"lat":12.95,"lng":77.64},"price":220}'


List bookings:

curl http://localhost:8080/bookings


List drivers:

curl http://localhost:8081/drivers


List open jobs (after booking.created consumed):

curl http://localhost:8081/jobs


Accept a job (first-wins; returns 200 on success, 409 if already taken):

curl -X POST http://localhost:8081/jobs/<booking_id>/accept \
 -H "Content-Type: application/json" \
 -d '{"driver_id":"d-1"}'

📡 How to See Messages (Kafka & Mongo)
🔹 Kafka — list topics & view messages

Get kafka container id/name:

docker-compose -f deploy/docker-compose.yml ps


List topics:

docker exec -it $(docker-compose -f deploy/docker-compose.yml ps -q kafka) \
  /opt/bitnami/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list


Consume messages (from beginning):

booking.created

docker exec -it $(docker-compose -f deploy/docker-compose.yml ps -q kafka) \
  /opt/bitnami/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 --topic booking.created --from-beginning --timeout-ms 10000


booking.accepted

docker exec -it $(docker-compose -f deploy/docker-compose.yml ps -q kafka) \
  /opt/bitnami/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 --topic booking.accepted --from-beginning --timeout-ms 10000


If --bootstrap-server localhost:9092 fails inside container, use the address the broker advertises.
(If you followed the recommended compose change it should be localhost:9092.)

You can also tail Kafka logs:

docker-compose -f deploy/docker-compose.yml logs -f kafka

📝 Assumptions

DB name: Used ride_app in examples. If you used a different DB name in Compass, set MONGO_DB accordingly.

Collections used: bookings, drivers, jobs (not booking_svc/driver_svc).
They will auto-create on insert if missing.

Kafka bootstrap address: Must be reachable from host when running go run.
Use localhost:9092. If Kafka advertises kafka:9092, host processes can't resolve it — fix with KAFKA_ADVERTISED_LISTENERS.

Driver seeding: driver_svc auto-seeds a couple of drivers at startup if collection is empty.

Atomic accept guarantee: POST /jobs/:booking_id/accept uses atomic DB update (first-wins).
First writer wins; others get 409.

No TLS/auth: Kafka & Mongo run without auth/TLS (dev only). Not for production.

Single-broker Kafka: Single-broker setup, good for local testing only.