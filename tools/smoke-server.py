#!/usr/bin/env python3
"""Exercise a real process, SQLite reopen, and SIGTERM without production access."""
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import time
import urllib.request

binary = str(Path(sys.argv[1]).resolve())
with tempfile.TemporaryDirectory(prefix="pocketlink-smoke-") as directory:
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        port = sock.getsockname()[1]
    env = dict(os.environ, POCKETLINK_LISTEN=f"127.0.0.1:{port}",
               POCKETLINK_DATABASE=str(Path(directory) / "relay.db"))
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    for iteration in range(2):
        with open(Path(directory) / f"run-{iteration}.log", "w+") as log:
            process = subprocess.Popen([binary], env=env, stdout=log, stderr=log)
            try:
                deadline = time.monotonic() + 15
                while True:
                    if process.poll() is not None:
                        log.seek(0)
                        raise RuntimeError(log.read())
                    try:
                        with opener.open(f"http://127.0.0.1:{port}/health/ready", timeout=1) as response:
                            assert json.load(response)["status"] == "ready"
                        break
                    except OSError:
                        if time.monotonic() >= deadline:
                            raise
                        time.sleep(0.1)
                subprocess.run([binary, "healthcheck"], env=env, check=True)
                with opener.open(f"http://127.0.0.1:{port}/api/v1/capabilities", timeout=1) as response:
                    assert json.load(response)["enabled_features"] == []
                process.terminate()
                assert process.wait(timeout=12) == 0
            finally:
                if process.poll() is None:
                    process.kill()
                    process.wait()
print("Process startup, database reopen, healthcheck and graceful shutdown: PASS")
