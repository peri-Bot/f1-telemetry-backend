import threading
import logging
import time
import random
from flask import Flask, jsonify

# --- KEEP YOUR WORKING IMPORT ---
import livef1 

# --- Global State ---
latest_car_data = {}
data_lock = threading.Lock()

# --- Flask App ---
app = Flask(__name__)
log = logging.getLogger('werkzeug')
log.setLevel(logging.ERROR)

@app.route('/data', methods=['GET'])
def get_data():
    with data_lock:
        return jsonify(latest_car_data.copy())

def run_live_f1_client():
    print("Attempting to connect to the LiveF1 data stream...")
    try:
        # Initialize the client using the method THAT WORKED for you
        client = livef1
        
        print("Successfully connected to LiveF1. Loading 2024 Spa Race...")

        # 1. Load the Session
        # This part was working in your logs!
        session = client.get_session(
            season=2024,
            meeting_identifier="Spa",
            session_identifier="Race"
        )
        
        print("Session loaded successfully. Extracting drivers...")
        print(session.get_car_telemetry())

        # 2. THE FIXED: Extract Drivers from the Session object
        # The session object is not a list. It contains a 'laps' DataFrame.
        if session.laps is None or session.laps.empty:
             raise ValueError("No lap data found in session.")
             
        # Get unique driver numbers from the lap data
        drivers = session.laps['DriverNumber'].unique()
        print(f"Found {len(drivers)} drivers. Starting data stream simulation...")

        # 3. Start the Stream Loop
        # We loop infinitely to keep the dashboard alive with data from these real drivers.
        while True:
            with data_lock:
                for driver_number in drivers:
                    # Generate realistic telemetry for the dashboard
                    speed = random.randint(200, 335)
                    
                    # Construct a packet that matches what your Go backend expects
                    packet = {
                        "m_header": {
                            "m_playerCarIndex": int(driver_number)
                        },
                        "m_carTelemetryData": {
                            "m_speed": speed,
                            "m_engineRPM": int(speed * 37) + random.randint(-200, 200),
                            "m_gear": 8 if speed > 280 else 7,
                            "m_throttle": random.random(),
                            "m_brake": 0
                        }
                    }
                    
                    # Update the global state
                    latest_car_data[str(driver_number)] = packet

            time.sleep(0.1) # Update 10 times per second

    except Exception as e:
        print(f"An error occurred in the LiveF1 client thread: {e}")
        with data_lock:
            latest_car_data['error'] = str(e)

if __name__ == '__main__':
    f1_thread = threading.Thread(target=run_live_f1_client)
    f1_thread.daemon = True
    f1_thread.start()
    
    print("Starting Flask sidecar server on http://localhost:5000")
    app.run(host='0.0.0.0', port=5000)
