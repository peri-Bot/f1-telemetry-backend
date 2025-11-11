# sidecar/data_forwarder.py

import threading
import logging
from flask import Flask, jsonify
import livef1 # <-- The real library

# --- Thread-Safe In-Memory Storage ---
# We will store the latest car data packet for each driver index.
# The lock ensures we don't read the data while it's being written.
latest_car_data = {}
data_lock = threading.Lock()

# --- Flask Web Server ---
# Create the Flask app
app = Flask(__name__)

# Silence the default Flask/Werkzeug logs for a cleaner terminal
log = logging.getLogger('werkzeug')
log.setLevel(logging.ERROR)

@app.route('/data', methods=['GET'])
def get_data():
    """Returns the latest telemetry data for all cars as a single JSON object."""
    with data_lock:
        # Return a copy of the data to avoid issues with concurrent access
        return jsonify(latest_car_data.copy())

def run_live_f1_client():
    """
    Connects to the F1 data stream and continuously updates the global
    'latest_car_data' dictionary with the latest packet for each car.
    """
    print("Attempting to connect to the LiveF1 data stream...")
    try:
        # Initialize the client
        client = livef1.LiveF1()
        print("Successfully connected to LiveF1 stream. Waiting for data...")

        # The get_car_data() method is a generator that yields data packets
        # as they arrive in real-time.
        for packet in client.get_car_data():
            # The packet is a dictionary. The header contains the car's index.
            # This is a unique identifier for the car during the session.
            car_index = packet.get('m_header', {}).get('m_playerCarIndex')

            if car_index is not None:
                # Safely update the shared dictionary
                with data_lock:
                    latest_car_data[car_index] = packet

    except Exception as e:
        print(f"An error occurred in the LiveF1 client thread: {e}")
        # In a real production system, you'd want more robust error handling
        # and reconnection logic here.
        with data_lock:
            latest_car_data['error'] = str(e)


if __name__ == '__main__':
    # Run the LiveF1 client in a separate thread so it doesn't
    # block the Flask web server.
    f1_thread = threading.Thread(target=run_live_f1_client)
    f1_thread.daemon = True # Allows main thread to exit even if this one is running
    f1_thread.start()
    
    # Start the Flask server on localhost, port 5000
    print("Starting Flask sidecar server on http://localhost:5000")
    app.run(host='0.0.0.0', port=5000)
