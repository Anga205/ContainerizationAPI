import os
import signal
import sys

while True:
    try:
        pid = os.fork()
    except OSError as exc:
        print(exc, file=sys.stderr)
        raise SystemExit(0)

    if pid == 0:
        while True:
            signal.pause()
