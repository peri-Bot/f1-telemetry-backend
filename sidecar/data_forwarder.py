import sys
import os
import time
import random
import logging
from concurrent import futures

import grpc
from grpc_health.v1 import health, health_pb2, health_pb2_grpc

# Add parent directory so we can import generated proto stubs
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from proto import telemetry_pb2, telemetry_pb2_grpc

import livef1

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger(__name__)


class TelemetryServicer(telemetry_pb2_grpc.TelemetryServiceServicer):
    """gRPC server that streams F1 telemetry data."""

    def __init__(self):
        self.drivers = []
        self._load_session()

    def _load_session(self):
        """Load a historical F1 session to get driver numbers."""
        logger.info("Loading LiveF1 session data...")
        try:
            session = livef1.get_session(
                season=2024,
                meeting_identifier="Spa",
                session_identifier="Race",
            )

            if session.laps is None or session.laps.empty:
                raise ValueError("No lap data found in session.")

            self.drivers = list(session.laps["DriverNumber"].unique())
            logger.info("Loaded %d drivers from 2024 Spa Race", len(self.drivers))
        except Exception as e:
            logger.error("Failed to load session: %s", e)
            # Fallback to a set of known 2024 driver numbers
            self.drivers = [1, 4, 10, 11, 14, 16, 18, 20, 22, 23, 24, 27, 31, 44, 55, 63, 77, 81]
            logger.info("Using fallback driver list (%d drivers)", len(self.drivers))

    def StreamTelemetry(self, request, context):
        """Server-streaming RPC: pushes TelemetryBatch at ~10 Hz."""
        logger.info("Client connected — starting telemetry stream")

        while context.is_active():
            updates = []
            for driver_number in self.drivers:
                speed = random.randint(200, 335)
                update = telemetry_pb2.DriverUpdate(
                    driver_number=str(driver_number),
                    car_data=telemetry_pb2.CarData(
                        speed=speed,
                        rpm=int(speed * 37) + random.randint(-200, 200),
                        gear=8 if speed > 280 else 7,
                        throttle=random.random(),
                        brake=0.0,
                        drs=speed > 300,
                    ),
                    position=telemetry_pb2.Position(
                        x=random.uniform(-1000, 1000),
                        y=random.uniform(-1000, 1000),
                        z=random.uniform(0, 50),
                        status="OnTrack",
                    ),
                )
                updates.append(update)

            batch = telemetry_pb2.TelemetryBatch(
                drivers=updates,
                timestamp=time.strftime("%Y-%m-%dT%H:%M:%S.000Z", time.gmtime()),
            )
            yield batch
            time.sleep(0.1)  # 10 Hz

        logger.info("Client disconnected — stream ended")


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))

    # Register the telemetry service
    telemetry_pb2_grpc.add_TelemetryServiceServicer_to_server(
        TelemetryServicer(), server
    )

    # Register the gRPC health service
    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)
    health_servicer.set(
        "telemetry.TelemetryService",
        health_pb2.HealthCheckResponse.SERVING,
    )

    port = os.environ.get("GRPC_PORT", "50051")
    server.add_insecure_port(f"0.0.0.0:{port}")
    logger.info("gRPC sidecar server starting on port %s", port)
    server.start()
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
