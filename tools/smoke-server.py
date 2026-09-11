#!/usr/bin/env python3
"""Exercise a real process, SQLite reopen, and SIGTERM without production access."""
import json
import os
from pathlib import Path
import socket
import secrets
import urllib.error
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
    env.pop("POCKETLINK_PUBLIC_ORIGIN", None)
    env.pop("POCKETLINK_ADMIN_PASSWORD_FILE", None)
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    password = secrets.token_urlsafe(32)
    password_file = Path(directory) / "admin-secret"
    password_file.write_text(password)
    password_file.chmod(0o600)
    cookie = credential = None
    def call(path, method="GET", data=None, admin=False, device=False):
        headers = {"Content-Type": "application/json"}
        if admin:
            headers["Origin"] = "https://console.example"
            if cookie: headers["Cookie"] = cookie
        if device: headers["Authorization"] = "Bearer " + credential
        req = urllib.request.Request(f"http://127.0.0.1:{port}/api/v1/" + path,
                                     data=json.dumps(data).encode() if data is not None else None,
                                     headers=headers, method=method)
        try:
            with opener.open(req, timeout=10) as response:
                return response.status, json.load(response), response.headers
        except urllib.error.HTTPError as error:
            return error.code, json.load(error), error.headers
    for iteration in range(4):
        if iteration >= 2:
            env["POCKETLINK_PUBLIC_ORIGIN"] = "https://console.example"
            env["POCKETLINK_ADMIN_PASSWORD_FILE"] = str(password_file)
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
                    features = json.load(response)["enabled_features"]
                    assert (features == []) if iteration < 2 else ("device_pairing" in features)
                if iteration == 2:
                    status, _, headers = call("auth/login", "POST", {"password": password}, admin=True)
                    assert status == 200
                    cookie = headers["Set-Cookie"].split(";", 1)[0]
                    status, pair, _ = call("pairings", "POST", {"name": "smoke device"}, admin=True)
                    assert status == 201
                    status, paired, _ = call("device/pair", "POST", {"code": pair["code"], "sn": "smoke-device"})
                    assert status == 201
                    credential = paired["credential"]
                    assert call("device/me", device=True)[0] == 200
                    status, room, _ = call("rooms", "POST", {"name": "smoke room"}, admin=True)
                    assert status == 201
                    room_path = "rooms/" + room["id"]
                    assert call(room_path + "/members/" + paired["device"]["id"], "PUT", admin=True)[0] == 200
                    status, message, _ = call(room_path + "/messages", "POST", {"request_id": "smoke-retry", "text": "persist me"}, admin=True)
                    assert status == 200
                    assert call(room_path + "/messages", "POST", {"request_id": "smoke-retry", "text": "persist me"}, admin=True)[1]["id"] == message["id"]
                if iteration == 3:
                    assert call("auth/me", admin=True)[0] == 401
                    assert call("device/me", device=True)[0] == 200
                    status, inbox, _ = call("device/inbox", device=True)
                    assert status == 200 and len(inbox["messages"]) == 1
                    assert inbox["messages"][0]["id"] == message["id"]
                    assert call("device/ack", "POST", {"message_id": message["id"], "state": "read"}, device=True)[0] == 200
                    assert call("device/ack", "POST", {"message_id": message["id"], "state": "received"}, device=True)[0] == 200
                    assert call("device/inbox", device=True)[1]["messages"] == []
                process.terminate()
                assert process.wait(timeout=12) == 0
            finally:
                if process.poll() is None:
                    process.kill()
                    process.wait()
print("Process startup, authentication, idempotent text, offline recovery, receipts and graceful shutdown: PASS")
